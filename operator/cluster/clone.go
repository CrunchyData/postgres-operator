package cluster

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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	batch_v1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	pgBackRestRepoSyncContainerImageName = "%s/pgo-backrest-repo-sync:%s"
	pgBackRestRepoSyncJobNamePrefix      = "pgo-backrest-repo-sync-%s-%s"
	pgBackRestStanza                     = "db" // this is hardcoded throughout...
	patchResource                        = "pgtasks"
	patchURL                             = "/spec/status"
	targetClusterPGDATAPath              = "/pgdata/%s"
)

// Clone allows for one to clone the data from an existing cluster to a new
// cluster in the Operator. It works by doing the following:
//
// 1. Create some PVCs that will be utilized by the new cluster
// 2. Syncing (i.e. using rsync) the pgBackRest repository from the old cluster
// to the new cluster
// 3. perform a pgBackRest delta restore to the new PVC
// 4. Create a new cluster by using the old cluster as a template and providing
// the specifications to the new cluster, with a few "opinionated" items (e.g.
// copying over the secrets)
func Clone(clientset *kubernetes.Clientset, client *rest.RESTClient, restConfig *rest.Config, namespace string, task *crv1.Pgtask) {
	// have a guard -- if the task is completed, don't proceed furter
	if task.Spec.Status == crv1.CompletedStatus {
		log.Warn(fmt.Sprintf("pgtask [%s] has already completed", task.Spec.Name))
		return
	}

	switch task.Spec.TaskType {
	// The first step is to ensure that we have PVCs available for creating the
	// cluster, so then we can kick off the first job which is to copy the
	// contents of the pgBackRes repo from the source cluster to a destination
	// cluster
	case crv1.PgtaskCloneStep1:
		cloneStep1(clientset, client, namespace, task)
	// The second step is to kick off a pgBackRest restore job to the target
	// cluster PVC
	case crv1.PgtaskCloneStep2:
		cloneStep2(clientset, client, restConfig, namespace, task)
	// The third step is to create the new cluster!
	case crv1.PgtaskCloneStep3:
		cloneStep3(clientset, client, namespace, task)
	}
}

// PublishCloneEvent lets one publish an event related to the clone process
func PublishCloneEvent(eventType string, namespace string, task *crv1.Pgtask, errorMessage string) {
	// get the boilerplate identifiers
	sourceClusterName, targetClusterName, workflowID := getCloneTaskIdentifiers(task)
	// set up the event header
	eventHeader := events.EventHeader{
		Namespace: namespace,
		Username:  task.ObjectMeta.Labels[config.LABEL_PGOUSER],
		Topic:     []string{events.EventTopicCluster},
		Timestamp: time.Now(),
		EventType: eventType,
	}
	// get the event format itself and publish it based on the event type
	switch eventType {
	case events.EventCloneCluster:
		publishCloneClusterEvent(eventHeader, sourceClusterName, targetClusterName, workflowID)
	case events.EventCloneClusterCompleted:
		publishCloneClusterCompletedEvent(eventHeader, sourceClusterName, targetClusterName, workflowID)
	case events.EventCloneClusterFailure:
		publishCloneClusterFailureEvent(eventHeader, sourceClusterName, targetClusterName, workflowID, errorMessage)
	}
}

// UpdateCloneWorkflow updates a Workflow with the current state of the clone task
func UpdateCloneWorkflow(client *rest.RESTClient, namespace, workflowID, status string) error {
	log.Debugf("clone workflow: update workflow [%s]", workflowID)

	// we have to look up the name of the workflow bt the workflow ID, which
	// involves using a selector
	selector := fmt.Sprintf("%s=%s", crv1.PgtaskWorkflowID, workflowID)
	taskList := crv1.PgtaskList{}

	if err := kubeapi.GetpgtasksBySelector(client, &taskList, selector, namespace); err != nil {
		log.Errorf("clone workflow: could not get workflow [%s]", workflowID)
		return err
	}

	// if there is not one unique result, then we should display an error here
	if len(taskList.Items) != 1 {
		errorMsg := fmt.Sprintf("clone workflow: workflow [%s] not found", workflowID)
		log.Errorf(errorMsg)
		return errors.New(errorMsg)
	}

	// get the first task and update on the current status based on how it is
	// progressing
	task := taskList.Items[0]
	task.Spec.Parameters[status] = time.Now().Format(time.RFC3339)

	if err := kubeapi.Updatepgtask(client, &task, task.Name, namespace); err != nil {
		log.Errorf("clone workflow: could not update workflow [%s] to status [%s]", workflowID, status)
		return err
	}

	return nil
}

