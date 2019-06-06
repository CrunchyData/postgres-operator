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
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type restorejobTemplateFields struct {
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

	//turn off autofail if its on
	if cluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] == "true" {
		cluster.ObjectMeta.Labels[config.LABEL_AUTOFAIL] = "false"
		log.Debugf("restore workflow: turning off autofail on %s", clusterName)
		err = kubeapi.Updatepgcluster(restclient, &cluster, clusterName, namespace)
		if err != nil {
			log.Errorf("restore workflow error: could not turn off autofail on %s", clusterName)
			return
		}
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
	//delete current primary deployment
	selector := config.LABEL_SERVICE_NAME + "=" + clusterName
	var depList *appsv1.DeploymentList
	depList, err = kubeapi.GetDeployments(clientset, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get depList using %s", selector)
		return
	}

	switch depLen := len(depList.Items); depLen {
	case 0:
		log.Debugf("restore workflow: no primary found using selector %s. Skipping deployment deletion.", selector)
	case 1:
		depToDelete := depList.Items[0]

		err = kubeapi.DeleteDeployment(clientset, depToDelete.Name, namespace)
		if err != nil {
			log.Errorf("restore workflow error: could not delete primary %s", depToDelete.Name)
			return
		}
		log.Debugf("restore workflow: deleted primary %s", depToDelete.Name)
	default:
		log.Errorf("restore workflow error: depList has invalid length of %d using selector %s", depLen, selector)
		return
	}

	//update backrest repo with new data path
	targetDepName := task.Spec.Parameters[config.LABEL_BACKREST_RESTORE_TO_PVC]
	err = UpdateDBPath(clientset, &cluster, targetDepName, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not bounce repo with new db path")
		return
	}
	log.Debugf("restore workflow: bounced backrest-repo with new db path")

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
	jobFields := restorejobTemplateFields{
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
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(cluster.Labels[config.LABEL_BACKREST],
			cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], clientset, namespace),
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
	task.Spec.Parameters[status] = time.Now().Format("2006-01-02.15.04.05")
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

//update the PGBACKREST_DB_PATH env var of the backrest-repo
//deployment for a given cluster, the deployment is bounced as
//part of this process
func UpdateDBPath(clientset *kubernetes.Clientset, cluster *crv1.Pgcluster, target, namespace string) error {
	var err error
	newPath := "/pgdata/" + target
	depName := cluster.Name + "-backrest-shared-repo"

	var deployment *appsv1.Deployment
	deployment, err = clientset.AppsV1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
	if err != nil {
		log.Error(err)
		log.Error("error getting deployment in UpdateDBPath using name " + depName)
		return err
	}

	//log.Debugf("replicas %d", *deployment.Spec.Replicas)

	//drain deployment to 0 pods
	*deployment.Spec.Replicas = 0

	containerIndex := -1
	envIndex := -1
	//update the env var Value
	//template->spec->containers->env["PGBACKREST_DB_PATH"]
	for kc, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == "database" {
			log.Debugf(" %s is the container name at %d", c.Name, kc)
			containerIndex = kc
			for ke, e := range c.Env {
				if e.Name == "PGBACKREST_DB_PATH" {
					log.Debugf("PGBACKREST_DB_PATH is %s", e.Value)
					envIndex = ke
				}
			}
		}
	}

	if containerIndex == -1 || envIndex == -1 {
		return errors.New("error in getting container with PGBACRKEST_DB_PATH for cluster " + cluster.Name)
	}

	deployment.Spec.Template.Spec.Containers[containerIndex].Env[envIndex].Value = newPath

	//update the deployment (drain and update the env var)
	err = kubeapi.UpdateDeployment(clientset, deployment, namespace)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	//wait till deployment goes to 0
	//TODO fix this loop to be a proper wait
	var zero bool
	for i := 0; i < 8; i++ {
		deployment, err = clientset.AppsV1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
		if err != nil {
			log.Error("could not get deployment UpdateDBPath " + err.Error())
			return err
		}

		log.Debugf("status replicas %d\n", deployment.Status.Replicas)
		if deployment.Status.Replicas == 0 {
			log.Debugf("deployment %s replicas is now 0", deployment.Name)
			zero = true
			break
		} else {
			log.Debug("UpdateDBPath: sleeping till deployment goes to 0")
			time.Sleep(time.Second * time.Duration(2))
		}
	}
	if !zero {
		return errors.New("deployment replicas never went to 0")
	}
	//update the deployment back to replicas 1
	*deployment.Spec.Replicas = 1
	err = kubeapi.UpdateDeployment(clientset, deployment, namespace)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Debugf("updated PGBACKREST_DB_PATH to %s on deployment %s", newPath, cluster.Name)

	return err
}

func CreateRestoredDeployment(restclient *rest.RESTClient, cluster *crv1.Pgcluster, clientset *kubernetes.Clientset,
	namespace, restoreToName, workflowID string, affinity *v1.Affinity) error {

	var err error

	//primaryLabels := operator.GetPrimaryLabels(cluster.Spec.Name, cluster.Spec.ClusterName, false, cluster.Spec.UserLabels)

	cluster.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = restoreToName
	cluster.Spec.UserLabels["name"] = cluster.Spec.Name
	cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER] = cluster.Spec.ClusterName

	archiveMode := "on"
	xlogdir := "false"
	archiveTimeout := cluster.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]
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
		Name:                    restoreToName,
		Replicas:                "1",
		PgMode:                  "primary",
		ClusterName:             cluster.Spec.Name,
		PrimaryHost:             restoreToName,
		Port:                    cluster.Spec.Port,
		LogStatement:            operator.Pgo.Cluster.LogStatement,
		LogMinDurationStatement: operator.Pgo.Cluster.LogMinDurationStatement,
		CCPImagePrefix:          operator.Pgo.Cluster.CCPImagePrefix,
		CCPImage:                cluster.Spec.CCPImage,
		CCPImageTag:             cluster.Spec.CCPImageTag,
		PVCName:                 util.CreatePVCSnippet(cluster.Spec.PrimaryStorage.StorageType, restoreToName),
		DeploymentLabels:        operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		PodLabels:               operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		DataPathOverride:        restoreToName,
		Database:                cluster.Spec.Database,
		ArchiveMode:             archiveMode,
		ArchivePVCName:          util.CreateBackupPVCSnippet(archivePVCName),
		XLOGDir:                 xlogdir,
		BackrestPVCName:         util.CreateBackrestPVCSnippet(backrestPVCName),
		ArchiveTimeout:          archiveTimeout,
		SecurityContext:         util.CreateSecContext(cluster.Spec.PrimaryStorage.Fsgroup, cluster.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:          cluster.Spec.RootSecretName,
		PrimarySecretName:       cluster.Spec.PrimarySecretName,
		UserSecretName:          cluster.Spec.UserSecretName,
		NodeSelector:            affinityStr,
		ContainerResources:      operator.GetContainerResourcesJSON(&cluster.Spec.ContainerResources),
		ConfVolume:              operator.GetConfVolume(clientset, cluster, namespace),
		CollectAddon:            operator.GetCollectAddon(clientset, namespace, &cluster.Spec),
		BadgerAddon:             operator.GetBadgerAddon(clientset, namespace, cluster, restoreToName),
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cluster.Labels[config.LABEL_BACKREST], cluster.Spec.ClusterName, restoreToName,
			cluster.Spec.Port, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(cluster.Labels[config.LABEL_BACKREST],
			cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], clientset, namespace),
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
