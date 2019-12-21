package backrest

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"os"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type BackrestRestoreJobTemplateFields struct {
	JobName             string
	ClusterName         string
	WorkflowID          string
	ToClusterPVCName    string
	SecurityContext     string
	PGOImagePrefix      string
	PGOImageTag         string
	CommandOpts         string
	PITRTarget          string
	PgbackrestStanza    string
	PgbackrestDBPath    string
	PgbackrestRepo1Path string
	PgbackrestRepo1Host string
	PgbackrestRepoType  string
	PgbackrestS3EnvVars string
	NodeSelector        string
}

// Restore ...
func Restore(restclient *rest.RESTClient, namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	clusterName := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER]
	log.Debugf("restore workflow: started for cluster %s", clusterName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow error: could not find a pgcluster in Restore Workflow for %s", clusterName)
		return
	}

	// disable autofail if it is currently enabled
	if err = util.ToggleAutoFailover(clientset, false, cluster.ObjectMeta.Labels[config.LABEL_PGHA_SCOPE],
		namespace); err != nil {
		log.Error(err)
		return
	}

	//use the storage config from pgo.yaml for Primary
	//use the storage config from the pgcluster for the restored pvc
	//storage := operator.Pgo.Storage[operator.Pgo.PrimaryStorage]
	storage := cluster.Spec.PrimaryStorage

	//create the "to-cluster" PVC to hold the new data
	var pvcName string
	pvcName, err = createPVC(storage, clusterName, restclient, namespace, clientset, task)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("restore workflow: created pvc %s for cluster %s", pvcName, clusterName)
	//delete current primary and all replica deployments
	selector := config.LABEL_PG_CLUSTER + "=" + clusterName + "," + config.LABEL_PG_DATABASE + "=true"
	var depList *appsv1.DeploymentList
	depList, err = kubeapi.GetDeployments(clientset, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get depList using %s", selector)
		return
	}

	if len(depList.Items) == 0 {
		log.Debugf("restore workflow: no primary or replicas found using selector %s. Skipping deployment deletion.", selector)
	} else {
		for _, depToDelete := range depList.Items {
			err = kubeapi.DeleteDeployment(clientset, depToDelete.Name, namespace)
			if err != nil {
				log.Errorf("restore workflow error: could not delete primary or replica %s", depToDelete.Name)
				return
			}
			log.Debugf("restore workflow: deleted primary or replica %s", depToDelete.Name)
		}
	}

	message := "Cluster is being restored"
	err = kubeapi.PatchpgclusterStatus(restclient, crv1.PgclusterStateRestore, message, &cluster, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	pgreplicaList := &crv1.PgreplicaList{}
	selector = config.LABEL_PG_CLUSTER + "=" + clusterName
	log.Debugf("Restored cluster %s went to ready, patching replicas", clusterName)
	err = kubeapi.GetpgreplicasBySelector(restclient, pgreplicaList, selector, namespace)
	if err != nil {
		log.Error(err)
		return
	}
	for _, pgreplica := range pgreplicaList.Items {
		pgreplica.Status.State = crv1.PgreplicaStatePendingRestore
		err = kubeapi.Updatepgreplica(restclient, &pgreplica, pgreplica.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}

	//sleep for a bit to give the bounce time to take effect and let
	//the backrest repo container come back and be able to service requests
	time.Sleep(time.Second * time.Duration(30))

	//since the restore 'to' name is dynamically generated we shouldn't
	//need to delete the previous job

	//delete the job if it exists from a prior run
	//kubeapi.DeleteJob(clientset, task.Spec.Name, namespace)
	//add a small sleep, this is due to race condition in delete propagation
	//time.Sleep(time.Second * 3)

	//create the Job to run the backrest restore container

	workflowID := task.Spec.Parameters[crv1.PgtaskWorkflowID]
	jobFields := BackrestRestoreJobTemplateFields{
		JobName:             "restore-" + task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER] + "-" + util.RandStringBytesRmndr(4),
		ClusterName:         task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_FROM_CLUSTER],
		SecurityContext:     util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		ToClusterPVCName:    pvcName,
		WorkflowID:          workflowID,
		CommandOpts:         task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_OPTS],
		PITRTarget:          task.Spec.Parameters[config.LABEL_BACKREST_PITR_TARGET],
		PGOImagePrefix:      operator.Pgo.Pgo.PGOImagePrefix,
		PGOImageTag:         operator.Pgo.Pgo.PGOImageTag,
		PgbackrestStanza:    task.Spec.Parameters[config.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:    task.Spec.Parameters[config.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepo1Path: task.Spec.Parameters[config.LABEL_PGBACKREST_REPO_PATH],
		PgbackrestRepo1Host: task.Spec.Parameters[config.LABEL_PGBACKREST_REPO_HOST],
		NodeSelector:        operator.GetAffinity(task.Spec.Parameters["NodeLabelKey"], task.Spec.Parameters["NodeLabelValue"], "In"),
		PgbackrestRepoType:  operator.GetRepoType(task.Spec.Parameters[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(cluster, clientset, namespace),
	}

	var doc2 bytes.Buffer
	err = config.BackrestRestorejobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		log.Error("restore workflow: error executing job template")
		return
	}

	if operator.CRUNCHY_DEBUG {
		config.BackrestRestorejobTemplate.Execute(os.Stdout, jobFields)
	}

	newjob := v1batch.Job{}
	err = json.Unmarshal(doc2.Bytes(), &newjob)
	if err != nil {
		log.Error("restore workflow: error unmarshalling json into Job " + err.Error())
		return
	}

	var jobName string
	jobName, err = kubeapi.CreateJob(clientset, &newjob, namespace)
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in creating restore job")
		return
	}
	log.Debugf("restore workflow: restore job %s created", jobName)

	publishRestore(cluster.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER], clusterName, task.ObjectMeta.Labels[config.LABEL_PGOUSER], namespace)

	err = updateWorkflow(restclient, workflowID, namespace, crv1.PgtaskWorkflowBackrestRestoreJobCreatedStatus)
	if err != nil {
		log.Error(err)
		log.Error("restore workflow: error in updating workflow status")
	}

}

