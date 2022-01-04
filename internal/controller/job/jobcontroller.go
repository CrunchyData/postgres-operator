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
	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the connections for the controller
type Controller struct {
	Client   *kubeapi.Client
	Informer batchinformers.JobInformer
}

// onAdd is called when a postgresql operator job is created and an associated add event is
// generated
func (c *Controller) onAdd(obj interface{}) {
	job := obj.(*apiv1.Job)
	labels := job.GetObjectMeta().GetLabels()

	// only process jobs with with vendor=crunchydata label
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		return
	}

	log.Debugf("Job Controller: onAdd ns=%s jobName=%s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
}

// onUpdate is called when a postgresql operator job is created and an associated update event is
// generated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	var err error
	job := newObj.(*apiv1.Job)
	labels := job.GetObjectMeta().GetLabels()

	// only process jobs with with vendor=crunchydata label
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		return
	}

	log.Debugf("[Job Controller] onUpdate ns=%s %s active=%d succeeded=%d conditions=[%v]",
		job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink, job.Status.Active, job.Status.Succeeded,
		job.Status.Conditions)

	labelExists := func(k string) bool { _, ok := labels[k]; return ok }
	// determine determine which handler to route the update event to
	switch {
	case labels[config.LABEL_RMDATA] == "true":
		err = c.handleRMDataUpdate(job)
	case labels[config.LABEL_BACKREST] == "true" ||
		labels[config.LABEL_BACKREST_RESTORE] == "true":
		c.handleBackrestUpdate(job)
	case labels[config.LABEL_BACKUP_TYPE_PGDUMP] == "true":
		err = c.handlePGDumpUpdate(job)
	case labels[config.LABEL_RESTORE_TYPE_PGRESTORE] == "true":
		err = c.handlePGRestoreUpdate(job)
	case labelExists(config.LABEL_PGHA_BOOTSTRAP):
		err = c.handleBootstrapUpdate(job)
	}

	if err != nil {
		log.Error(err)
	}
}

// onDelete is called when a postgresql operator job is deleted
func (c *Controller) onDelete(obj interface{}) {
	job := obj.(*apiv1.Job)
	labels := job.GetObjectMeta().GetLabels()

	// only process jobs with with vendor=crunchydata label
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		return
	}

	log.Debugf("[Job Controller] onDelete ns=%s %s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
}

// AddJobEventHandler adds the job event handler to the job informer
func (c *Controller) AddJobEventHandler() {
	c.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	log.Debugf("Job Controller: added event handler to informer")
}