// cloneStep1 covers the creation of the PVCs for the new PostgreSQL cluster,
// as well as sets up and executes a job to copy (via rsync) the PgBackRest
// repository from the source cluster to the destination cluster
func cloneStep1(clientset *kubernetes.Clientset, client *rest.RESTClient, namespace string, task *crv1.Pgtask) {
	sourceClusterName, targetClusterName, workflowID := getCloneTaskIdentifiers(task)

	log.Debugf("clone step 1 called: namespace:[%s] sourcecluster:[%s] targetcluster:[%s] workflowid:[%s]",
		namespace, sourceClusterName, targetClusterName, workflowID)

	// before we get stared, let's ensure we publish an event that the clone
	// workflow has begun
	// (eventType string, namespace string, task *crv1.Pgtask, errorMessage string)
	PublishCloneEvent(events.EventCloneCluster, namespace, task, "")

	// first, update the workflow to indicate that we are creating the PVCs
	// update the workflow to indicate that the cluster is being created
	if err := UpdateCloneWorkflow(client, namespace, workflowID, crv1.PgtaskWorkflowCloneCreatePVC); err != nil {
		log.Error(err)
		// if updating the workflow fails, we can continue onward
	}

	// get the information about the current pgcluster by name, to ensure it
	// exists
	sourcePgcluster, err := getSourcePgcluster(client, namespace, sourceClusterName)

	// if there is an error getting the pgcluster, abort here
	if err != nil {
		log.Error(err)
		// publish a failure event
		errorMessage := fmt.Sprintf("Could not find source cluster: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	sourceClusterBackrestStorageType := sourcePgcluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]
	cloneBackrestStorageType := task.Spec.Parameters["backrestStorageType"]
	// if 's3' storage was selected for the clone, ensure it is enabled in the current pg cluster.
	// also, if 'local' was selected, or if no storage type was selected, ensure the cluster is using
	// local storage
	err = util.ValidateBackrestStorageTypeOnBackupRestore(cloneBackrestStorageType,
		sourceClusterBackrestStorageType, true)
	if err != nil {
		log.Error(err)
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, err.Error())
		return
	}

	// Ensure that there does *not* already exist a Pgcluster for the target
	if found := checkTargetPgCluster(client, namespace, targetClusterName); found {
		log.Errorf("[%s] already exists", targetClusterName)
		errorMessage := fmt.Sprintf("Not cloning the cluster: %s already exists", targetClusterName)
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// create PVCs for pgBackRest and PostgreSQL
	if _, _, _, _, err = createPVCs(clientset, client, task, namespace, sourcePgcluster, targetClusterName); err != nil {
		log.Error(err)
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, err.Error())
		return
	}

	log.Debug("clone step 1: created pvcs")

	// awesome. now it's time to synchronize the source and targe cluster
	// pgBackRest repositories

	// update the workflow to indicate that we are going to sync the repositories
	if err := UpdateCloneWorkflow(client, namespace, workflowID, crv1.PgtaskWorkflowCloneSyncRepo); err != nil {
		log.Error(err)
		// if updating the workflow fails, we can continue onward
	}

	// now, synchronize the repositories
	if jobName, err := createPgBackRestRepoSyncJob(clientset, namespace, task, sourcePgcluster); err == nil {
		log.Debugf("clone step 1: created pgbackrest repo sync job: [%s]", jobName)
	}

	// finally, update the pgtask to indicate that it's completed
	patchPgtaskComplete(client, namespace, task.Spec.Name)
}

// cloneStep2 creates a pgBackRest restore job for the new PostgreSQL cluster by
// running a restore from the new target cluster pgBackRest repository to the
// new target cluster PVC
func cloneStep2(clientset *kubernetes.Clientset, client *rest.RESTClient, restConfig *rest.Config, namespace string, task *crv1.Pgtask) {
	sourceClusterName, targetClusterName, workflowID := getCloneTaskIdentifiers(task)

	log.Debugf("clone step 2 called: namespace:[%s] sourcecluster:[%s] targetcluster:[%s] workflowid:[%s]",
		namespace, sourceClusterName, targetClusterName, workflowID)

	// get the information about the current pgcluster by name, to ensure it
	// exists, as we still need information about the PrimaryStorage
	sourcePgcluster, err := getSourcePgcluster(client, namespace, sourceClusterName)

	// if there is an error getting the pgcluster, abort here
	if err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not find source cluster: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// interpret the storage specs again. the volumes were already created during
	// a prior step.
	_, dataVolume, walVolume, tablespaceVolumes, err := createPVCs(
		clientset, client, task, namespace, sourcePgcluster, targetClusterName)
	if err != nil {
		log.Error(err)
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, err.Error())
		return
	}

	// Retrieve current S3 key & key secret
	s3Creds, err := util.GetS3CredsFromBackrestRepoSecret(clientset, namespace, sourcePgcluster.Name)
	if err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Unable to get S3 key and key secret from source cluster "+
			"backrest repo secret: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// we need to set up the secret for the pgBackRest repo. This is the place to
	// do it
	if err := util.CreateBackrestRepoSecrets(clientset,
		util.BackrestRepoConfig{
			BackrestS3CA:        s3Creds.AWSS3CA,
			BackrestS3Key:       s3Creds.AWSS3Key,
			BackrestS3KeySecret: s3Creds.AWSS3KeySecret,
			ClusterName:         targetClusterName,
			ClusterNamespace:    namespace,
			OperatorNamespace:   operator.PgoNamespace,
		}); err != nil {
		log.Error(err)
		// publish a failure event
		errorMessage := fmt.Sprintf("Could not find source cluster: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// ok, time for a little bit of grottiness. Ideally here we would attempt to
	// bring up the pgBackRest repo and allow the Operator to respond to this
	// event in an...evented way. However, for now, we're going to set a loop and
	// wait for the pgBackRest deployment to come up
	// to do this, we are going to mock out a targetPgcluster with the exact
	// attributes we need to make this successful
	targetPgcluster := crv1.Pgcluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      targetClusterName,
			Namespace: namespace,
			Labels: map[string]string{
				config.LABEL_BACKREST: "true",
			},
		},
		Spec: crv1.PgclusterSpec{
			BackrestS3Bucket:   sourcePgcluster.Spec.BackrestS3Bucket,
			BackrestS3Endpoint: sourcePgcluster.Spec.BackrestS3Endpoint,
			BackrestS3Region:   sourcePgcluster.Spec.BackrestS3Region,
			Port:               sourcePgcluster.Spec.Port,
			PrimaryStorage:     sourcePgcluster.Spec.PrimaryStorage,
			CCPImagePrefix:     sourcePgcluster.Spec.CCPImagePrefix,
			PGOImagePrefix:     sourcePgcluster.Spec.PGOImagePrefix,
			UserLabels: map[string]string{
				config.LABEL_BACKREST_STORAGE_TYPE: sourcePgcluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE],
			},
		},
	}

	// create the deployment without creating the PVC given we've already done that
	if err := backrest.CreateRepoDeployment(clientset, namespace, &targetPgcluster, false,
		1); err != nil {
		log.Error(err)
		// publish a failure event
		errorMessage := fmt.Sprintf("Could not create new pgbackrest repo: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// ok, let's wait for the deployment to come up...per above note.
	backrestRepoDeploymentName := fmt.Sprintf(util.BackrestRepoDeploymentName, targetClusterName)
	if err := waitForDeploymentReady(clientset, namespace, backrestRepoDeploymentName, 30, 3); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not start pgbackrest repo: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// set up a map of the names of the tablespaces as well as the storage classes
	tablespaceStorageTypeMap := operator.GetTablespaceStorageTypeMap(sourcePgcluster.Spec.TablespaceMounts)

	// combine supplemental groups from all volumes
	var supplementalGroups []int64
	supplementalGroups = append(supplementalGroups, dataVolume.SupplementalGroups...)
	for _, v := range tablespaceVolumes {
		supplementalGroups = append(supplementalGroups, v.SupplementalGroups...)
	}

	backrestRestoreJobFields := backrest.BackrestRestoreJobTemplateFields{
		JobName:          fmt.Sprintf("restore-%s-%s", targetClusterName, util.RandStringBytesRmndr(4)),
		ClusterName:      targetClusterName,
		SecurityContext:  operator.GetPodSecurityContext(supplementalGroups),
		ToClusterPVCName: targetClusterName, // the PVC name should match that of the target cluster
		WorkflowID:       workflowID,
		// use a delta restore in order to optimize how the restore occurs
		CommandOpts: "--delta",
		// PITRTarget is not supported in the first iteration of clone
		PGOImagePrefix:      util.GetValueOrDefault(sourcePgcluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:         operator.Pgo.Pgo.PGOImageTag,
		PgbackrestStanza:    pgBackRestStanza,
		PgbackrestDBPath:    fmt.Sprintf(targetClusterPGDATAPath, targetClusterName),
		PgbackrestRepo1Path: util.GetPGBackRestRepoPath(targetPgcluster),
		PgbackrestRepo1Host: fmt.Sprintf(util.BackrestRepoServiceName, targetClusterName),
		PgbackrestRepoType:  operator.GetRepoType(task.Spec.Parameters["backrestStorageType"]),
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(sourcePgcluster, clientset, namespace),

		TablespaceVolumes:      operator.GetTablespaceVolumesJSON(targetClusterName, tablespaceStorageTypeMap),
		TablespaceVolumeMounts: operator.GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap),
	}

	if sourcePgcluster.Spec.WALStorage.StorageType != "" {
		arg, err := getLinkMap(clientset, restConfig, sourcePgcluster, targetClusterName)
		if err != nil {
			log.Error(err)
			errorMessage := fmt.Sprintf("Could not determine PostgreSQL version: %s", err.Error())
			PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
			return
		}
		backrestRestoreJobFields.CommandOpts += " " + arg
	}

	// substitute the variables into the BackrestRestore job template
	var backrestRestoreJobDoc bytes.Buffer

	if err = config.BackrestRestorejobTemplate.Execute(&backrestRestoreJobDoc, backrestRestoreJobFields); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not create pgbackrest restore template: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// create the pgBackRest restore job!
	job := batch_v1.Job{}

	if err := json.Unmarshal(backrestRestoreJobDoc.Bytes(), &job); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not turn pgbackrest restore template into JSON: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	if sourcePgcluster.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToBackRest(&job.Spec.Template.Spec, walVolume)
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_BACKREST_RESTORE,
		&job.Spec.Template.Spec.Containers[0])

	// update the job annotations to include information about the source and
	// target cluster
	if job.ObjectMeta.Annotations == nil {
		job.ObjectMeta.Annotations = map[string]string{}
	}

	job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_BACKREST_PVC_SIZE] = task.Spec.Parameters[util.CloneParameterBackrestPVCSize]
	job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_ENABLE_METRICS] = task.Spec.Parameters[util.CloneParameterEnableMetrics]
	job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_PVC_SIZE] = task.Spec.Parameters[util.CloneParameterPVCSize]
	job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_SOURCE_CLUSTER_NAME] = sourcePgcluster.Spec.ClusterName
	job.ObjectMeta.Annotations[config.ANNOTATION_CLONE_TARGET_CLUSTER_NAME] = targetClusterName
	// also add the label to indicate this is also part of a clone job!
	if job.ObjectMeta.Labels == nil {
		job.ObjectMeta.Labels = map[string]string{}
	}
	job.ObjectMeta.Labels[config.LABEL_PGO_CLONE_STEP_2] = "true"
	job.ObjectMeta.Labels[config.LABEL_PGOUSER] = task.ObjectMeta.Labels[config.LABEL_PGOUSER]

	// create the Job in Kubernetes
	if jobName, err := kubeapi.CreateJob(clientset, &job, namespace); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not create pgbackrest restore job: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
	} else {
		log.Debugf("clone step 2: created restore job [%s]", jobName)
	}

	// finally, update the pgtask to indicate it's complete
	patchPgtaskComplete(client, namespace, task.Spec.Name)
}

