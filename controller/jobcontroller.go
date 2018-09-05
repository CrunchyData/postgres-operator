package controller

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	apiv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// JobController holds the connections for the controller
type JobController struct {
	JobClient    *rest.RESTClient
	JobClientset *kubernetes.Clientset
	Namespace    string
}

// Run starts an pod resource controller
func (c *JobController) Run(ctx context.Context) error {

	_, err := c.watchJobs(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for job resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchJobs is the event loop for job resources
func (c *JobController) watchJobs(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.JobClientset.BatchV1().RESTClient(),
		"jobs",
		c.Namespace,
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

	go controller.Run(ctx.Done())
	return controller, nil
}

// onAdd is called when a pgcluster is added
func (c *JobController) onAdd(obj interface{}) {
}

// onUpdate is called when a pgcluster is updated
func (c *JobController) onUpdate(oldObj, newObj interface{}) {
	job := newObj.(*apiv1.Job)
	log.Debugf("[JobCONTROLLER] OnUpdate %s active=%d succeeded=%d conditions=[%v]", job.ObjectMeta.SelfLink, job.Status.Active, job.Status.Succeeded, job.Status.Conditions)
	//label is "pgrmdata" and Status of Succeeded
	labels := job.GetObjectMeta().GetLabels()
	if job.Status.Succeeded > 0 && labels[util.LABEL_RMDATA] != "" {
		log.Debugf("rmdata job labels=[%v]", labels)
		log.Debugf("got a pgrmdata job status=%d", job.Status.Succeeded)
		//remove the pvc referenced by that job
		log.Debugf("deleting pvc " + labels["claimName"])
		err := pvc.Delete(c.JobClientset, labels["claimName"], c.Namespace)
		if err != nil {
			log.Error(err)
		}

		//delete the pgtask to cleanup
		log.Debugf("deleting pgtask for rmdata job name is %s\n", job.ObjectMeta.Name)
		kubeapi.Deletepgtasks(c.JobClient, util.LABEL_RMDATA+"=true", c.Namespace)
		kubeapi.DeleteJobs(c.JobClientset, util.LABEL_PG_CLUSTER+"="+job.ObjectMeta.Labels[util.LABEL_PG_CLUSTER], c.Namespace)

	} else if job.Status.Succeeded > 0 && labels[util.LABEL_PGBACKUP] != "" {
		log.Debugf("got a pgbackup job status=%d", job.Status.Succeeded)
		log.Debugf("update the status to completed here for pgbackup %s\n ", labels[util.LABEL_PG_DATABASE])
		dbname := job.ObjectMeta.Labels[util.LABEL_PG_DATABASE]

		err := util.Patch(c.JobClient, "/spec/backupstatus", crv1.UpgradeCompletedStatus, "pgbackups", dbname, c.Namespace)

		if err != nil {
			log.Error("error in patching pgbackup " + labels["pg-database"] + err.Error())
		}

	} else if job.Status.Succeeded > 0 && labels[util.LABEL_BACKREST] != "" {
		log.Debugf("got a backrest job status=%d", job.Status.Succeeded)
		log.Debugf("update the status to completed here for backrest %s\n ", labels[util.LABEL_PG_DATABASE])
		err := util.Patch(c.JobClient, "/spec/backreststatus", crv1.UpgradeCompletedStatus, "pgtasks", job.ObjectMeta.SelfLink, c.Namespace)
		if err != nil {
			log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		}

	}
}

// onDelete is called when a pgcluster is deleted
func (c *JobController) onDelete(obj interface{}) {
	job := obj.(*apiv1.Job)
	log.Debugf("[JobCONTROLLER] OnDelete %s", job.ObjectMeta.SelfLink)
}
