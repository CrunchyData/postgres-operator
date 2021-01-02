package job

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	clusteroperator "github.com/crunchydata/postgres-operator/internal/operator/cluster"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
)

// handleRepoSyncUpdate is responsible for handling updates to repo sync jobs
func (c *Controller) handleRepoSyncUpdate(job *apiv1.Job) error {

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

	log.Debugf("jobController onUpdate clone step 1 job case")
	log.Debugf("clone step 1 job status=%d", job.Status.Succeeded)

	namespace := job.ObjectMeta.Namespace
	sourceClusterName := job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_SOURCE_CLUSTER_NAME]
	targetClusterName := job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_TARGET_CLUSTER_NAME]
	workflowID := job.ObjectMeta.Labels[config.LABEL_WORKFLOW_ID]

	log.Debugf("workflow to update is %s", workflowID)

	// first, make sure the Pgtask resource knows that the job is complete,
	// which is using this legacy bit of code
	if err := util.Patch(c.Client.CrunchydataV1().RESTClient(), patchURL, crv1.JobCompletedStatus, patchResource, job.Name, namespace); err != nil {
		log.Error(err)
		// we can continue on, even if this fails...
	}

	// next, update the workflow to indicate that step 1 is complete
	clusteroperator.UpdateCloneWorkflow(c.Client, namespace, workflowID, crv1.PgtaskWorkflowCloneRestoreBackup)

	// determine the storage source (e.g. local or s3) to use for the restore based on the storage
	// source utilized for the backrest repo sync job
	var storageSource string
	for _, envVar := range job.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "BACKREST_STORAGE_SOURCE" {
			storageSource = envVar.Value
		}
	}

	// now, set up a new pgtask that will allow us to perform the restore
	cloneTask := util.CloneTask{
		BackrestPVCSize:       job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_BACKREST_PVC_SIZE],
		BackrestStorageSource: storageSource,
		EnableMetrics:         job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_ENABLE_METRICS] == "true",
		PGOUser:               job.ObjectMeta.Labels[config.LABEL_PGOUSER],
		PVCSize:               job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_PVC_SIZE],
		SourceClusterName:     sourceClusterName,
		TargetClusterName:     targetClusterName,
		TaskStepLabel:         config.LABEL_PGO_CLONE_STEP_2,
		TaskType:              crv1.PgtaskCloneStep2,
		Timestamp:             time.Now(),
		WorkflowID:            workflowID,
	}

	task := cloneTask.Create()

	// finally, create the pgtask!
	if _, err := c.Client.CrunchydataV1().Pgtasks(namespace).Create(task); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not create pgtask for step 2: %s", err.Error())
		clusteroperator.PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return err
	}

	// ...we really shouldn't need a return here the way this function is
	// constructed...but just in case
	return nil
}