// cloneStep3 creates the new cluster by creating a new Pgcluster
func cloneStep3(clientset *kubernetes.Clientset, client *rest.RESTClient, namespace string, task *crv1.Pgtask) {
	sourceClusterName, targetClusterName, workflowID := getCloneTaskIdentifiers(task)

	log.Debugf("clone step 3 called: namespace:[%s] sourcecluster:[%s] targetcluster:[%s] workflowid:[%s]",
		namespace, sourceClusterName, targetClusterName, workflowID)

	// get the information about the current pgcluster by name, to ensure we can
	// copy over some of the necessary cluster attributes
	sourcePgcluster, err := getSourcePgcluster(client, namespace, sourceClusterName)

	// if there is an error getting the pgcluster, abort here
	if err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not find source cluster: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// first, clean up any existing pgBackRest repo deployment and services, as
	// these will be recreated
	backrestRepoDeploymentName := fmt.Sprintf(util.BackrestRepoDeploymentName, targetClusterName)
	// ignore errors here...we can let the errors occur later on, e.g. if there is
	// a failure to delete
	_ = kubeapi.DeleteDeployment(clientset, backrestRepoDeploymentName, namespace)
	_ = kubeapi.DeleteService(clientset, backrestRepoDeploymentName, namespace)

	// let's actually wait to see if they are deleted
	if err := waitForDeploymentDelete(clientset, namespace, backrestRepoDeploymentName, 30, 3); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not remove temporary pgbackrest repo: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return
	}

	// and go forth and create the cluster!
	if err := createCluster(clientset, client, task, sourcePgcluster, namespace, targetClusterName, workflowID); err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Could not create cloned cluster: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
	}

	// we did all we can do with the clone! publish an event
	PublishCloneEvent(events.EventCloneClusterCompleted, namespace, task, "")

	// finally, update the pgtask to indicate it's complete
	patchPgtaskComplete(client, namespace, task.Spec.Name)
}

