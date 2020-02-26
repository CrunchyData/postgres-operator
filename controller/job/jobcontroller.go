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
	"context"
	"sync"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Controller holds the connections for the controller
type Controller struct {
	JobConfig          *rest.Config
	JobClient          *rest.RESTClient
	JobClientset       *kubernetes.Clientset
	Ctx                context.Context
	informerNsMutex    sync.Mutex
	InformerNamespaces map[string]struct{}
}

const (
	patchResource = "pgtasks"
	patchURL      = "/spec/status"
)

// Run starts an pod resource controller
func (c *Controller) Run() error {

	err := c.watchJobs(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for job resource: %v\n", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
}

// watchJobs is the event loop for job resources
func (c *Controller) watchJobs(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.JobClientset, operator.InstallationName)
	log.Debugf("jobController watching %v namespaces", nsList)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting job controller for ns [%s]", nsList[i])
		c.SetupWatch(nsList[i])

	}
	return nil
}

// onAdd is called when a postgresql operator job is created and an associated add event is
// generated
func (c *Controller) onAdd(obj interface{}) {

	job := obj.(*apiv1.Job)

	//only process jobs with with vendor=crunchydata label
	labels := job.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] == "crunchydata" {
		log.Debugf("Job Controller: onAdd ns=%s jobName=%s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
		return
	}
}

// onUpdate is called when a postgresql operator job is created and an associated update event is
// generated
func (c *Controller) onUpdate(oldObj, newObj interface{}) {

	var err error
	job := newObj.(*apiv1.Job)
	labels := job.GetObjectMeta().GetLabels()

	//only process jobs with with vendor=crunchydata label
	if labels[config.LABEL_VENDOR] == "crunchydata" {
		log.Debugf("[Job Controller] onUpdate ns=%s %s active=%d succeeded=%d conditions=[%v]",
			job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink, job.Status.Active, job.Status.Succeeded,
			job.Status.Conditions)
	}

	// determine determine which handler to route the update event to
	switch {
	case labels[config.LABEL_RMDATA] == "true":
		err = c.handleRMDataUpdate(job)
	case labels[config.LABEL_BACKREST] == "true" ||
		labels[config.LABEL_BACKREST_RESTORE] == "true":
		err = c.handleBackrestUpdate(job)
	case labels[config.LABEL_BACKUP_TYPE_PGDUMP] == "true":
		err = c.handlePGDumpUpdate(job)
	case labels[config.LABEL_RESTORE_TYPE_PGRESTORE] == "true":
		err = c.handlePGRestoreUpdate(job)
	case labels[config.LABEL_PGO_BENCHMARK] == "true":
		err = c.handleBenchmarkUpdate(job)
	case labels[config.LABEL_PGO_LOAD] == "true":
		err = c.handleLoadUpdate(job)
	case labels[config.LABEL_PGO_CLONE_STEP_1] == "true":
		err = c.handleRepoSyncUpdate(job)
	}

	if err != nil {
		log.Error(err)
	}
	return
}

// onDelete is called when a postgresql operator job is deleted
func (c *Controller) onDelete(obj interface{}) {

	job := obj.(*apiv1.Job)

	//only process jobs with with vendor=crunchydata label
	labels := job.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] == "crunchydata" {
		log.Debugf("[Job Controller] onDelete ns=%s %s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
		return
	}
}

// SetupWatch creates creates a new controller that provides event notifications when jobs are
// added, updated and deleted in the specific namespace specified.  This includes defining the
// funtions that should be called when various add, update and delete events are received.  Only
// one controller can be created per namespace to ensure duplicate events are not generated.
func (c *Controller) SetupWatch(ns string) {

	// don't create informer for namespace if one has already been created
	c.informerNsMutex.Lock()
	defer c.informerNsMutex.Unlock()
	if _, ok := c.InformerNamespaces[ns]; ok {
		return
	}
	c.InformerNamespaces[ns] = struct{}{}

	source := cache.NewListWatchFromClient(
		c.JobClientset.BatchV1().RESTClient(),
		"jobs",
		ns,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&apiv1.Job{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(c.Ctx.Done())
	log.Debugf("Job Controller: created informer for namespace %s", ns)
}
