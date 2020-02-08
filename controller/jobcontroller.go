package controller

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
	"fmt"
	"sync"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	backrestoperator "github.com/crunchydata/postgres-operator/operator/backrest"
	backupoperator "github.com/crunchydata/postgres-operator/operator/backup"
	benchmarkoperator "github.com/crunchydata/postgres-operator/operator/benchmark"
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	taskoperator "github.com/crunchydata/postgres-operator/operator/task"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// JobController holds the connections for the controller
type JobController struct {
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
func (c *JobController) Run() error {

	err := c.watchJobs(c.Ctx)
	if err != nil {
		log.Errorf("Failed to register watch for job resource: %v\n", err)
		return err
	}

	<-c.Ctx.Done()
	return c.Ctx.Err()
}

// watchJobs is the event loop for job resources
func (c *JobController) watchJobs(ctx context.Context) error {
	nsList := ns.GetNamespaces(c.JobClientset, operator.InstallationName)
	log.Debugf("jobController watching %v namespaces", nsList)

	for i := 0; i < len(nsList); i++ {
		log.Infof("starting job controller for ns [%s]", nsList[i])
		c.SetupWatch(nsList[i])

	}
	return nil
}

func (c *JobController) onAdd(obj interface{}) {
	job := obj.(*apiv1.Job)

	//don't process any jobs unless they have a vendor=crunchydata
	//label
	labels := job.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("JobController: onAdd skipping job that is not crunchydata %s", job.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("JobController: onAdd ns=%s jobName=%s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
}

func (c *JobController) onUpdate(oldObj, newObj interface{}) {
	job := newObj.(*apiv1.Job)

	labels := job.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("JobController: onUpdate skipping job that is not crunchydata %s", job.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("[JobController] onUpdate ns=%s %s active=%d succeeded=%d conditions=[%v]", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink, job.Status.Active, job.Status.Succeeded, job.Status.Conditions)

	// determine if a foreground deletion of this resource is in progress
	isForegroundDeletion := false
	for _, finalizer := range job.Finalizers {
		if finalizer == meta_v1.FinalizerDeleteDependents {
			isForegroundDeletion = true
			break
		}
	}

	var err error

	//handle the case of rmdata jobs succeeding
	if job.Status.Succeeded > 0 && labels[config.LABEL_RMDATA] == "true" {
		log.Debugf("jobController onUpdate rmdata job succeeded")
		publishDeleteClusterComplete(labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER],
			job.ObjectMeta.Labels[config.LABEL_PGOUSER], job.ObjectMeta.Namespace)

		log.Debugf("jobController onUpdate rmdata job case")

		err = handleRmdata(job, c.JobClient, c.JobClientset, job.ObjectMeta.Namespace)
		if err != nil {
			log.Error(err)
		}

		return
	}

	//handle the case of a pgbasebackup job being updated
	if labels[config.LABEL_PGBACKUP] == "true" {
		log.Debugf("jobController onUpdate pgbasebackup job case")
		clusterName := job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER]
		status := crv1.JobCompletedStatus
		log.Debugf("got a pgbackup job status=%d for cluster %s", job.Status.Succeeded, clusterName)
		if job.Status.Succeeded == 0 {
			status = crv1.JobSubmittedStatus
		}
		if job.Status.Failed > 0 {
			status = crv1.JobErrorStatus
		}

		//get the pgbackup for this job
		backupName := clusterName
		var found bool
		backup := crv1.Pgbackup{}
		found, err = kubeapi.Getpgbackup(c.JobClient, &backup, backupName, job.ObjectMeta.Namespace)
		if !found {
			log.Errorf("jobController onUpdate could not find pgbackup %s", backupName)
			return
		}

		//update the backup paths if the job completed
		if status == crv1.JobCompletedStatus {
			path := backupoperator.UpdateBackupPaths(c.JobClientset, job.Name, job.ObjectMeta.Namespace)
			backup.Spec.Toc[path] = path

			// update pgtask for workflow
			taskoperator.CompleteBackupWorkflow(clusterName, c.JobClientset, c.JobClient, job.ObjectMeta.Namespace)

			publishBackupComplete(clusterName, job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], job.ObjectMeta.Labels[config.LABEL_PGOUSER], "pgbasebackup", job.ObjectMeta.Namespace, path)

		}

		err = kubeapi.Updatepgbackup(c.JobClient, &backup, backupName, job.ObjectMeta.Namespace)
		if err != nil {
			log.Error("error in updating pgbackup " + labels["pg-cluster"] + err.Error())
			return
		}

		return

	}

	//handle the case of a backrest restore job being added
	// (there is a separate handler for the case of a pgBackRest restore done
	// during the clone workflow. NOTE: this is setup to work with the existing
	// codebase)
	if labels[config.LABEL_BACKREST_RESTORE] == "true" && labels[config.LABEL_PGO_CLONE_STEP_2] != "true" {
		log.Debugf("jobController onUpdate backrest restore job case")
		log.Debugf("got a backrest restore job status=%d", job.Status.Succeeded)
		if job.Status.Succeeded == 1 {
			log.Debugf("set status to restore job completed  for %s", labels[config.LABEL_PG_CLUSTER])
			log.Debugf("workflow to update is %s", labels[crv1.PgtaskWorkflowID])
			err = util.Patch(c.JobClient, patchURL, crv1.JobCompletedStatus, patchResource, job.Name, job.ObjectMeta.Namespace)
			if err != nil {
				log.Error("error in patching pgtask " + labels[config.LABEL_JOB_NAME] + err.Error())
			}

			backrestoperator.UpdateRestoreWorkflow(c.JobClient, c.JobClientset, labels[config.LABEL_PG_CLUSTER],
				crv1.PgtaskWorkflowBackrestRestorePVCCreatedStatus, job.ObjectMeta.Namespace, labels[crv1.PgtaskWorkflowID],
				labels[config.LABEL_BACKREST_RESTORE_TO_PVC], job.Spec.Template.Spec.Affinity)
			publishRestoreComplete(labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], job.ObjectMeta.Labels[config.LABEL_PGOUSER], job.ObjectMeta.Namespace)
		}

		return
	}

	// handle the case of a pgdump job being added
	if labels[config.LABEL_BACKUP_TYPE_PGDUMP] == "true" {
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

		//update the pgdump task status to submitted - updates task, not the job.
		dumpTask := labels[config.LABEL_PGTASK]
		err = util.Patch(c.JobClient, patchURL, status, patchResource, dumpTask, job.ObjectMeta.Namespace)

		if err != nil {
			log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		}

		return
	}

	// handle the case of a pgrestore job being added
	if labels[config.LABEL_RESTORE_TYPE_PGRESTORE] == "true" {
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

		//update the pgdump task status to submitted - updates task, not the job.
		restoreTask := labels[config.LABEL_PGTASK]
		err = util.Patch(c.JobClient, patchURL, status, patchResource, restoreTask, job.ObjectMeta.Namespace)

		if err != nil {
			log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
		}

		return
	}

	//handle the case of a backrest job being added
	if !isForegroundDeletion && labels[config.LABEL_BACKREST] == "true" &&
		labels[config.LABEL_BACKREST_COMMAND] == "backup" {
		log.Debugf("jobController onUpdate backrest job case")
		log.Debugf("got a backrest job status=%d", job.Status.Succeeded)
		if job.Status.Succeeded == 1 {
			log.Debugf("update the status to completed here for backrest %s job %s", labels[config.LABEL_PG_CLUSTER], job.Name)
			err = util.Patch(c.JobClient, patchURL, crv1.JobCompletedStatus, patchResource, job.Name, job.ObjectMeta.Namespace)
			if err != nil {
				log.Error("error in patching pgtask " + job.ObjectMeta.SelfLink + err.Error())
			}
			publishBackupComplete(labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], job.ObjectMeta.Labels[config.LABEL_PGOUSER], "pgbackrest", job.ObjectMeta.Namespace, "")

			// if initial cluster backup, now annotate all existing pgreplica's to initiate replica creation
			pgreplicaList := &crv1.PgreplicaList{}
			selector := config.LABEL_PG_CLUSTER + "=" + labels[config.LABEL_PG_CLUSTER]
			if labels[config.LABEL_PGHA_BACKUP_TYPE] == crv1.BackupTypeBootstrap {
				log.Debugf("jobController onUpdate initial backup complete")

				// get the pgcluster resource for the cluster the replica is a part of
				cluster := crv1.Pgcluster{}
				_, err = kubeapi.Getpgcluster(c.JobClient, &cluster, labels[config.LABEL_PG_CLUSTER],
					job.ObjectMeta.Namespace)
				if err != nil {
					log.Error(err)
					return
				}
				clusterStatus := cluster.Status.State
				message := "Cluster has been initialized"
				err = kubeapi.PatchpgclusterStatus(c.JobClient, crv1.PgclusterStateInitialized, message,
					&cluster, job.ObjectMeta.Namespace)
				if err != nil {
					log.Error(err)
					return
				}

				err := kubeapi.GetpgreplicasBySelector(c.JobClient, pgreplicaList, selector, job.ObjectMeta.Namespace)
				if err != nil {
					log.Error(err)
					return
				}
				for _, pgreplica := range pgreplicaList.Items {
					if pgreplica.Annotations == nil {
						pgreplica.Annotations = make(map[string]string)
					}
					if clusterStatus == crv1.PgclusterStateRestore {
						pgreplica.Spec.Status = "restore"
					} else {
						pgreplica.Annotations[config.ANNOTATION_PGHA_BOOTSTRAP_REPLICA] = "true"
					}
					err = kubeapi.Updatepgreplica(c.JobClient, &pgreplica, pgreplica.Name, job.ObjectMeta.Namespace)
					if err != nil {
						log.Error(err)
						return
					}
				}
			} else if labels[config.LABEL_PGHA_BACKUP_TYPE] == crv1.BackupTypeFailover {
				err := clusteroperator.RemovePrimaryOnRoleChangeTag(c.JobClientset, c.JobConfig,
					labels[config.LABEL_PG_CLUSTER], job.ObjectMeta.Namespace)
				if err != nil {
					log.Error(err)
					return
				}
			}
		}
		return
	}

	// create an initial full backup for replica creation once stanza creation is complete
	if !isForegroundDeletion && labels[config.LABEL_BACKREST] == "true" &&
		labels[config.LABEL_BACKREST_COMMAND] == crv1.PgtaskBackrestStanzaCreate {
		log.Debugf("jobController onUpdate backrest stanza-create job case")
		if job.Status.Succeeded == 1 {
			log.Debugf("backrest stanza successfully created for cluster %s", labels[config.LABEL_PG_CLUSTER])
			log.Debugf("proceeding with the initial full backup for cluster %s as needed for replica creation",
				labels[config.LABEL_PG_CLUSTER])

			var backrestRepoPodName string
			for _, cont := range job.Spec.Template.Spec.Containers {
				for _, envVar := range cont.Env {
					if envVar.Name == "PODNAME" {
						backrestRepoPodName = envVar.Value
						log.Debugf("the backrest repo pod for the initial backup of cluster %s is %s",
							labels[config.LABEL_PG_CLUSTER], backrestRepoPodName)
					}
				}
			}
			backrestoperator.CreateInitialBackup(c.JobClient, job.ObjectMeta.Namespace,
				labels[config.LABEL_PG_CLUSTER], backrestRepoPodName)
		}
		return
	}

	if labels[config.LABEL_PGBASEBACKUP_RESTORE] == "true" {
		log.Debugf("jobController onUpdate pgbasebackup restore job case")
		log.Debugf("got a pgbasebackup restore job status=%d", job.Status.Succeeded)

		status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"
		if job.Status.Succeeded == 0 {
			status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
		}

		if job.Status.Failed > 0 {
			status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
		}

		// patch 'pgbasebackuprestore' pgtask status with job status
		err = util.Patch(c.JobClient, patchURL, status, patchResource, labels[config.LABEL_PGTASK], job.ObjectMeta.Namespace)
		if err != nil {
			log.Error("error patching pgtask '" + labels["pg-task"] + "': " + err.Error())
		}
	}

	//handle the case of a benchmark job being upddated
	if labels[config.LABEL_PGO_BENCHMARK] == "true" {
		log.Debugf("jobController onUpdate benchmark job case")
		log.Debugf("got a benchmark job status=%d", job.Status.Succeeded)

		status := crv1.JobCompletedStatus + " [" + job.ObjectMeta.Name + "]"
		if job.Status.Succeeded == 0 {
			status = crv1.JobSubmittedStatus + " [" + job.ObjectMeta.Name + "]"
		}

		if job.Status.Failed > 0 {
			status = crv1.JobErrorStatus + " [" + job.ObjectMeta.Name + "]"
		}

		err = util.Patch(c.JobClient, patchURL, status, patchResource, job.Name, job.ObjectMeta.Namespace)
		if err != nil {
			log.Error("error in patching pgtask " + labels["workflowName"] + err.Error())
		}

		benchmarkoperator.UpdateWorkflow(c.JobClient, labels["workflowName"], job.ObjectMeta.Namespace, crv1.JobCompletedStatus)

		//publish event benchmark completed
		topics := make([]string, 1)
		topics[0] = events.EventTopicCluster

		f := events.EventBenchmarkCompletedFormat{
			EventHeader: events.EventHeader{
				Namespace: job.ObjectMeta.Namespace,
				Username:  job.ObjectMeta.Labels[config.LABEL_PGOUSER],
				Topic:     topics,
				Timestamp: time.Now(),
				EventType: events.EventBenchmarkCompleted,
			},
			Clustername: labels[config.LABEL_PG_CLUSTER],
		}

		err = events.Publish(f)
		if err != nil {
			log.Error(err.Error())
		}

		return
	}

	// handle the case of a the clone "repo sync" step (aka "step 1")
	// being completed
	if labels[config.LABEL_PGO_CLONE_STEP_1] == "true" {
		log.Debugf("jobController onUpdate clone step 1 job case")

		// first, if this is being called as the result of a propogated delete, exit
		if len(job.ObjectMeta.Finalizers) > 0 {
			log.Debugf("skipping onUpdate clone step 1 job case: deletion in progress")
			return
		}

		log.Debugf("clone step 1 job status=%d", job.Status.Succeeded)

		// if the job succeed, we need to kick off step 2!
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

			// next, update the workflow to indicate that step 1 is complete
			clusteroperator.UpdateCloneWorkflow(c.JobClient, namespace, workflowID, crv1.PgtaskWorkflowCloneRestoreBackup)

			// now, set up a new pgtask that will allow us to perform the restore
			cloneTask := util.CloneTask{
				PGOUser:           job.ObjectMeta.Labels[config.LABEL_PGOUSER],
				SourceClusterName: sourceClusterName,
				TargetClusterName: targetClusterName,
				TaskStepLabel:     config.LABEL_PGO_CLONE_STEP_2,
				TaskType:          crv1.PgtaskCloneStep2,
				Timestamp:         time.Now(),
				WorkflowID:        workflowID,
			}

			task := cloneTask.Create()

			// finally, create the pgtask!
			if err := kubeapi.Createpgtask(c.JobClient, task, namespace); err != nil {
				log.Error(err)
				errorMessage := fmt.Sprintf("Could not create pgtask for step 2: %s", err.Error())
				clusteroperator.PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
			}
		}
		// ...we really shouldn't need a return here the way this function is
		// constructed...but just in case
		return
	}

	// handle the case of the clone "pgBackRest restore" step (aka "step 2")
	// being completed
	if labels[config.LABEL_PGO_CLONE_STEP_2] == "true" {
		log.Debugf("jobController onUpdate clone step 2 job case")

		// first, if this is being called as the result of a propogated delete, exit
		if len(job.ObjectMeta.Finalizers) > 0 {
			log.Debugf("skipping onUpdate clone step 2 job case: deletion in progress")
			return
		}

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
				PGOUser:           job.ObjectMeta.Labels[config.LABEL_PGOUSER],
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

		return
	}

	//handle the case of a load job being upddated
	if labels[config.LABEL_PGO_LOAD] == "true" {
		log.Debugf("jobController onUpdate load job case")
		log.Debugf("got a load job status=%d", job.Status.Succeeded)

		if job.Status.Succeeded == 1 {
			log.Debugf("load job succeeded=%d", job.Status.Succeeded)
		}

		return
	}
}