// createPgBackRestRepoSyncJob prepares and creates the job that will use
// rsync to synchronize two pgBackRest repositories, i.e. it will copy the files
// from the source PostgreSQL cluster to the pgBackRest repository in the target
// cluster
func createPgBackRestRepoSyncJob(clientset *kubernetes.Clientset, namespace string, task *crv1.Pgtask, sourcePgcluster crv1.Pgcluster) (string, error) {
	targetClusterName := task.Spec.Parameters["targetClusterName"]
	workflowID := task.Spec.Parameters[crv1.PgtaskWorkflowID]
	// set the name of the job, with the "entropy" that we add
	jobName := fmt.Sprintf(pgBackRestRepoSyncJobNamePrefix, targetClusterName, util.RandStringBytesRmndr(4))

	podSecurityContext := v1.PodSecurityContext{
		SupplementalGroups: sourcePgcluster.Spec.BackrestStorage.GetSupplementalGroups(),
	}

	if !operator.Pgo.Cluster.DisableFSGroup {
		podSecurityContext.FSGroup = &crv1.PGFSGroup
	}

	// set up the job template to synchronize the pgBackRest repo
	job := batch_v1.Job{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: jobName,
			Annotations: map[string]string{
				// these annotations are used for the subsequent steps to be
				// able to identify how to connect these jobs
				config.ANNOTATION_CLONE_BACKREST_PVC_SIZE:   task.Spec.Parameters[util.CloneParameterBackrestPVCSize],
				config.ANNOTATION_CLONE_ENABLE_METRICS:      task.Spec.Parameters[util.CloneParameterEnableMetrics],
				config.ANNOTATION_CLONE_PVC_SIZE:            task.Spec.Parameters[util.CloneParameterPVCSize],
				config.ANNOTATION_CLONE_SOURCE_CLUSTER_NAME: sourcePgcluster.Spec.ClusterName,
				config.ANNOTATION_CLONE_TARGET_CLUSTER_NAME: targetClusterName,
			},
			Labels: map[string]string{
				config.LABEL_VENDOR:           config.LABEL_CRUNCHY,
				config.LABEL_PGO_CLONE_STEP_1: "true",
				config.LABEL_PGOUSER:          task.ObjectMeta.Labels[config.LABEL_PGOUSER],
				config.LABEL_PG_CLUSTER:       targetClusterName,
				config.LABEL_WORKFLOW_ID:      workflowID,
			},
		},
		Spec: batch_v1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: jobName,
					Labels: map[string]string{
						config.LABEL_VENDOR:           config.LABEL_CRUNCHY,
						config.LABEL_PGO_CLONE_STEP_1: "true",
						config.LABEL_PGOUSER:          task.ObjectMeta.Labels[config.LABEL_PGOUSER],
						config.LABEL_PG_CLUSTER:       targetClusterName,
						config.LABEL_SERVICE_NAME:     targetClusterName,
					},
				},
				// Spec for the pod that will run the pgo-backrest-repo-sync job
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "rsync",
							Image: fmt.Sprintf(pgBackRestRepoSyncContainerImageName,
								util.GetValueOrDefault(sourcePgcluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix), operator.Pgo.Pgo.PGOImageTag),
							Env: []v1.EnvVar{
								{
									Name:  "PGBACKREST_REPO1_HOST",
									Value: fmt.Sprintf(util.BackrestRepoServiceName, sourcePgcluster.Spec.ClusterName),
								},
								{
									Name:  "PGBACKREST_REPO1_PATH",
									Value: util.GetPGBackRestRepoPath(sourcePgcluster),
								},
								// NOTE: this needs to be a name like this in order to not
								// confuse pgBackRest, which does support "REPO*" name
								{
									Name: "NEW_PGBACKREST_REPO",
									Value: util.GetPGBackRestRepoPath(crv1.Pgcluster{
										ObjectMeta: meta_v1.ObjectMeta{
											Name: targetClusterName,
										},
									}),
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: config.VOLUME_PGBACKREST_REPO_MOUNT_PATH,
									Name:      config.VOLUME_PGBACKREST_REPO_NAME,
								},
								{
									MountPath: config.VOLUME_SSHD_MOUNT_PATH,
									Name:      config.VOLUME_SSHD_NAME,
									ReadOnly:  true,
								},
							},
						},
					},
					RestartPolicy:      v1.RestartPolicyNever,
					SecurityContext:    &podSecurityContext,
					ServiceAccountName: config.LABEL_BACKREST,
					Volumes: []v1.Volume{
						{
							Name: config.VOLUME_PGBACKREST_REPO_NAME,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf(util.BackrestRepoPVCName, targetClusterName),
								},
							},
						},
						// the SSHD volume that contains the SSHD secrets
						{
							Name: config.VOLUME_SSHD_NAME,
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									// the SSHD secret is stored under the name of the *source*
									// cluster, as we have yet to create the target cluster!
									SecretName: fmt.Sprintf("%s-backrest-repo-config", sourcePgcluster.Spec.ClusterName),
									// DefaultMode: &pgBackRestRepoVolumeDefaultMode,
								},
							},
						},
					},
				},
			},
		},
	}

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_BACKREST_REPO_SYNC,
		&job.Spec.Template.Spec.Containers[0])

	// Retrieve current S3 key & key secret
	s3Creds, err := util.GetS3CredsFromBackrestRepoSecret(clientset, namespace, sourcePgcluster.Name)
	if err != nil {
		log.Error(err)
		errorMessage := fmt.Sprintf("Unable to get S3 key and key secret from source cluster "+
			"backrest repo secret: %s", err.Error())
		PublishCloneEvent(events.EventCloneClusterFailure, namespace, task, errorMessage)
		return "", err
	}
	// if using S3 for the clone, the add the S3 env vars to the env
	if strings.Contains(sourcePgcluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE],
		"s3") {
		syncEnv := job.Spec.Template.Spec.Containers[0].Env
		syncEnv = append(syncEnv, []v1.EnvVar{
			{
				Name:  "BACKREST_STORAGE_SOURCE",
				Value: task.Spec.Parameters["backrestStorageType"],
			},
			{
				Name: "PGBACKREST_REPO1_S3_BUCKET",
				Value: getS3Param(sourcePgcluster.Spec.BackrestS3Bucket,
					operator.Pgo.Cluster.BackrestS3Bucket),
			},
			{
				Name: "PGBACKREST_REPO1_S3_ENDPOINT",
				Value: getS3Param(sourcePgcluster.Spec.BackrestS3Endpoint,
					operator.Pgo.Cluster.BackrestS3Endpoint),
			},
			{
				Name: "PGBACKREST_REPO1_S3_REGION",
				Value: getS3Param(sourcePgcluster.Spec.BackrestS3Region,
					operator.Pgo.Cluster.BackrestS3Region),
			},
			{
				Name:  "PGBACKREST_REPO1_S3_KEY",
				Value: s3Creds.AWSS3Key,
			},
			{
				Name:  "PGBACKREST_REPO1_S3_KEY_SECRET",
				Value: s3Creds.AWSS3KeySecret,
			},
			{
				Name:  "PGBACKREST_REPO1_S3_CA_FILE",
				Value: "/sshd/aws-s3-ca.crt",
			},
		}...)
		if operator.IsLocalAndS3Storage(
			sourcePgcluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]) {
			syncEnv = append(syncEnv, []v1.EnvVar{
				{
					Name:  "PGHA_PGBACKREST_LOCAL_S3_STORAGE",
					Value: "true",
				},
			}...)
		}
		job.Spec.Template.Spec.Containers[0].Env = syncEnv
	}

	// create the job!
	if jobName, err := kubeapi.CreateJob(clientset, &job, namespace); err != nil {
		log.Error(err)
		// the error event occurs at a different level
		return "", err
	} else {
		return jobName, nil
	}
}

