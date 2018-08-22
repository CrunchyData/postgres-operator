// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

const AffinityInOperator = "In"
const AFFINITY_NOTINOperator = "NotIn"

type affinityTemplateFields struct {
	NodeLabelKey   string
	NodeLabelValue string
	OperatorValue  string
}

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

type collectTemplateFields struct {
	Name            string
	JobName         string
	PrimaryPassword string
	CCPImageTag     string
	CCPImagePrefix  string
}
type badgerTemplateFields struct {
	CCPImageTag    string
	CCPImagePrefix string
	BadgerTarget   string
}

// Strategy1  ...
type Strategy1 struct{}

// AddCluster ...
func (r Strategy1) AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string, primaryPVCName string) error {
	var primaryDoc bytes.Buffer
	var err error

	log.Info("creating Pgcluster object using Strategy 1" + " in namespace " + namespace)
	log.Info("created with Name=" + cl.Spec.Name + " in namespace " + namespace)

	st := operator.Pgo.Cluster.ServiceType
	if cl.Spec.UserLabels[util.LABEL_SERVICE_TYPE] != "" {
		st = cl.Spec.UserLabels[util.LABEL_SERVICE_TYPE]
	}

	//create the primary service
	serviceFields := ServiceTemplateFields{
		Name:        cl.Spec.Name,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
		ServiceType: st,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error("error in creating primary service " + err.Error())
		return err
	}

	primaryLabels := getPrimaryLabels(cl.Spec.Name, cl.Spec.ClusterName, false, cl.Spec.UserLabels)

	archivePVCName := ""
	archiveMode := "off"
	archiveTimeout := "60"
	if cl.Spec.UserLabels[util.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
		archiveTimeout = cl.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = cl.Spec.Name + "-xlog"
	}

	backrestPVCName := ""
	if cl.Spec.UserLabels[util.LABEL_BACKREST] == "true" {
		backrestPVCName = cl.Spec.Name + "-backrestrepo"
		//backrest requires us to turn on archive mode
		archiveMode = "on"
		archiveTimeout = cl.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = cl.Spec.Name + "-xlog"
	}

	//create the primary deployment
	deploymentFields := DeploymentTemplateFields{
		Name:               cl.Spec.Name,
		Replicas:           "1",
		PgMode:             "primary",
		ClusterName:        cl.Spec.Name,
		PrimaryHost:        cl.Spec.Name,
		Port:               cl.Spec.Port,
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        cl.Spec.CCPImageTag,
		PVCName:            util.CreatePVCSnippet(cl.Spec.PrimaryStorage.StorageType, primaryPVCName),
		OperatorLabels:     util.GetLabelsFromMap(primaryLabels),
		BackupPVCName:      util.CreateBackupPVCSnippet(cl.Spec.BackupPVCName),
		BackupPath:         cl.Spec.BackupPath,
		DataPathOverride:   cl.Spec.Name,
		Database:           cl.Spec.Database,
		ArchiveMode:        archiveMode,
		ArchivePVCName:     util.CreateBackupPVCSnippet(archivePVCName),
		BackrestPVCName:    util.CreateBackrestPVCSnippet(backrestPVCName),
		ArchiveTimeout:     archiveTimeout,
		SecurityContext:    util.CreateSecContext(cl.Spec.PrimaryStorage.Fsgroup, cl.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:     cl.Spec.RootSecretName,
		PrimarySecretName:  cl.Spec.PrimarySecretName,
		UserSecretName:     cl.Spec.UserSecretName,
		NodeSelector:       GetAffinity(cl.Spec.UserLabels["NodeLabelKey"], cl.Spec.UserLabels["NodeLabelValue"], "In"),
		ContainerResources: GetContainerResources(&cl.Spec.ContainerResources),
		ConfVolume:         GetConfVolume(clientset, cl.Spec.CustomConfig, namespace),
		CollectAddon:       GetCollectAddon(clientset, namespace, &cl.Spec),
		BadgerAddon:        GetBadgerAddon(clientset, namespace, &cl.Spec),
	}

	log.Debug("collectaddon value is [" + deploymentFields.CollectAddon + "]")
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

	if deploymentExists(clientset, namespace, cl.Spec.Name) == false {
		err = kubeapi.CreateDeployment(clientset, &deployment, namespace)
		if err != nil {
			return err
		}
	} else {
		log.Info("primary Deployment " + cl.Spec.Name + " in namespace " + namespace + " already existed so not creating it ")
	}

	err = util.PatchClusterCRD(client, primaryLabels, cl, namespace)
	if err != nil {
		log.Error("could not patch primary crv1 with labels")
		return err
	}

	return err

}

// DeleteCluster ...
func (r Strategy1) DeleteCluster(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {

	var err error
	log.Info("deleting Pgcluster object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//delete the primary and replica deployments and replica sets
	err = shutdownCluster(clientset, restclient, cl, namespace)
	if err != nil {
		log.Error("error deleting primary Deployment " + err.Error())
	}

	//delete the primary service
	kubeapi.DeleteService(clientset, cl.Spec.Name, namespace)

	//delete the replica service
	kubeapi.DeleteService(clientset, cl.Spec.Name+ReplicaSuffix, namespace)

	//delete the pgpool deployment if necessary
	if cl.Spec.UserLabels["crunchy-pgpool"] == "true" {
		DeletePgpool(clientset, cl.Spec.Name, namespace)
	}

	//delete the pgreplicas if necessary
	DeletePgreplicas(restclient, cl.Spec.Name, namespace)

	//delete the pgbackups if necessary
	kubeapi.Deletepgbackup(restclient, cl.Spec.Name, namespace)

	return err

}

// shutdownCluster ...
func shutdownCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {
	var err error

	deployments, err := kubeapi.GetDeployments(clientset,
		util.LABEL_PG_CLUSTER+"="+cl.Spec.Name, namespace)
	if err != nil {
		return err
	}

	for _, d := range deployments.Items {
		err = kubeapi.DeleteDeployment(clientset, d.ObjectMeta.Name, namespace)
	}

	return err

}

// deploymentExists ...
func deploymentExists(clientset *kubernetes.Clientset, namespace, clusterName string) bool {

	_, found, _ := kubeapi.GetDeployment(clientset, clusterName, namespace)
	return found
}

// UpdatePolicyLabels ...
func (r Strategy1) UpdatePolicyLabels(clientset *kubernetes.Clientset, clusterName string, namespace string, newLabels map[string]string) error {

	deployment, found, err := kubeapi.GetDeployment(clientset, clusterName, namespace)
	if !found {
		return err
	}

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	//update the deployment labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	createdPatch := err == nil
	if err != nil {
		return err
	}
	if createdPatch {
		log.Debug("created merge patch")
	}

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(clusterName, types.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}

// CreateReplica ...
func (r Strategy1) CreateReplica(serviceName string, clientset *kubernetes.Clientset, cl *crv1.Pgcluster, depName, pvcName, namespace string) error {
	var replicaDoc bytes.Buffer
	var err error

	clusterName := cl.Spec.ClusterName

	replicaLabels := getPrimaryLabels(serviceName, clusterName, true, cl.Spec.UserLabels)

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:               depName,
		ClusterName:        clusterName,
		PgMode:             "replica",
		Port:               cl.Spec.Port,
		CCPImagePrefix:     operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:        cl.Spec.CCPImageTag,
		PVCName:            util.CreatePVCSnippet(cl.Spec.ReplicaStorage.StorageType, pvcName),
		BackupPVCName:      util.CreateBackupPVCSnippet(cl.Spec.BackupPVCName),
		DataPathOverride:   depName,
		PrimaryHost:        cl.Spec.PrimaryHost,
		BackupPath:         "",
		Database:           cl.Spec.Database,
		Replicas:           "1",
		ConfVolume:         GetConfVolume(clientset, cl.Spec.CustomConfig, namespace),
		OperatorLabels:     util.GetLabelsFromMap(replicaLabels),
		SecurityContext:    util.CreateSecContext(cl.Spec.ReplicaStorage.Fsgroup, cl.Spec.ReplicaStorage.SupplementalGroups),
		RootSecretName:     cl.Spec.RootSecretName,
		PrimarySecretName:  cl.Spec.PrimarySecretName,
		ContainerResources: GetContainerResources(&cl.Spec.ContainerResources),
		UserSecretName:     cl.Spec.UserSecretName,
		NodeSelector:       GetAffinity(cl.Spec.UserLabels["NodeLabelKey"], cl.Spec.UserLabels["NodeLabelValue"], "NotIn"),
	}

	switch cl.Spec.ReplicaStorage.StorageType {
	case "", "emptydir":
		log.Debug("PrimaryStorage.StorageType is emptydir")
		err = operator.DeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	case "existing", "create", "dynamic":
		log.Debug("using the shared replica template ")
		err = operator.DeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	}

	if err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		operator.DeploymentTemplate1.Execute(os.Stdout, replicaDeploymentFields)
	}

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling replica json into Deployment " + err.Error())
		return err
	}

	err = kubeapi.CreateDeployment(clientset, &replicaDeployment, namespace)
	return err
}

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
		if key == util.LABEL_NODE_LABEL_KEY || key == util.LABEL_NODE_LABEL_VALUE {
			//dont add these since they can break label expression checks
		} else {
			primaryLabels[key] = value
		}
	}
	return primaryLabels
}

