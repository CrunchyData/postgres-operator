package controller

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	apiv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"time"
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

func (c *JobController) onAdd(obj interface{}) {
}

func (c *JobController) onUpdate(oldObj, newObj interface{}) {
	job := newObj.(*apiv1.Job)
	log.Debugf("[JobCONTROLLER] OnUpdate %s active=%d succeeded=%d conditions=[%v]", job.ObjectMeta.SelfLink, job.Status.Active, job.Status.Succeeded, job.Status.Conditions)
	var err error
	//label is "pgrmdata" and Status of Succeeded
	labels := job.GetObjectMeta().GetLabels()
	if job.Status.Succeeded > 0 && labels[util.LABEL_RMDATA] != "" {
		err = handleRmdata(job, c.JobClient, c.JobClientset, c.Namespace)
		if err != nil {
			log.Error(err)
		}
	} else if labels[util.LABEL_PGBACKUP] != "" {
		dbname := job.ObjectMeta.Labels[util.LABEL_PG_CLUSTER]
		status := crv1.JobCompletedStatus
		log.Debugf("got a pgbackup job status=%d for %s", job.Status.Succeeded, dbname)
		if job.Status.Succeeded == 0 {
			status = crv1.JobSubmittedStatus
		}
		if job.Status.Failed > 0 {
			status = crv1.JobErrorStatus
		}

		if labels[util.LABEL_BACKREST] != "true" {
			err = util.Patch(c.JobClient, "/spec/backupstatus", status, "pgbackups", dbname, c.Namespace)
			if err != nil {
				log.Error("error in patching pgbackup " + labels["pg-database"] + err.Error())
			}
		}

	} else if labels[util.LABEL_BACKREST_RESTORE] == "true" {
		log.Debugf("got a backrest restore job status=%d", job.Status.Succeeded)
		if job.Status.Succeeded == 1 {
			log.Debugf("set status to restore job completed  for %s", labels[util.LABEL_PG_CLUSTER])
			log.Debugf("workflow to update is %s", labels[crv1.PgtaskWorkflowID])
			err = util.Patch(c.JobClient, "/spec/backreststatus", crv1.JobCompletedStatus, "pgtasks", labels[util.LABEL_JOB_NAME], c.Namespace)
			if err != nil {
				log.Error("error in patching pgtask " + labels[util.LABEL_JOB_NAME] + err.Error())
			}

			backrestoperator.UpdateRestoreWorkflow(c.JobClient, c.JobClientset, labels[util.LABEL_PG_CLUSTER], crv1.PgtaskWorkflowBackrestRestorePVCCreatedStatus, c.Namespace, labels[crv1.PgtaskWorkflowID], labels[util.LABEL_BACKREST_RESTORE_TO_PVC])
		}

	} else if labels[util.LABEL_BACKUP_TYPE_PGDUMP] == "true" {
		log.Debugf("pgdump job status=%d", job.Status.Succeeded)
		log.Debugf("update the status to completed here for pgdump %s", labels[util.LABEL_PG_DATABASE])
		status := crv1.JobCompletedStatus
		if job.Status.Succeeded == 0 {
			status = crv1.JobErrorStatus
		}
		//update the pgdump task status to submitted - updates task, not the job.
		err = util.Patch(c.JobClient, "/spec/status", status, "pgtasks", job.ObjectMeta.Name, c.Namespace)

		if err != nil {
			log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		}

	} else if labels[util.LABEL_BACKREST] != "" {
		log.Debugf("got a backrest job status=%d", job.Status.Succeeded)
		log.Debugf("update the status to completed here for backrest %s", labels[util.LABEL_PG_DATABASE])
		status := crv1.JobCompletedStatus
		if job.Status.Succeeded == 0 {
			status = crv1.JobErrorStatus
		}
		err = util.Patch(c.JobClient, "/spec/backreststatus", status, "pgtasks", job.ObjectMeta.SelfLink, c.Namespace)
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

func handleRmdata(job *apiv1.Job, restClient *rest.RESTClient, clientset *kubernetes.Clientset, namespace string) error {
	var err error

	log.Debugf("got a pgrmdata job status=%d", job.Status.Succeeded)
	labels := job.GetObjectMeta().GetLabels()

	//delete the pgtask to cleanup
	log.Debugf("deleting pgtask for rmdata job name is %s", job.ObjectMeta.Name)
	err = kubeapi.Deletepgtasks(restClient, util.LABEL_RMDATA+"=true", namespace)
	if err != nil {
		return err
	}
	//	kubeapi.DeleteJobs(c.JobClientset, util.LABEL_PG_CLUSTER+"="+job.ObjectMeta.Labels[util.LABEL_PG_CLUSTER], c.Namespace)

	time.Sleep(time.Second * time.Duration(5))

	//remove the pvc referenced by that job
	log.Debugf("deleting pvc %s", labels[util.LABEL_CLAIM_NAME])
	err = pvc.Delete(clientset, labels[util.LABEL_CLAIM_NAME], namespace)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}