// createPVCs is the first step in cloning a PostgreSQL cluster. It creates
// several PVCs that are required for operating a PostgreSQL cluster:
// - the PVC that stores the PostgreSQL PGDATA
// - the PVC that stores the PostgreSQL WAL
// - the PVC that stores the pgBackRest repo
//
// Additionally, if there are any tablespaces on the original cluster, it will
// create those too.
//
// if the user spceified a different PVCSize than what is in the storage spec,
// then that gets used
func createPVCs(clientset *kubernetes.Clientset, client *rest.RESTClient,
	task *crv1.Pgtask, namespace string, sourcePgcluster crv1.Pgcluster, targetClusterName string,
) (
	backrestVolume, dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult,
	err error,
) {
	// first, create the PVC for the pgBackRest storage, as we will be needing
	// that sooner
	{
		storage := sourcePgcluster.Spec.BackrestStorage
		if size := task.Spec.Parameters[util.CloneParameterBackrestPVCSize]; size != "" {
			storage.Size = size
		}
		// the PVCName for pgBackRest is derived from the target cluster name
		backrestPVCName := fmt.Sprintf(util.BackrestRepoPVCName, targetClusterName)
		backrestVolume, err = pvc.CreateIfNotExists(clientset,
			storage, backrestPVCName, targetClusterName, namespace)
	}

	// now create the PVC for the target cluster
	if err == nil {
		storage := sourcePgcluster.Spec.PrimaryStorage
		if size := task.Spec.Parameters[util.CloneParameterPVCSize]; size != "" {
			storage.Size = size
		}
		dataVolume, err = pvc.CreateIfNotExists(clientset,
			storage, targetClusterName, targetClusterName, namespace)
	}

	if err == nil {
		walVolume, err = pvc.CreateIfNotExists(clientset,
			sourcePgcluster.Spec.WALStorage, targetClusterName+"-wal", targetClusterName, namespace)
	}

	// if there are any tablespaces, create PVCs for those
	tablespaceVolumes = make(map[string]operator.StorageResult, len(sourcePgcluster.Spec.TablespaceMounts))
	for tablespaceName, storageSpec := range sourcePgcluster.Spec.TablespaceMounts {
		if err == nil {
			// generate the tablespace PVC name from the name of the clone cluster and
			// the name of this tablespace
			tablespacePVCName := operator.GetTablespacePVCName(targetClusterName, tablespaceName)
			tablespaceVolumes[tablespaceName], err = pvc.CreateIfNotExists(clientset,
				storageSpec, tablespacePVCName, targetClusterName, namespace)
		}
	}

	return
}

func createCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, task *crv1.Pgtask, sourcePgcluster crv1.Pgcluster, namespace string, targetClusterName string, workflowID string) error {
	// first, handle copying over the cluster secrets so they are available when
	// the cluster is created
	cloneClusterSecrets := util.CloneClusterSecrets{
		// ensure the pgBackRest secret is not copied over, as we will need to
		// initialize a new repository
		AdditionalSelectors: []string{"pgo-backrest-repo!=true"},
		ClientSet:           clientset,
		Namespace:           namespace,
		SourceClusterName:   sourcePgcluster.Spec.ClusterName,
		TargetClusterName:   targetClusterName,
	}

	if err := cloneClusterSecrets.Clone(); err != nil {
		log.Error(err)
		return err
	}

	// set up the target cluster
	targetPgcluster := &crv1.Pgcluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Annotations: map[string]string{
				config.ANNOTATION_CURRENT_PRIMARY: targetClusterName,
			},
			Name: targetClusterName,
			Labels: map[string]string{
				config.LABEL_NAME: targetClusterName,
				// we will be opinionated and say that HA must be enabled
				config.LABEL_AUTOFAIL: "true",
				// we will also be opinionated and say that pgBackRest must be enabled,
				// otherwise a later step will cloning the pgBackRest repo will fail
				config.LABEL_BACKREST: "true",
				// carry the original user who issued the clone request to here
				config.LABEL_PGOUSER: task.ObjectMeta.Labels[config.LABEL_PGOUSER],
				// assign the current workflow ID
				config.LABEL_WORKFLOW_ID: workflowID,
				// want to have the vendor label here
				config.LABEL_VENDOR: config.LABEL_CRUNCHY,
			},
		},
		Spec: crv1.PgclusterSpec{
			ArchiveStorage:     sourcePgcluster.Spec.ArchiveStorage,
			BackrestStorage:    sourcePgcluster.Spec.BackrestStorage,
			BackrestS3Bucket:   sourcePgcluster.Spec.BackrestS3Bucket,
			BackrestS3Endpoint: sourcePgcluster.Spec.BackrestS3Endpoint,
			BackrestS3Region:   sourcePgcluster.Spec.BackrestS3Region,
			BackrestResources:  sourcePgcluster.Spec.BackrestResources,
			ClusterName:        targetClusterName,
			CCPImage:           sourcePgcluster.Spec.CCPImage,
			CCPImagePrefix:     sourcePgcluster.Spec.CCPImagePrefix,
			CCPImageTag:        sourcePgcluster.Spec.CCPImageTag,
			// We're not copying over the collect container in the clone...but we will
			// maintain the secret in case one brings up the collect container
			CollectSecretName: fmt.Sprintf("%s%s", targetClusterName, crv1.CollectSecretSuffix),
			// CustomConfig is not set as in the future this will be a parameter we
			// allow the user to pass in
			Database:       sourcePgcluster.Spec.Database,
			ExporterPort:   sourcePgcluster.Spec.ExporterPort,
			Name:           targetClusterName,
			Namespace:      namespace,
			PGBadgerPort:   sourcePgcluster.Spec.PGBadgerPort,
			PGOImagePrefix: sourcePgcluster.Spec.PGOImagePrefix,
			// PgBouncer will be disabled to start
			PgBouncer:         crv1.PgBouncerSpec{},
			PodAntiAffinity:   sourcePgcluster.Spec.PodAntiAffinity,
			Policies:          sourcePgcluster.Spec.Policies,
			Port:              sourcePgcluster.Spec.Port,
			PrimaryHost:       sourcePgcluster.Spec.PrimaryHost,
			PrimaryStorage:    sourcePgcluster.Spec.PrimaryStorage,
			PrimarySecretName: fmt.Sprintf("%s%s", targetClusterName, crv1.PrimarySecretSuffix),
			// Replicas is set to "0" because we want to ensure that no replicas are
			// provisioned with the clone
			Replicas:       "0",
			ReplicaStorage: sourcePgcluster.Spec.ReplicaStorage,
			Resources:      sourcePgcluster.Spec.Resources,
			RootSecretName: fmt.Sprintf("%s%s", targetClusterName, crv1.RootSecretSuffix),
			// SecretFrom needs to be set as the "sourcePgcluster.Spec.ClusterName"
			// as this will indicate we got our secrets from the original cluster
			// ...this does NOT copy over the secrets as I thought it would.
			SecretFrom:      sourcePgcluster.Spec.ClusterName,
			SyncReplication: sourcePgcluster.Spec.SyncReplication,
			User:            sourcePgcluster.Spec.User,
			UserSecretName:  fmt.Sprintf("%s-%s%s", targetClusterName, sourcePgcluster.Spec.User, crv1.UserSecretSuffix),
			// UserLabels can be further expanded, but for now we will just track
			// which version of pgo is creating this
			UserLabels: map[string]string{
				config.LABEL_PGO_VERSION:           msgs.PGO_VERSION,
				config.LABEL_BACKREST_STORAGE_TYPE: sourcePgcluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE],
			},
			TablespaceMounts: sourcePgcluster.Spec.TablespaceMounts,
			WALStorage:       sourcePgcluster.Spec.WALStorage,
		},
		Status: crv1.PgclusterStatus{
			State:   crv1.PgclusterStateCreated,
			Message: "Created, not processed yet",
		},
	}

	// if any of the PVC sizes are overridden, indicate this in the cluster spec
	// here
	// first, handle the override for the primary/replica PVC size
	if task.Spec.Parameters[util.CloneParameterPVCSize] != "" {
		targetPgcluster.Spec.PrimaryStorage.Size = task.Spec.Parameters[util.CloneParameterPVCSize]
		targetPgcluster.Spec.ReplicaStorage.Size = task.Spec.Parameters[util.CloneParameterPVCSize]
	}

	// next, for the pgBackRest PVC
	if task.Spec.Parameters[util.CloneParameterBackrestPVCSize] != "" {
		targetPgcluster.Spec.BackrestStorage.Size = task.Spec.Parameters[util.CloneParameterBackrestPVCSize]
	}

	// check to see if the metrics collection should be performed
	if task.Spec.Parameters[util.CloneParameterEnableMetrics] == "true" {
		targetPgcluster.Spec.UserLabels[config.LABEL_COLLECT] = "true"
	}

	// update the workflow to indicate that the cluster is being created
	if err := UpdateCloneWorkflow(client, namespace, workflowID, crv1.PgtaskWorkflowCloneClusterCreate); err != nil {
		log.Error(err)
		return err
	}

	// create the new cluster!
	if err := kubeapi.Createpgcluster(client, targetPgcluster, namespace); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// checkTargetPgCluster checks to see if the target Pgcluster may already exist.