func UpdateRestoreWorkflow(restclient *rest.RESTClient, clientset *kubernetes.Clientset, clusterName, status, namespace,
	workflowID, restoreToName string, affinity *v1.Affinity) {
	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType
	log.Debugf("restore workflow phase 2: taskName is %s", taskName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow phase 2 error: could not find a pgclustet in updateRestoreWorkflow for %s", clusterName)
		return
	}

	operator.UpdatePghaDefaultConfigInitFlag(clientset, true, clusterName, namespace)

	//create the new primary deployment
	CreateRestoredDeployment(restclient, &cluster, clientset, namespace, restoreToName, workflowID, affinity)

	log.Debugf("restore workflow phase  2: created restored primary was %s now %s", cluster.Spec.Name, restoreToName)

	//update workflow
	err = updateWorkflow(restclient, workflowID, namespace, crv1.PgtaskWorkflowBackrestRestorePrimaryCreatedStatus)

}

func updateWorkflow(restclient *rest.RESTClient, workflowID, namespace, status string) error {
	//update workflow
	log.Debugf("restore workflow: update workflow %s", workflowID)
	selector := crv1.PgtaskWorkflowID + "=" + workflowID
	taskList := crv1.PgtaskList{}
	err := kubeapi.GetpgtasksBySelector(restclient, &taskList, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get workflow %s", workflowID)
		return err
	}
	if len(taskList.Items) != 1 {
		log.Errorf("restore workflow error: workflow %s not found", workflowID)
		return errors.New("restore workflow error: workflow not found")
	}

	task := taskList.Items[0]
	task.Spec.Parameters[status] = time.Now().Format(time.RFC3339)
	err = kubeapi.Updatepgtask(restclient, &task, task.Name, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not update workflow %s to status %s", workflowID, status)
		return err
	}
	return err
}

func createPVC(storage crv1.PgStorageSpec, clusterName string, restclient *rest.RESTClient, namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) (string, error) {
	var err error

	//create the "to-cluster" PVC to hold the new data
	pvcName := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_TO_PVC]
	pgstoragespec := storage

	var found bool
	_, found, err = kubeapi.GetPVC(clientset, pvcName, namespace)
	if !found {
		log.Debugf("pvc %s not found, will create as part of restore", pvcName)
		//create the pvc
		err = pvc.Create(clientset, pvcName, clusterName, &pgstoragespec, namespace)
		if err != nil {
			log.Error(err.Error())
			return "", err
		}
	} else if err != nil {
		log.Error(err.Error())
		return "", err
	} else {
		log.Debugf("pvc %s found, will NOT recreate as part of restore", pvcName)
	}
	return pvcName, err

}

