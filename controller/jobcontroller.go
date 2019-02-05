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
	job := obj.(*apiv1.Job)
	log.Debugf("JobController: onAdd ns=%s jobName=%s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
}

func (c *JobController) onUpdate(oldObj, newObj interface{}) {
	job := newObj.(*apiv1.Job)
	log.Debugf("[JobController] onUpdate ns=%s %s active=%d succeeded=%d conditions=[%v]", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink, job.Status.Active, job.Status.Succeeded, job.Status.Conditions)

	var err error

	//handle the case of rmdata jobs succeeding
	labels := job.GetObjectMeta().GetLabels()
	if job.Status.Succeeded > 0 && labels[util.LABEL_RMDATA] == "true" {
		log.Debugf("jobController onUpdate rmdata job case")
		err = handleRmdata(job, c.JobClient, c.JobClientset, job.ObjectMeta.Namespace)
		if err != nil {
			log.Error(err)
		}
		return
	}

	//handle the case of a pgbasebackup job being added
	if labels[util.LABEL_PGBACKUP] == "true" {
		log.Debugf("jobController onUpdate pgbasebackup job case")
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
			err = util.Patch(c.JobClient, "/spec/backupstatus", status, "pgbackups", dbname, job.ObjectMeta.Namespace)
			if err != nil {
				log.Error("error in patching pgbackup " + labels["pg-database"] + err.Error())
			}
		}

		return

	}

	//handle the case of a backrest restore job being added
	if labels[util.LABEL_BACKREST_RESTORE] == "true" {
		log.Debugf("jobController onUpdate backrest restore job case")
		log.Debugf("got a backrest restore job status=%d", job.Status.Succeeded)
		if job.Status.Succeeded == 1 {
			log.Debugf("set status to restore job completed  for %s", labels[util.LABEL_PG_CLUSTER])
			log.Debugf("workflow to update is %s", labels[crv1.PgtaskWorkflowID])
			err = util.Patch(c.JobClient, "/spec/backreststatus", crv1.JobCompletedStatus, "pgtasks", job.Name, job.ObjectMeta.Namespace)
			if err != nil {
				log.Error("error in patching pgtask " + labels[util.LABEL_JOB_NAME] + err.Error())
			}

			backrestoperator.UpdateRestoreWorkflow(c.JobClient, c.JobClientset, labels[util.LABEL_PG_CLUSTER], crv1.PgtaskWorkflowBackrestRestorePVCCreatedStatus, job.ObjectMeta.Namespace, labels[crv1.PgtaskWorkflowID], labels[util.LABEL_BACKREST_RESTORE_TO_PVC])
		}

		return
	}

	// handle the case of a pgdump job being added
	if labels[util.LABEL_BACKUP_TYPE_PGDUMP] == "true" {
		log.Debugf("jobController onUpdate pgdump job case")
		log.Debugf("pgdump job status=%d", job.Status.Succeeded)
		log.Debugf("update the status to completed here for pgdump %s", labels[util.LABEL_PG_DATABASE])

		status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"

		if job.Status.Succeeded == 0 {
			status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
		}

		if job.Status.Failed > 0 {
			status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
		}

		//update the pgdump task status to submitted - updates task, not the job.
		dumpTask := labels[util.LABEL_PGTASK]
		err = util.Patch(c.JobClient, "/spec/status", status, "pgtasks", dumpTask, job.ObjectMeta.Namespace)

		if err != nil {
			log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		}

		return
	}

	// handle the case of a pgrestore job being added
	if labels[util.LABEL_RESTORE_TYPE_PGRESTORE] == "true" {
		log.Debugf("jobController onUpdate pgrestore job case")
		log.Debugf("pgdump job status=%d", job.Status.Succeeded)
		log.Debugf("update the status to completed here for pgrestore %s", labels[util.LABEL_PG_DATABASE])

		status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"

		if job.Status.Succeeded == 0 {
			status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
		}

		if job.Status.Failed > 0 {
			status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
		}

		//update the pgdump task status to submitted - updates task, not the job.
		restoreTask := labels[util.LABEL_PGTASK]
		err = util.Patch(c.JobClient, "/spec/status", status, "pgtasks", restoreTask, job.ObjectMeta.Namespace)

		if err != nil {
			log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		}

		return
	}

	//handle the case of a backrest job being added
	if labels[util.LABEL_BACKREST] == "true" {
		log.Debugf("jobController onUpdate backrest job case")
		log.Debugf("got a backrest job status=%d", job.Status.Succeeded)
		if job.Status.Succeeded == 1 {
			log.Debugf("update the status to completed here for backrest %s job %s", labels[util.LABEL_PG_DATABASE], job.Name)
			err = util.Patch(c.JobClient, "/spec/backreststatus", crv1.JobCompletedStatus, "pgtasks", job.Name, job.ObjectMeta.Namespace)
			if err != nil {
				log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
			}
		}

		return

	}
}

// onDelete is called when a pgcluster is deleted
func (c *JobController) onDelete(obj interface{}) {
	job := obj.(*apiv1.Job)
	log.Debugf("[JobController] onDelete ns=%s %s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
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

	selector := util.LABEL_PG_CLUSTER + "=" + job.ObjectMeta.Labels[util.LABEL_PG_CLUSTER] + "," + util.LABEL_RMDATA + "=true"
	kubeapi.DeleteJobs(clientset, selector, namespace)

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
