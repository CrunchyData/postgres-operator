package backrest

/*
 Copyright 2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	//"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

// consolidate with cluster.PgbackrestEnvVarsTemplateFields
type PgbackrestEnvVarsTemplateFields struct {
	PgbackrestStanza    string
	PgbackrestDBPath    string
	PgbackrestRepo1Path string
	PgbackrestRepo1Host string
}

// consolidate with cluster.affinityTemplateFields
const AffinityInOperator = "In"
const AFFINITY_NOTINOperator = "NotIn"

type affinityTemplateFields struct {
	NodeLabelKey   string
	NodeLabelValue string
	OperatorValue  string
}

// consolidate
type collectTemplateFields struct {
	Name            string
	JobName         string
	PrimaryPassword string
	CCPImageTag     string
	CCPImagePrefix  string
}

//consolidate
type badgerTemplateFields struct {
	CCPImageTag        string
	CCPImagePrefix     string
	BadgerTarget       string
	ContainerResources string
}

// needs to be consolidated with cluster.DeploymentTemplateFields
// DeploymentTemplateFields ...
type DeploymentTemplateFields struct {
	Name                    string
	ClusterName             string
	Port                    string
	PgMode                  string
	LogStatement            string
	LogMinDurationStatement string
	CCPImagePrefix          string
	CCPImageTag             string
	Database                string
	DeploymentLabels        string
	PodLabels               string
	DataPathOverride        string
	ArchiveMode             string
	ArchivePVCName          string
	ArchiveTimeout          string
	XLOGDir                 string
	BackrestPVCName         string
	PVCName                 string
	BackupPVCName           string
	BackupPath              string
	RootSecretName          string
	UserSecretName          string
	PrimarySecretName       string
	SecurityContext         string
	ContainerResources      string
	NodeSelector            string
	ConfVolume              string
	CollectAddon            string
	BadgerAddon             string
	PgbackrestEnvVars       string
	//next 2 are for the replica deployment only
	Replicas    string
	PrimaryHost string
}

type restorejobTemplateFields struct {
	JobName             string
	ClusterName         string
	WorkflowID          string
	ToClusterPVCName    string
	SecurityContext     string
	COImagePrefix       string
	COImageTag          string
	CommandOpts         string
	PITRTarget          string
	PgbackrestStanza    string
	PgbackrestDBPath    string
	PgbackrestRepo1Path string
	PgbackrestRepo1Host string
}

// Restore ...
func Restore(restclient *rest.RESTClient, namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	clusterName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER]
	log.Debugf("restore workflow: started for cluster %s", clusterName)
	//pvcName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC]

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow error: could not find a pgcluster in Restore Workflow for %s", clusterName)
		return
	}

	//turn off autofail if its on
	if cluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL] == "true" {
		cluster.ObjectMeta.Labels[util.LABEL_AUTOFAIL] = "false"
		log.Debugf("restore workflow: turning off autofail on %s", clusterName)
		err = kubeapi.Updatepgcluster(restclient, &cluster, clusterName, namespace)
		if err != nil {
			log.Errorf("restore workflow error: could not turn off autofail on %s", clusterName)
			return
		}
	}

	//use the storage config from pgo.yaml for Primary
	storage := operator.Pgo.Storage[operator.Pgo.PrimaryStorage]

	//create the "to-cluster" PVC to hold the new data
	var pvcName string
	pvcName, err = createPVC(restclient, namespace, clientset, task)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("restore workflow: created pvc %s for cluster %s", pvcName, clusterName)
	//delete current primary deployment
	selector := util.LABEL_SERVICE_NAME + "=" + clusterName
	var depList *v1beta1.DeploymentList
	depList, err = kubeapi.GetDeployments(clientset, selector, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not get depList using %s", selector)
		return
	}
	if len(depList.Items) != 1 {
		log.Errorf("restore workflow error: depList not equal to 1 %s", selector)
		return
	}

	depToDelete := depList.Items[0]

	err = kubeapi.DeleteDeployment(clientset, depToDelete.Name, namespace)
	if err != nil {
		log.Errorf("restore workflow error: could not delete primary %s", depToDelete.Name)
		return
	}
	log.Debugf("restore workflow: deleted primary %s", depToDelete.Name)

	//update backrest repo with new data path
	targetDepName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC]
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
		JobName:             "backrest-restore-" + task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER] + "-to-" + pvcName,
		ClusterName:         task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_FROM_CLUSTER],
		SecurityContext:     util.CreateSecContext(storage.Fsgroup, storage.SupplementalGroups),
		ToClusterPVCName:    pvcName,
		WorkflowID:          workflowID,
		CommandOpts:         task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_OPTS],
		PITRTarget:          task.Spec.Parameters[util.LABEL_BACKREST_PITR_TARGET],
		COImagePrefix:       operator.Pgo.Pgo.COImagePrefix,
		COImageTag:          operator.Pgo.Pgo.COImageTag,
		PgbackrestStanza:    task.Spec.Parameters[util.LABEL_PGBACKREST_STANZA],
		PgbackrestDBPath:    task.Spec.Parameters[util.LABEL_PGBACKREST_DB_PATH],
		PgbackrestRepo1Path: task.Spec.Parameters[util.LABEL_PGBACKREST_REPO_PATH],
		PgbackrestRepo1Host: task.Spec.Parameters[util.LABEL_PGBACKREST_REPO_HOST],
	}

	var doc2 bytes.Buffer
	err = operator.BackrestRestorejobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		log.Error("restore workflow: error executing job template")
		return
	}

	if operator.CRUNCHY_DEBUG {
		operator.BackrestRestorejobTemplate.Execute(os.Stdout, jobFields)
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

func UpdateRestoreWorkflow(restclient *rest.RESTClient, clientset *kubernetes.Clientset, clusterName, status, namespace, workflowID, restoreToName string) {
	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackrestRestoreType
	log.Debugf("restore workflow phase 2: taskName is %s", taskName)

	cluster := crv1.Pgcluster{}

	found, err := kubeapi.Getpgcluster(restclient, &cluster, clusterName, namespace)
	if !found || err != nil {
		log.Errorf("restore workflow phase 2 error: could not find a pgclustet in updateRestoreWorkflow for %s", clusterName)
		return
	}

	//create the new primary deployment
	CreateRestoredDeployment(restclient, &cluster, clientset, namespace, restoreToName, workflowID)

	log.Debugf("restore workflow phase  2: created restored primary was %s now %s", cluster.Spec.Name, restoreToName)
	//cluster.Spec.Name = restoreToName
	//cluster.ObjectMeta.Labels[util.LABEL_CURRENT_PRIMARY] = restoreToName

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

func createPVC(restclient *rest.RESTClient, namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) (string, error) {
	var err error
	clusterName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC]

	//use the storage config from pgo.yaml for Primary
	storage := operator.Pgo.Storage[operator.Pgo.PrimaryStorage]

	//create the "to-cluster" PVC to hold the new data
	pvcName := task.Spec.Parameters[util.LABEL_BACKREST_RESTORE_TO_PVC]
	pgstoragespec := crv1.PgStorageSpec{}
	pgstoragespec.AccessMode = storage.AccessMode
	pgstoragespec.Size = storage.Size
	pgstoragespec.StorageType = storage.StorageType
	pgstoragespec.StorageClass = storage.StorageClass
	pgstoragespec.Fsgroup = storage.Fsgroup
	pgstoragespec.SupplementalGroups = storage.SupplementalGroups
	pgstoragespec.MatchLabels = storage.MatchLabels

	var found bool
	_, found, err = kubeapi.GetPVC(clientset, pvcName, namespace)
	if !found {
		log.Debugf("pvc %s not found, will create as part of restore", pvcName)
		//delete the pvc if it already exists
		//kubeapi.DeletePVC(clientset, pvcName, namespace)

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

	var deployment *v1beta1.Deployment
	deployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
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
		deployment, err = clientset.ExtensionsV1beta1().Deployments(namespace).Get(depName, meta_v1.GetOptions{})
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

func CreateRestoredDeployment(restclient *rest.RESTClient, cluster *crv1.Pgcluster, clientset *kubernetes.Clientset, namespace, restoreToName, workflowID string) error {

	var err error

	primaryLabels := getPrimaryLabels(cluster.Spec.Name, cluster.Spec.ClusterName, false, cluster.Spec.UserLabels)

	primaryLabels[util.LABEL_DEPLOYMENT_NAME] = restoreToName

	archiveMode := "on"
	xlogdir := "false"
	archiveTimeout := cluster.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
	archivePVCName := cluster.Spec.Name + "-xlog"
	backrestPVCName := cluster.Spec.Name + "-backrestrepo"

	deploymentFields := DeploymentTemplateFields{
		Name:                    restoreToName,
		Replicas:                "1",
		PgMode:                  "primary",
		ClusterName:             cluster.Spec.Name,
		PrimaryHost:             restoreToName,
		Port:                    cluster.Spec.Port,
		LogStatement:            operator.Pgo.Cluster.LogStatement,
		LogMinDurationStatement: operator.Pgo.Cluster.LogMinDurationStatement,
		CCPImagePrefix:          operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:             cluster.Spec.CCPImageTag,
		PVCName:                 util.CreatePVCSnippet(cluster.Spec.PrimaryStorage.StorageType, restoreToName),
		DeploymentLabels:        GetLabelsFromMap(primaryLabels),
		PodLabels:               GetLabelsFromMap(primaryLabels),
		BackupPVCName:           util.CreateBackupPVCSnippet(cluster.Spec.BackupPVCName),
		BackupPath:              cluster.Spec.BackupPath,
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
		NodeSelector:            GetAffinity(cluster.Spec.UserLabels["NodeLabelKey"], cluster.Spec.UserLabels["NodeLabelValue"], "In"),
		ContainerResources:      operator.GetContainerResourcesJSON(&cluster.Spec.ContainerResources),
		ConfVolume:              GetConfVolume(clientset, cluster, namespace),
		CollectAddon:            GetCollectAddon(clientset, namespace, &cluster.Spec),
		BadgerAddon:             GetBadgerAddon(clientset, namespace, &cluster.Spec),
		PgbackrestEnvVars:       GetPgbackrestEnvVars(cluster.Spec.UserLabels[util.LABEL_BACKREST], cluster.Spec.Name, restoreToName),
	}

	log.Debug("collectaddon value is [" + deploymentFields.CollectAddon + "]")
	var primaryDoc bytes.Buffer
	err = operator.DeploymentTemplate1.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	//a form of debugging
	if operator.CRUNCHY_DEBUG {
		operator.DeploymentTemplate1.Execute(os.Stdout, deploymentFields)
	}

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(primaryDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling primary json into Deployment " + err.Error())
		return err
	}
	err = kubeapi.CreateDeployment(clientset, &deployment, namespace)
	if err != nil {
		return err
	}

	primaryLabels[util.LABEL_CURRENT_PRIMARY] = restoreToName

	cluster.Spec.UserLabels[crv1.PgtaskWorkflowID] = workflowID

	err = util.PatchClusterCRD(restclient, primaryLabels, cluster, namespace)
	if err != nil {
		log.Error("could not patch primary crv1 with labels")
		return err
	}
	return err

}

//needs to be consolidated with cluster.GetAffinity
// GetAffinity ...
func GetAffinity(nodeLabelKey, nodeLabelValue string, affoperator string) string {
	log.Debugf("GetAffinity with nodeLabelKey=[%s] nodeLabelKey=[%s] and operator=[%s]\n", nodeLabelKey, nodeLabelValue, affoperator)
	output := ""
	if nodeLabelKey == "" {
		return output
	}

	affinityTemplateFields := affinityTemplateFields{}
	affinityTemplateFields.NodeLabelKey = nodeLabelKey
	affinityTemplateFields.NodeLabelValue = nodeLabelValue
	affinityTemplateFields.OperatorValue = affoperator

	var affinityDoc bytes.Buffer
	err := operator.AffinityTemplate1.Execute(&affinityDoc, affinityTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return output
	}

	if operator.CRUNCHY_DEBUG {
		operator.AffinityTemplate1.Execute(os.Stdout, affinityTemplateFields)
	}

	return affinityDoc.String()
}

//neds to be consolidated with cluster.getPrimaryLabels
// getPrimaryLabels ...
func getPrimaryLabels(Name string, ClusterName string, replicaFlag bool, userLabels map[string]string) map[string]string {
	primaryLabels := make(map[string]string)
	primaryLabels[util.LABEL_PRIMARY] = "true"
	if replicaFlag {
		primaryLabels[util.LABEL_PRIMARY] = "false"
	}

	primaryLabels["name"] = Name
	primaryLabels[util.LABEL_PG_CLUSTER] = ClusterName

	for key, value := range userLabels {
		if key == util.LABEL_AUTOFAIL || key == util.LABEL_NODE_LABEL_KEY || key == util.LABEL_NODE_LABEL_VALUE {
			//dont add these since they can break label expression checks
			//or autofail toggling
		} else {
			primaryLabels[key] = value
		}
	}
	return primaryLabels
}

// needs to be consolidated with cluster.GetLabelsFromMap
// GetLabelsFromMap ...
func GetLabelsFromMap(labels map[string]string) string {
	var output string

	mapLen := len(labels)
	i := 1
	for key, value := range labels {
		if i < mapLen {
			output += fmt.Sprintf("\"" + key + "\": \"" + value + "\",")
		} else {
			output += fmt.Sprintf("\"" + key + "\": \"" + value + "\"")
		}
		i++
	}
	return output
}

//consolidate with cluster.GetConfVolume
func GetConfVolume(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string) string {
	var found bool

	//check for user provided configmap
	if cl.Spec.CustomConfig != "" {
		_, found = kubeapi.GetConfigMap(clientset, cl.Spec.CustomConfig, namespace)
		if !found {
			//you should NOT get this error because of apiserver validation of this value!
			log.Errorf("%s was not found, error, skipping user provided configMap", cl.Spec.CustomConfig)
		} else {
			log.Debugf("user provided configmap %s was used for this cluster", cl.Spec.CustomConfig)
			return "\"configMap\": { \"name\": \"" + cl.Spec.CustomConfig + "\" }"
		}

	}

	//check for global custom configmap "pgo-custom-pg-config"
	_, found = kubeapi.GetConfigMap(clientset, util.GLOBAL_CUSTOM_CONFIGMAP, namespace)
	if !found {
		log.Debug(util.GLOBAL_CUSTOM_CONFIGMAP + " was not found, , skipping global configMap")
	} else {
		return "\"configMap\": { \"name\": \"pgo-custom-pg-config\" }"
	}

	//the default situation
	return "\"emptyDir\": { \"medium\": \"Memory\" }"
}

//consolidate with cluster.GetCollectAddon
func GetCollectAddon(clientset *kubernetes.Clientset, namespace string, spec *crv1.PgclusterSpec) string {

	if spec.UserLabels[util.LABEL_COLLECT] == "true" {
		log.Debug("crunchy_collect was found as a label on cluster create")
		_, PrimaryPassword, err3 := util.GetPasswordFromSecret(clientset, namespace, spec.PrimarySecretName)
		if err3 != nil {
			log.Error(err3)
		}

		collectTemplateFields := collectTemplateFields{}
		collectTemplateFields.Name = spec.Name
		collectTemplateFields.JobName = spec.Name
		collectTemplateFields.PrimaryPassword = PrimaryPassword
		collectTemplateFields.CCPImageTag = spec.CCPImageTag
		collectTemplateFields.CCPImagePrefix = operator.Pgo.Cluster.CCPImagePrefix

		var collectDoc bytes.Buffer
		err := operator.CollectTemplate1.Execute(&collectDoc, collectTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if operator.CRUNCHY_DEBUG {
			operator.CollectTemplate1.Execute(os.Stdout, collectTemplateFields)
		}
		return collectDoc.String()
	}
	return ""
}

//consolidate with cluster.GetBadgerAddon
func GetBadgerAddon(clientset *kubernetes.Clientset, namespace string, spec *crv1.PgclusterSpec) string {

	if spec.UserLabels[util.LABEL_BADGER] == "true" {
		log.Debug("crunchy_badger was found as a label on cluster create")
		badgerTemplateFields := badgerTemplateFields{}
		badgerTemplateFields.CCPImageTag = spec.CCPImageTag
		badgerTemplateFields.BadgerTarget = spec.Name
		badgerTemplateFields.CCPImagePrefix = operator.Pgo.Cluster.CCPImagePrefix
		badgerTemplateFields.ContainerResources = ""

		if operator.Pgo.DefaultBadgerResources != "" {
			tmp, err := operator.Pgo.GetContainerResource(operator.Pgo.DefaultBadgerResources)
			if err != nil {
				log.Error(err)
				return ""
			}
			badgerTemplateFields.ContainerResources = operator.GetContainerResourcesJSON(&tmp)

		}

		var badgerDoc bytes.Buffer
		err := operator.BadgerTemplate1.Execute(&badgerDoc, badgerTemplateFields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}

		if operator.CRUNCHY_DEBUG {
			operator.BadgerTemplate1.Execute(os.Stdout, badgerTemplateFields)
		}
		return badgerDoc.String()
	}
	return ""
}

//consolidate with cluster.GetPgbackrestEnvVars
func GetPgbackrestEnvVars(backrestEnabled, clusterName, depName string) string {
	if backrestEnabled == "true" {
		fields := PgbackrestEnvVarsTemplateFields{
			PgbackrestStanza:    "db",
			PgbackrestRepo1Host: clusterName + "-backrest-shared-repo",
			PgbackrestRepo1Path: "/backrestrepo/" + clusterName + "-backrest-shared-repo",
			PgbackrestDBPath:    "/pgdata/" + depName,
		}

		var doc bytes.Buffer
		err := operator.PgbackrestEnvVarsTemplate.Execute(&doc, fields)
		if err != nil {
			log.Error(err.Error())
			return ""
		}
		return doc.String()
	}
	return ""

}