func CreateRestoredDeployment(restclient *rest.RESTClient, cluster *crv1.Pgcluster, clientset *kubernetes.Clientset,
	namespace, restoreToName, workflowID string, affinity *v1.Affinity) error {

	var err error

	//primaryLabels := operator.GetPrimaryLabels(cluster.Spec.Name, cluster.Spec.ClusterName, false, cluster.Spec.UserLabels)

	cluster.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = restoreToName
	cluster.Spec.UserLabels["name"] = cluster.Spec.Name
	cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER] = cluster.Spec.ClusterName

	// Set the Patroni scope to the name of the primary deployment.  Replicas will get scope using the
	// 'current-primary' label on the pgcluster
	cluster.Spec.UserLabels[config.LABEL_PGHA_SCOPE] = restoreToName

	archiveMode := "on"
	xlogdir := "false"
	archivePVCName := ""
	backrestPVCName := cluster.Spec.Name + "-backrestrepo"

	var affinityStr string
	if affinity != nil {
		log.Debugf("Affinity found on restore job, and will applied to the restored deployment")
		affinityBytes, err := json.MarshalIndent(affinity, "", "  ")
		if err != nil {
			log.Error("unable to marshall affinity obtained from restore job spec")
		}
		affinityStr = "\"affinity\":" + string(affinityBytes) + ","
	} else {
		affinityStr = operator.GetAffinity(cluster.Spec.UserLabels["NodeLabelKey"], cluster.Spec.UserLabels["NodeLabelValue"], "In")
	}

	log.Debugf("creating restored PG deployment with bouncer pass of [%s]", cluster.Spec.UserLabels[config.LABEL_PGBOUNCER_PASS])

	deploymentFields := operator.DeploymentTemplateFields{
		Name:               restoreToName,
		IsInit:             true,
		Replicas:           "1",
		ClusterName:        cluster.Spec.Name,
		PrimaryHost:        restoreToName,
		Port:               cluster.Spec.Port,
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImage:           cluster.Spec.CCPImage,
		CCPImageTag:        cluster.Spec.CCPImageTag,
		PVCName:            util.CreatePVCSnippet(cluster.Spec.PrimaryStorage.StorageType, restoreToName),
		DeploymentLabels:   operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		PodLabels:          operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		DataPathOverride:   restoreToName,
		Database:           cluster.Spec.Database,
		ArchiveMode:        archiveMode,
		ArchivePVCName:     util.CreateBackupPVCSnippet(archivePVCName),
		XLOGDir:            xlogdir,
		BackrestPVCName:    util.CreateBackrestPVCSnippet(backrestPVCName),
		SecurityContext:    util.CreateSecContext(cluster.Spec.PrimaryStorage.Fsgroup, cluster.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:     cluster.Spec.RootSecretName,
		PrimarySecretName:  cluster.Spec.PrimarySecretName,
		UserSecretName:     cluster.Spec.UserSecretName,
		NodeSelector:       affinityStr,
		PodAntiAffinity:    operator.GetPodAntiAffinity(cluster.Spec.PodAntiAffinity, cluster.Spec.Name),
		ContainerResources: operator.GetContainerResourcesJSON(&cluster.Spec.ContainerResources),
		ConfVolume:         operator.GetConfVolume(clientset, cluster, namespace),
		CollectAddon:       operator.GetCollectAddon(clientset, namespace, &cluster.Spec),
		CollectVolume:      operator.GetCollectVolume(clientset, cluster, namespace),
		BadgerAddon:        operator.GetBadgerAddon(clientset, namespace, cluster, restoreToName),
		ScopeLabel:         config.LABEL_PGHA_SCOPE,
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cluster.Labels[config.LABEL_BACKREST], cluster.Spec.ClusterName, restoreToName,
			cluster.Spec.Port, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:      operator.GetPgbackrestS3EnvVars(*cluster, clientset, namespace),
		EnableCrunchyadm:         operator.Pgo.Cluster.EnableCrunchyadm,
		ReplicaReinitOnStartFail: !operator.Pgo.Cluster.DisableReplicaStartFailReinit,
		SyncReplication:          operator.GetSyncReplication(cluster.Spec.SyncReplication),
	}

	log.Debug("collectaddon value is [" + deploymentFields.CollectAddon + "]")
	var primaryDoc bytes.Buffer
	err = config.DeploymentTemplate.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	//a form of debugging
	if operator.CRUNCHY_DEBUG {
		config.DeploymentTemplate.Execute(os.Stdout, deploymentFields)
	}

	deployment := appsv1.Deployment{}
	err = json.Unmarshal(primaryDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling primary json into Deployment " + err.Error())
		return err
	}
	err = kubeapi.CreateDeployment(clientset, &deployment, namespace)
	if err != nil {
		return err
	}

	cluster.Spec.UserLabels[config.LABEL_CURRENT_PRIMARY] = restoreToName

	cluster.Spec.UserLabels[crv1.PgtaskWorkflowID] = workflowID

	err = util.PatchClusterCRD(restclient, cluster.Spec.UserLabels, cluster, namespace)
	if err != nil {
		log.Error("could not patch primary crv1 with labels")
		return err
	}
	return err

}

func publishRestore(id, clusterName, username, namespace string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventRestoreClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventRestoreCluster,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

}