// onDelete is called when a pgcluster is deleted
func (c *JobController) onDelete(obj interface{}) {
	job := obj.(*apiv1.Job)

	labels := job.GetObjectMeta().GetLabels()
	if labels[config.LABEL_VENDOR] != "crunchydata" {
		log.Debugf("JobController: onDelete skipping job that is not crunchydata %s", job.ObjectMeta.SelfLink)
		return
	}

	log.Debugf("[JobController] onDelete ns=%s %s", job.ObjectMeta.Namespace, job.ObjectMeta.SelfLink)
}

func handleRmdata(job *apiv1.Job, restClient *rest.RESTClient, clientset *kubernetes.Clientset, namespace string) error {
	var err error

	log.Debugf("got a pgrmdata job status=%d", job.Status.Succeeded)
	labels := job.GetObjectMeta().GetLabels()
	clusterName := labels[config.LABEL_PG_CLUSTER]

	if err = kubeapi.DeleteJob(clientset, job.Name, namespace); err != nil {
		log.Error(err)
	}

	MAX_TRIES := 10
	DURATION := 5
	removed := false
	for i := 0; i < MAX_TRIES; i++ {
		log.Debugf("sleeping while job %s is removed cleanly", job.Name)
		time.Sleep(time.Second * time.Duration(DURATION))
		_, found := kubeapi.GetJob(clientset, job.Name, namespace)
		if !found {
			removed = true
			break
		}
	}

	if !removed {
		log.Error("could not remove Job %s for some reason after max tries", job.Name)
		return err
	}

	//if a user has specified --archive for a cluster then
	// an xlog PVC will be present and can be removed
	var found bool
	pvcName := clusterName + "-xlog"
	_, found, err = kubeapi.GetPVC(clientset, pvcName, namespace)
	if found {
		log.Debugf("deleting pvc %s", pvcName)
		err = pvc.Delete(clientset, pvcName, namespace)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	//delete any completed jobs for this cluster as a cleanup
	var jobList *apiv1.JobList
	jobList, err = kubeapi.GetJobs(clientset, config.LABEL_PG_CLUSTER+"="+clusterName, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	for _, j := range jobList.Items {
		if j.Status.Succeeded > 0 {
			log.Debugf("removing Job %s since it was completed", job.Name)
			err := kubeapi.DeleteJob(clientset, j.Name, namespace)
			if err != nil {
				log.Error(err)
			}

		}
	}

	return err
}

func (c *JobController) SetupWatch(ns string) {

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
	log.Debugf("JobController: created informer for namespace %s", ns)
}

func publishBackupComplete(clusterName, clusterIdentifier, username, backuptype, namespace, path string) {
	topics := make([]string, 2)
	topics[0] = events.EventTopicCluster
	topics[1] = events.EventTopicBackup

	f := events.EventCreateBackupCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreateBackupCompleted,
		},
		Clustername: clusterName,
		BackupType:  backuptype,
		Path:        path,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}
func publishRestoreComplete(clusterName, identifier, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventRestoreClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventRestoreClusterCompleted,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}

func publishDeleteClusterComplete(clusterName, identifier, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventDeleteClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeleteClusterCompleted,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}