// GetReplicaAffinity ...
// use NotIn as an operator when a node-label is not specified on the
// replica, use the node labels from the primary in this case
// use In as an operator when a node-label is specified on the replica
// use the node labels from the replica in this case
func GetReplicaAffinity(clusterLabels, replicaLabels map[string]string) string {
	var operator, key, value string
	log.Debug("GetReplicaAffinity ")
	if replicaLabels[util.LABEL_NODE_LABEL_KEY] != "" {
		//use the replica labels
		operator = "In"
		key = replicaLabels[util.LABEL_NODE_LABEL_KEY]
		value = replicaLabels[util.LABEL_NODE_LABEL_VALUE]
	} else {
		//use the cluster labels
		operator = "NotIn"
		key = clusterLabels[util.LABEL_NODE_LABEL_KEY]
		value = clusterLabels[util.LABEL_NODE_LABEL_VALUE]
	}
	return GetAffinity(key, value, operator)
}

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

// GetConfVolume ...
func GetConfVolume(clientset *kubernetes.Clientset, customConfig, namespace string) string {
	var found bool

	//check for user provided configmap
	if customConfig != "" {
		_, found = kubeapi.GetConfigMap(clientset, customConfig, namespace)
		if !found {
			//you should NOT get this error because of apiserver validation of this value!
			log.Error(customConfig + " was not found, error, skipping user provided configMap")
		}
		return "\"configMap\": { \"name\": \"" + customConfig + "\" }"

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

// GetContainerResources ...
func GetContainerResources(resources *crv1.PgContainerResources) string {

	//test for the case where no container resources are specified
	if resources.RequestsMemory == "" || resources.RequestsCPU == "" ||
		resources.LimitsMemory == "" || resources.LimitsCPU == "" {
		return ""
	}
	fields := containerResourcesTemplateFields{}
	fields.RequestsMemory = resources.RequestsMemory
	fields.RequestsCPU = resources.RequestsCPU
	fields.LimitsMemory = resources.LimitsMemory
	fields.LimitsCPU = resources.LimitsCPU

	var doc bytes.Buffer
	err := operator.ContainerResourcesTemplate1.Execute(&doc, fields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	if operator.CRUNCHY_DEBUG {
		operator.ContainerResourcesTemplate1.Execute(os.Stdout, fields)
	}

	return doc.String()
}

// Scale ...
func (r Strategy1) Scale(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace, pvcName string, cluster *crv1.Pgcluster) error {
	var err error
	log.Debug("Scale called for " + replica.Name)
	log.Debug("Scale called pvcName " + pvcName)
	log.Debug("Scale called namespace " + namespace)

	var replicaDoc bytes.Buffer

	serviceName := replica.Spec.ClusterName + "-replica"
	replicaFlag := true

	replicaLabels := getPrimaryLabels(serviceName, replica.Spec.ClusterName, replicaFlag, cluster.Spec.UserLabels)
	replicaLabels[util.LABEL_REPLICA_NAME] = replica.Spec.Name

	archivePVCName := ""
	archiveMode := "off"
	archiveTimeout := "60"
	if cluster.Spec.UserLabels[util.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
		archiveTimeout = cluster.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = replica.Spec.Name + "-xlog"
	}

	backrestPVCName := ""
	if cluster.Spec.UserLabels[util.LABEL_BACKREST] == "true" {
		backrestPVCName = replica.Spec.Name + "-backrestrepo"
		//backrest requires archive mode be set to on
		archiveMode = "on"
		archiveTimeout = cluster.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = cluster.Spec.Name + "-xlog"
	}

	//check for --ccp-image-tag at the command line
	imageTag := cluster.Spec.CCPImageTag
	if replica.Spec.UserLabels[util.LABEL_CCP_IMAGE_TAG_KEY] != "" {
		imageTag = replica.Spec.UserLabels[util.LABEL_CCP_IMAGE_TAG_KEY]
	}

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:              replica.Spec.Name,
		ClusterName:       replica.Spec.ClusterName,
		PgMode:            "replica",
		Port:              cluster.Spec.Port,
		CCPImagePrefix:    operator.Pgo.Cluster.CCPImagePrefix,
		CCPImageTag:       imageTag,
		PVCName:           util.CreatePVCSnippet(cluster.Spec.ReplicaStorage.StorageType, pvcName),
		BackupPVCName:     util.CreateBackupPVCSnippet(cluster.Spec.BackupPVCName),
		PrimaryHost:       cluster.Spec.PrimaryHost,
		BackupPath:        "",
		Database:          cluster.Spec.Database,
		DataPathOverride:  replica.Spec.Name,
		ArchiveMode:       archiveMode,
		ArchivePVCName:    util.CreateBackupPVCSnippet(archivePVCName),
		BackrestPVCName:   util.CreateBackrestPVCSnippet(backrestPVCName),
		ArchiveTimeout:    archiveTimeout,
		Replicas:          "1",
		ConfVolume:        GetConfVolume(clientset, cluster.Spec.CustomConfig, namespace),
		OperatorLabels:    util.GetLabelsFromMap(replicaLabels),
		SecurityContext:   util.CreateSecContext(replica.Spec.ReplicaStorage.Fsgroup, replica.Spec.ReplicaStorage.SupplementalGroups),
		RootSecretName:    cluster.Spec.RootSecretName,
		PrimarySecretName: cluster.Spec.PrimarySecretName,
		UserSecretName:    cluster.Spec.UserSecretName,
		NodeSelector:      GetReplicaAffinity(cluster.Spec.UserLabels, replica.Spec.UserLabels),
		CollectAddon:      GetCollectAddon(clientset, namespace, &cluster.Spec),
		BadgerAddon:       GetBadgerAddon(clientset, namespace, &cluster.Spec),
	}

	switch replica.Spec.ReplicaStorage.StorageType {
	case "", "emptydir":
		log.Debug("PrimaryStorage.StorageType is emptydir")
		err = operator.DeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	case "existing", "create", "dynamic":
		log.Debug("using the shared replica template ")
		err = operator.DeploymentTemplate1.Execute(&replicaDoc, replicaDeploymentFields)
	}

	if err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		operator.DeploymentTemplate1.Execute(os.Stdout, replicaDeploymentFields)
	}

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling replica json into Deployment " + err.Error())
		return err
	}

	err = kubeapi.CreateDeployment(clientset, &replicaDeployment, namespace)

	return err
}

// DeleteReplica ...
func (r Strategy1) DeleteReplica(clientset *kubernetes.Clientset, cl *crv1.Pgreplica, namespace string) error {

	var err error
	log.Info("deleting Pgreplica object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)
	err = kubeapi.DeleteDeployment(clientset, cl.Spec.Name, namespace)

	return err

}

func GetBadgerAddon(clientset *kubernetes.Clientset, namespace string, spec *crv1.PgclusterSpec) string {

	if spec.UserLabels[util.LABEL_BADGER] == "true" {
		log.Debug("crunchy_badger was found as a label on cluster create")
		badgerTemplateFields := badgerTemplateFields{}
		badgerTemplateFields.CCPImageTag = spec.CCPImageTag
		badgerTemplateFields.BadgerTarget = spec.Name
		badgerTemplateFields.CCPImagePrefix = operator.Pgo.Cluster.CCPImagePrefix

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
