package job

/*
Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// handlePGDumpUpdate is responsible for handling updates to pg_dump jobs
func (c *Controller) handlePGDumpUpdate(job *apiv1.Job) error {
	ctx := context.TODO()

	labels := job.GetObjectMeta().GetLabels()

	log.Debugf("jobController onUpdate pgdump job case")
	log.Debugf("pgdump job status=%d", job.Status.Succeeded)
	log.Debugf("update the status to completed here for pgdump %s", labels[config.LABEL_PG_CLUSTER])

	status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"
	if job.Status.Succeeded == 0 {
		status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
	}
	if job.Status.Failed > 0 {
		status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
	}

	// update the pgdump task status to submitted - updates task, not the job.
	dumpTask := labels[config.LABEL_PGTASK]
	patch, err := kubeapi.NewJSONPatch().Add("spec", "status")(status).Bytes()
	if err == nil {
		log.Debugf("patching task %s: %s", dumpTask, patch)
		_, err = c.Client.CrunchydataV1().Pgtasks(job.Namespace).
			Patch(ctx, dumpTask, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		return err
	}

	return nil
}

// handlePGDumpUpdate is responsible for handling updates to pg_restore jobs
func (c *Controller) handlePGRestoreUpdate(job *apiv1.Job) error {
	ctx := context.TODO()

	labels := job.GetObjectMeta().GetLabels()

	log.Debugf("jobController onUpdate pgrestore job case")
	log.Debugf("pgdump job status=%d", job.Status.Succeeded)
	log.Debugf("update the status to completed here for pgrestore %s", labels[config.LABEL_PG_CLUSTER])

	status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"

	if job.Status.Succeeded == 0 {
		status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
	}

	if job.Status.Failed > 0 {
		status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
	}

	// update the pgdump task status to submitted - updates task, not the job.
	restoreTask := labels[config.LABEL_PGTASK]
	patch, err := kubeapi.NewJSONPatch().Add("spec", "status")(status).Bytes()
	if err == nil {
		log.Debugf("patching task %s: %s", restoreTask, patch)
		_, err = c.Client.CrunchydataV1().Pgtasks(job.Namespace).
			Patch(ctx, restoreTask, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		return err
	}

	return nil
}
