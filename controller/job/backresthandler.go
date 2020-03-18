package job

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
)

// backrestUpdateHandler is responsible for handling updates to backrest jobs
func (c *Controller) handleBackrestUpdate(job *apiv1.Job) error {

	// return if job wasn't successful
	if !isJobSuccessful(job) {
		log.Debugf("jobController onUpdate job %s was unsuccessful and will be ignored",
			job.Name)
		return nil
	}

	// return if job is being deleted
	if isJobInForegroundDeletion(job) {
		log.Debugf("jobController onUpdate job %s is being deleted and will be ignored",
			job.Name)
		return nil
	}

	labels := job.GetObjectMeta().GetLabels()

	// Route the backrest job update to the appropriate function depending on the type of
	// job.  Please note that thee LABE_PGO_CLONE_STEP_2 label represents a special form of
	// pgBackRest restore that is utilized as part of the clone process.  Since jobs with
	// the LABEL_PGO_CLONE_STEP_2 also inlcude the LABEL_BACKREST_RESTORE label, it is
	// necessary to first check for the presence of the LABEL_PGO_CLONE_STEP_2 prior to the
	// LABEL_BACKREST_RESTORE label to determine if the restore is part of and ongoing clone.
	switch {
	case labels[config.LABEL_BACKREST_COMMAND] == "backup":
		c.handleBackrestBackupUpdate(job)
	case labels[config.LABEL_PGO_CLONE_STEP_2] == "true":
		c.handleCloneBackrestRestoreUpdate(job)
	case labels[config.LABEL_BACKREST_RESTORE] == "true":
		c.handleBackrestRestoreUpdate(job)
	case labels[config.LABEL_BACKREST_COMMAND] == crv1.PgtaskBackrestStanzaCreate:
		c.handleBackrestStanzaCreateUpdate(job)
	}

	return nil
}

// handleBackrestRestoreUpdate is responsible for handling updates to backrest restore,
// (specifically those that have not been submitted as part of the clone process)
func (c *Controller) handleBackrestRestoreUpdate(job *apiv1.Job) error {

	labels := job.GetObjectMeta().GetLabels()
	log.Debugf("jobController onUpdate backrest restore job case")
	log.Debugf("got a backrest restore job status=%d", job.Status.Succeeded)
	log.Debugf("set status to restore job completed  for %s", labels[config.LABEL_PG_CLUSTER])
	log.Debugf("workflow to update is %s", labels[crv1.PgtaskWorkflowID])
	if err := util.Patch(c.JobClient, patchURL, crv1.JobCompletedStatus, patchResource, job.Name,
		job.ObjectMeta.Namespace); err != nil {
		log.Error("error in patching pgtask " + labels[config.LABEL_JOB_NAME] + err.Error())
	}

	backrestoperator.UpdateRestoreWorkflow(c.JobClient, c.JobClientset, labels[config.LABEL_PG_CLUSTER],
		crv1.PgtaskWorkflowBackrestRestorePVCCreatedStatus, job.ObjectMeta.Namespace, labels[crv1.PgtaskWorkflowID],
		labels[config.LABEL_BACKREST_RESTORE_TO_PVC], job.Spec.Template.Spec.Affinity)
	publishRestoreComplete(labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], job.ObjectMeta.Labels[config.LABEL_PGOUSER], job.ObjectMeta.Namespace)

	return nil
}