// if it does, the likely action of the caller is to abort the clone, as we do
// not want to override a PostgreSQL cluster that already exists, but we will
// let the function caller
func checkTargetPgCluster(client *rest.RESTClient, namespace, targetClusterName string) bool {
	targetPgcluster := crv1.Pgcluster{}

	found, _ := kubeapi.Getpgcluster(client, &targetPgcluster, targetClusterName, namespace)

	return found
}

// getCloneTaskIdentifiers returns the source and target cluster names as well
// as the workflow ID
func getCloneTaskIdentifiers(task *crv1.Pgtask) (string, string, string) {
	return task.Spec.Parameters["sourceClusterName"],
		task.Spec.Parameters["targetClusterName"],
		task.Spec.Parameters[crv1.PgtaskWorkflowID]
}

// getLinkMap returns the pgBackRest argument to support a WAL volume.
func getLinkMap(clientset *kubernetes.Clientset, restConfig *rest.Config, cluster crv1.Pgcluster, targetClusterName string) (string, error) {
	pods, err := kubeapi.GetPods(clientset, "pgo-pg-database=true,pg-cluster="+cluster.Name, cluster.Namespace)
	if err != nil {
		return "", err
	}
	if len(pods.Items) < 1 {
		return "", errors.New("found no cluster pods")
	}

	// PGVERSION environment variable is available in our PostgreSQL containers.
	// The following is the same logic we use in shell scripts there.
	stdout, _, err := kubeapi.ExecToPodThroughAPI(restConfig, clientset,
		[]string{"bash", "-c", `
		if printf '10\n'${PGVERSION} | sort -VC
		then
			echo -n '--link-map=pg_wal='
		else
			echo -n '--link-map=pg_xlog='
		fi`},
		pods.Items[0].Spec.Containers[0].Name,
		pods.Items[0].Name,
		pods.Items[0].Namespace,
		nil)

	return stdout + config.PostgreSQLWALPath(targetClusterName), err
}

