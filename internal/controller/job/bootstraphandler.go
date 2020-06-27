package job

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// handleBootstrapUpdate is responsible for handling updates to bootstrap jobs that are responsible
// for bootstrapping a cluster from an existing data source
func (c *Controller) handleBootstrapUpdate(job *apiv1.Job) error {

	clusterName := job.GetLabels()[config.LABEL_PG_CLUSTER]
	namespace := job.GetNamespace()

	// return if job is being deleted
	if isJobInForegroundDeletion(job) {
		log.Debugf("jobController onUpdate job %s is being deleted and will be ignored",
			job.Name)
		return nil
	}

	cluster, err := c.Client.CrunchydataV1().Pgclusters(namespace).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// if the job has exceeded its backoff limit then simply cleanup and bootstrap resources
	if isBackoffLimitExceeded(job) {
		log.Debugf("Backoff limit exceeded for bootstrap Job %s, will now cleanup bootstrap "+
			"resources", job.Name)
		if err := c.cleanupBootstrapResources(job, cluster); err != nil {
			return err
		}
		return nil
	}

	// return if job wasn't successful
	if !isJobSuccessful(job) {
		log.Debugf("jobController onUpdate job %s was unsuccessful and will be ignored",
			job.Name)
		return nil
	}

	// If the job was successful we updated the state of the pgcluster to a "bootstrapped" status.
	// This will then trigger full initialization of the cluster.  We also cleanup any resources
	// from the bootstrap job.
	if cluster.Status.State == crv1.PgclusterStateBootstrapping {

		if err := c.cleanupBootstrapResources(job, cluster); err != nil {
			return err
		}

		patch, err := json.Marshal(map[string]interface{}{
			"status": crv1.PgclusterStatus{
				State:   crv1.PgclusterStateBootstrapped,
				Message: "Pgcluster successfully bootstrapped from an existing data source",
			},
		})
		if err == nil {
			_, err = c.Client.CrunchydataV1().Pgclusters(namespace).Patch(cluster.Name, types.MergePatchType, patch)
		}
		if err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

// cleanupBootstrapResources is responsible for cleaning up the resources from a bootstrap Job.
// This includes deleting any pgBackRest repository and service created specifically the restore
// (i.e. a repository and service not associated with a current cluster but rather the cluster
// being restored from to bootstrap the cluster).
func (c *Controller) cleanupBootstrapResources(job *apiv1.Job, cluster *crv1.Pgcluster) error {

	namespace := job.GetNamespace()
	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom

	repoName := fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName)
	repoDeployment, err := c.Client.AppsV1().Deployments(namespace).Get(repoName,
		metav1.GetOptions{})
	if err != nil {
		return err
	}

	if _, ok := repoDeployment.GetLabels()[config.LABEL_PGHA_BOOTSTRAP]; ok {
		// now delete the service for the bootstrap repo
		if err := c.Client.CoreV1().Services(namespace).Delete(
			fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName),
			&metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
			return err
		}

		// and finally delete the bootstrap repo deployment
		if err := c.Client.AppsV1().Deployments(namespace).Delete(
			fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName),
			&metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