// handleBackrestRestoreUpdate is responsible for handling updates to backrest restore jobs that
// have been submitted in order to clone a cluster
func (c *Controller) handleCloneBackrestRestoreUpdate(job *apiv1.Job) error {

	log.Debugf("jobController onUpdate clone step 2 job case")
	log.Debugf("clone step 2 job status=%d", job.Status.Succeeded)

	if job.Status.Succeeded == 1 {
		namespace := job.ObjectMeta.Namespace
		sourceClusterName := job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_SOURCE_CLUSTER_NAME]
		targetClusterName := job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_TARGET_CLUSTER_NAME]
		workflowID := job.ObjectMeta.Labels[config.LABEL_WORKFLOW_ID]

		log.Debugf("workflow to update is %s", workflowID)

		// first, make sure the Pgtask resource knows that the job is complete,
		// which is using this legacy bit of code
		if err := util.Patch(c.JobClient, patchURL, crv1.JobCompletedStatus, patchResource, job.Name, namespace); err != nil {
			log.Error(err)
			// we can continue on, even if this fails...
		}

		// next, update the workflow to indicate that step 2 is complete
		clusteroperator.UpdateCloneWorkflow(c.JobClient, namespace, workflowID, crv1.PgtaskWorkflowCloneClusterCreate)

		// alright, we can move on the step 3 which is the final step, where we
		// create the cluster
		cloneTask := util.CloneTask{
			BackrestPVCSize:   job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_BACKREST_PVC_SIZE],
			PGOUser:           job.ObjectMeta.Labels[config.LABEL_PGOUSER],
			PVCSize:           job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_PVC_SIZE],
			SourceClusterName: sourceClusterName,
			TargetClusterName: targetClusterName,
			TaskStepLabel:     config.LABEL_PGO_CLONE_STEP_3,
			TaskType:          crv1.PgtaskCloneStep3,
			Timestamp:         time.Now(),
			WorkflowID:        workflowID,
		}

		task := cloneTask.Create()

		// create the pgtask!
		if err := kubeapi.Createpgtask(c.JobClient, task, namespace); err != nil {
			log.Error(err)
			errorMessage := fmt.Sprintf("Could not create pgtask for step 3: %s", err.Error())
			clusteroperator.PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		}
	}

	return nil
}

// handleBackrestRestoreUpdate is responsible for handling updates to backrest backup jobs
func (c *Controller) handleBackrestBackupUpdate(job *apiv1.Job) error {

	labels := job.GetObjectMeta().GetLabels()
	log.Debugf("jobController onUpdate backrest job case")
	log.Debugf("got a backrest job status=%d", job.Status.Succeeded)
	log.Debugf("update the status to completed here for backrest %s job %s", labels[config.LABEL_PG_CLUSTER], job.Name)
	err := util.Patch(c.JobClient, patchURL, crv1.JobCompletedStatus, patchResource, job.Name, job.ObjectMeta.Namespace)
	if err != nil {
		log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
	}
	publishBackupComplete(labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], job.ObjectMeta.Labels[config.LABEL_PGOUSER], "pgbackrest", job.ObjectMeta.Namespace, "")

	// If the completed backup was a cluster bootstrap backup, then mark the cluster as initialized
	// and initiate the creation of any replicas.  Otherwise if the completed backup was taken as
	// the result of a failover, then proceed with tremove the "primary_on_role_change" tag.
	if labels[config.LABEL_PGHA_BACKUP_TYPE] == crv1.BackupTypeBootstrap {
		log.Debugf("jobController onUpdate initial backup complete")

		controller.SetClusterInitializedStatus(c.JobClient, labels[config.LABEL_PG_CLUSTER],
			job.ObjectMeta.Namespace)

		// now initialize the creation of any replica
		controller.InitializeReplicaCreation(c.JobClient, labels[config.LABEL_PG_CLUSTER],
			job.ObjectMeta.Namespace)

	} else if labels[config.LABEL_PGHA_BACKUP_TYPE] == crv1.BackupTypeFailover {
		err := clusteroperator.RemovePrimaryOnRoleChangeTag(c.JobClientset, c.JobConfig,
			labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Namespace)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// handleBackrestRestoreUpdate is responsible for handling updates to backrest stanza create jobs
func (c *Controller) handleBackrestStanzaCreateUpdate(job *apiv1.Job) error {

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

		cluster := crv1.Pgcluster{}
		if _, err := kubeapi.Getpgcluster(c.JobClient, &cluster,
			clusterName, namespace); err != nil {
			return err
		}
		// If the cluster is a standby cluster, then no need to proceed with backup creation.
		// Instead the cluster can be set to initialized following creation of the stanza.
		if cluster.Spec.Standby {
			log.Debugf("job Controller: standby cluster %s will now be set to an initialized "+
				"status", clusterName)
			controller.SetClusterInitializedStatus(c.JobClient, clusterName, namespace)
			return nil
		}

		backrestoperator.CreateInitialBackup(c.JobClient, job.ObjectMeta.Namespace,
			clusterName, backrestRepoPodName)

	}
	return nil
}
