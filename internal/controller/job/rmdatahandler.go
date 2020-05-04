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

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
)

const (
	deleteRMDataJobMaxTries = 10
	deleteRMDataJobDuration = 5
)

// handleRMDataUpdate is responsible for handling updates to rmdata jobs
func (c *Controller) handleRMDataUpdate(job *apiv1.Job) error {

	labels := job.GetObjectMeta().GetLabels()

	// return if job wasn't successful
	if !isJobSuccessful(job) {
		log.Debugf("jobController onUpdate rmdata job %s was unsuccessful and will be ignored",
			job.Name)
		return nil
	}

	log.Debugf("jobController onUpdate rmdata job succeeded")

	publishDeleteClusterComplete(labels[config.LABEL_PG_CLUSTER],
		job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER],
		job.ObjectMeta.Labels[config.LABEL_PGOUSER],
		job.ObjectMeta.Namespace)

	clusterName := labels[config.LABEL_PG_CLUSTER]

	if err := kubeapi.DeleteJob(c.JobClientset, job.Name, job.Namespace); err != nil {
		log.Error(err)
	}

	removed := false
	for i := 0; i < deleteRMDataJobMaxTries; i++ {
		log.Debugf("sleeping while job %s is removed cleanly", job.Name)
		time.Sleep(time.Second * time.Duration(deleteRMDataJobDuration))
		_, found := kubeapi.GetJob(c.JobClientset, job.Name, job.Namespace)
		if !found {
			removed = true
			break
		}
	}

	if !removed {
		return fmt.Errorf("could not remove Job %s for some reason after max tries", job.Name)
	}

	//if a user has specified --archive for a cluster then
	// an xlog PVC will be present and can be removed
	pvcName := clusterName + "-xlog"
	if err := pvc.DeleteIfExists(c.JobClientset, pvcName, job.Namespace); err != nil {
		log.Error(err)
		return err
	}

	//delete any completed jobs for this cluster as a cleanup
	jobList, err := kubeapi.GetJobs(c.JobClientset, config.LABEL_PG_CLUSTER+"="+clusterName, job.Namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, j := range jobList.Items {
		if j.Status.Succeeded > 0 {
			log.Debugf("removing Job %s since it was completed", job.Name)
			if err := kubeapi.DeleteJob(c.JobClientset, j.Name, job.Namespace); err != nil {
				log.Error(err)
				return err
			}

		}
	}

	return nil
}