// getS3Param returns either the value provided by 'sourceClusterS3param' if not en empty string,
// otherwise return the equivlant value from the pgo.yaml global configuration filer
func getS3Param(sourceClusterS3param, pgoConfigParam string) string {
	if sourceClusterS3param != "" {
		return sourceClusterS3param
	}

	return pgoConfigParam
}

// getSourcePgcluster attempts to find the Pgcluster CRD for the source cluster
// used for the clone
func getSourcePgcluster(client *rest.RESTClient, namespace, sourceClusterName string) (crv1.Pgcluster, error) {
	sourcePgcluster := crv1.Pgcluster{}

	_, err := kubeapi.Getpgcluster(client, &sourcePgcluster, sourceClusterName,
		namespace)

	return sourcePgcluster, err
}

// patchPgtaskComplete updates the pgtask CRD to indicate that the task is now
// complete
func patchPgtaskComplete(client *rest.RESTClient, namespace, taskName string) {
	if err := util.Patch(client, patchURL, crv1.CompletedStatus, patchResource, taskName, namespace); err != nil {
		log.Error("error in status patch " + err.Error())
	}
}

// publishCloneClusterEvent publishes the event when the cluster clone process
// has started
func publishCloneClusterEvent(eventHeader events.EventHeader, sourceClusterName, targetClusterName, workflowID string) {
	// set up the event
	event := events.EventCloneClusterFormat{
		EventHeader:       eventHeader,
		SourceClusterName: sourceClusterName,
		TargetClusterName: targetClusterName,
		WorkflowID:        workflowID,
	}
	// attempt to publish the event; if it fails, log the error, but keep moving
	// on
	if err := events.Publish(event); err != nil {
		log.Error(err)
	}
}

// publishCloneClusterCompleted publishes the event when the cluster clone process
// has successfully completed
func publishCloneClusterCompletedEvent(eventHeader events.EventHeader, sourceClusterName, targetClusterName, workflowID string) {
	// set up the event
	event := events.EventCloneClusterCompletedFormat{
		EventHeader:       eventHeader,
		SourceClusterName: sourceClusterName,
		TargetClusterName: targetClusterName,
		WorkflowID:        workflowID,
	}
	// attempt to publish the event; if it fails, log the error, but keep moving
	// on
	if err := events.Publish(event); err != nil {
		log.Error(err)
	}
}

// publishCloneClusterCompleted publishes the event when the cluster clone process
// has successfully completed, including the error message
func publishCloneClusterFailureEvent(eventHeader events.EventHeader, sourceClusterName, targetClusterName, workflowID, errorMessage string) {
	// set up the event
	event := events.EventCloneClusterFailureFormat{
		EventHeader:       eventHeader,
		ErrorMessage:      errorMessage,
		SourceClusterName: sourceClusterName,
		TargetClusterName: targetClusterName,
		WorkflowID:        workflowID,
	}
	// attempt to publish the event; if it fails, log the error, but keep moving
	// on
	if err := events.Publish(event); err != nil {
		log.Error(err)
	}
}

// waitForDeploymentDelete waits until a deployment and its associated service
// are deleted
func waitForDeploymentDelete(clientset *kubernetes.Clientset, namespace, deploymentName string, timeoutSecs, periodSecs time.Duration) error {
	timeout := time.After(timeoutSecs * time.Second)
	tick := time.NewTicker(periodSecs * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("Timed out waiting for deployment to be deleted: [%s]", deploymentName))
		case <-tick.C:
			_, deploymentFound, _ := kubeapi.GetDeployment(clientset, deploymentName, namespace)
			_, serviceFound, _ := kubeapi.GetService(clientset, deploymentName, namespace)
			if !(deploymentFound || serviceFound) {
				return nil
			}
			log.Debugf("deployment deleted: %t, service deleted: %t", !deploymentFound, !serviceFound)
		}
	}
}

// waitFotDeploymentReady waits for a deployment to be ready, or times out
func waitForDeploymentReady(clientset *kubernetes.Clientset, namespace, deploymentName string, timeoutSecs, periodSecs time.Duration) error {
	timeout := time.After(timeoutSecs * time.Second)
	tick := time.NewTicker(periodSecs * time.Second)
	defer tick.Stop()

	// loop until the timeout is met, or that all the replicas are ready
	for {
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("Timed out waiting for deployment to become ready: [%s]", deploymentName))
		case <-tick.C:
			if deployment, found, err := kubeapi.GetDeployment(clientset, deploymentName, namespace); err != nil {
				// if there is an error, log it but continue through the loop
				log.Error(err)
			} else if found {
				// check to see if the deployment status has succeed...if so, break out
				// of the loop
				if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
					return nil
				}
			}
		}
	}
}
