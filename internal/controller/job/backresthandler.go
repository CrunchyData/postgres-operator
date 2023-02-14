package job

/*
Copyright 2017 - 2023 Crunchy Data Solutions, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
)

// backrestUpdateHandler is responsible for handling updates to backrest jobs
func (c *Controller) handleBackrestUpdate(job *apiv1.Job) {
	// return if job wasn't successful
	if !isJobSuccessful(job) {
		log.Debugf("jobController onUpdate job %s was unsuccessful and will be ignored",
			job.Name)
		return
	}

	// return if job is being deleted
	if isJobInForegroundDeletion(job) {
		log.Debugf("jobController onUpdate job %s is being deleted and will be ignored",
			job.Name)
		return
	}

	labels := job.GetObjectMeta().GetLabels()

	switch {
	case labels[config.LABEL_BACKREST_COMMAND] == "backup":
		_ = c.handleBackrestBackupUpdate(job)
	case labels[config.LABEL_BACKREST_COMMAND] == crv1.PgtaskBackrestStanzaCreate:
		_ = c.handleBackrestStanzaCreateUpdate(job)
	}
}

// handleBackrestRestoreUpdate is responsible for handling updates to backrest backup jobs
func (c *Controller) handleBackrestBackupUpdate(job *apiv1.Job) error {
	ctx := context.TODO()

	labels := job.GetObjectMeta().GetLabels()

	log.Debugf("jobController onUpdate backrest job case")
	log.Debugf("got a backrest job status=%d", job.Status.Succeeded)
	log.Debugf("update the status to completed here for backrest %s job %s", labels[config.LABEL_PG_CLUSTER], job.Name)

	patch, err := kubeapi.NewJSONPatch().Add("spec", "status")(crv1.JobCompletedStatus).Bytes()
	if err == nil {
		log.Debugf("patching task %s: %s", job.Name, patch)
		_, err = c.Client.CrunchydataV1().Pgtasks(job.Namespace).
			Patch(ctx, job.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Errorf("error in patching pgtask %s: %s", job.ObjectMeta.SelfLink, err.Error())
	}
	publishBackupComplete(labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Labels[config.LABEL_PGOUSER], "pgbackrest", job.ObjectMeta.Namespace, "")

	// If the completed backup was a cluster bootstrap backup, then mark the cluster as initialized
	// and initiate the creation of any replicas.  Otherwise if the completed backup was taken as
	// the result of a failover, then proceed with tremove the "primary_on_role_change" tag.
	if labels[config.LABEL_PGHA_BACKUP_TYPE] == crv1.BackupTypeBootstrap {
		log.Debugf("jobController onUpdate initial backup complete")

		if err := controller.SetClusterInitializedStatus(c.Client, labels[config.LABEL_PG_CLUSTER],
			job.ObjectMeta.Namespace); err != nil {
			log.Error(err)
			return err
		}

		// now initialize the creation of any replicas
		if err := controller.InitializeReplicaCreation(c.Client, labels[config.LABEL_PG_CLUSTER],
			job.ObjectMeta.Namespace); err != nil {
			log.Error(err)
			return err
		}

	} else if labels[config.LABEL_PGHA_BACKUP_TYPE] == crv1.BackupTypeFailover {
		if err := operator.RemovePrimaryOnRoleChangeTag(c.Client, c.Client.Config,
			labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Namespace); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// handleBackrestStanzaCreateUpdate is responsible for handling updates to
// backrest stanza create jobs
func (c *Controller) handleBackrestStanzaCreateUpdate(job *apiv1.Job) error {
	ctx := context.TODO()

	labels := job.GetObjectMeta().GetLabels()
	log.Debugf("jobController onUpdate backrest stanza-create job case")

	// grab the cluster name and namespace for use in various places below
	clusterName := labels[config.LABEL_PG_CLUSTER]
	namespace := job.Namespace

	if job.Status.Succeeded == 1 {
		log.Debugf("backrest stanza successfully created for cluster %s", clusterName)
		log.Debugf("proceeding with the initial full backup for cluster %s as needed for replica creation",
			clusterName)

		var backrestRepoPodName string
		for _, cont := range job.Spec.Template.Spec.Containers {
			for _, envVar := range cont.Env {
				if envVar.Name == "PODNAME" {
					backrestRepoPodName = envVar.Value
					log.Debugf("the backrest repo pod for the initial backup of cluster %s is %s",
						clusterName, backrestRepoPodName)
				}
			}
		}

		cluster, err := c.Client.CrunchydataV1().Pgclusters(namespace).Get(ctx, clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		// If the cluster is a standby cluster, then no need to proceed with backup creation.
		// Instead the cluster can be set to initialized following creation of the stanza.
		if cluster.Spec.Standby {
			log.Debugf("job Controller: standby cluster %s will now be set to an initialized "+
				"status", clusterName)
			if err := controller.SetClusterInitializedStatus(c.Client, clusterName,
				namespace); err != nil {
				log.Error(err)
				return err
			}

			// now initialize the creation of any replica
			if err := controller.InitializeReplicaCreation(c.Client, clusterName,
				namespace); err != nil {
				log.Error(err)
				return err
			}
			return nil
		}

		// clean any backup resources that might already be present, e.g. when restoring and these
		// resources might already exist from initial creation of the cluster
		if err := backrest.CleanBackupResources(c.Client, job.ObjectMeta.Namespace,
			clusterName); err != nil {
			log.Error(err)
			return err
		}

		if _, err := backrest.CreateInitialBackup(c.Client, job.ObjectMeta.Namespace,
			clusterName, backrestRepoPodName); err != nil {
			log.Error(err)
			return err
		}

		// now that the initial backup has been initiated, proceed with deleting the stanza-create
		// pgtask and associated Job.  This will ensure any subsequent updates to the stanza-create
		// Job do not trigger more initial backup Jobs.
		if err := backrest.CleanStanzaCreateResources(namespace, clusterName, c.Client); err != nil {
			log.Error(err)
			return err
		}

	}
	return nil
}
