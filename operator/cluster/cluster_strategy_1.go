// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/util"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

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
		ServiceName: cl.Spec.Name,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
		ServiceType: st,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error("error in creating primary service " + err.Error())
		return err
	}

	primaryLabels := operator.GetPrimaryLabels(cl.Spec.Name, cl.Spec.ClusterName, false, cl.Spec.UserLabels)

	archivePVCName := ""
	archiveMode := "off"
	archiveTimeout := "60"
	xlogdir := "false"
	if cl.Spec.UserLabels[util.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
		archiveTimeout = cl.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = cl.Spec.Name + "-xlog"
		//xlogdir = "true"
	}

	if cl.Spec.UserLabels[util.LABEL_BACKREST] == "true" {
		//backrest requires us to turn on archive mode
		archiveMode = "on"
		archiveTimeout = cl.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = cl.Spec.Name + "-xlog"
		xlogdir = "false"
		err = backrest.CreateRepoDeployment(clientset, namespace, cl)
		if err != nil {
			log.Error("could not create backrest repo deployment")
			return err
		}
	}

	primaryLabels[util.LABEL_DEPLOYMENT_NAME] = cl.Spec.Name

	//create the primary deployment
	deploymentFields := operator.DeploymentTemplateFields{
		Name:                    cl.Spec.Name,
		Replicas:                "1",
		PgMode:                  "primary",
		ClusterName:             cl.Spec.Name,
		PrimaryHost:             cl.Spec.Name,
		Port:                    cl.Spec.Port,
		LogStatement:            operator.Pgo.Cluster.LogStatement,
		LogMinDurationStatement: operator.Pgo.Cluster.LogMinDurationStatement,
		CCPImagePrefix:          operator.Pgo.Cluster.CCPImagePrefix,
		CCPImage:                cl.Spec.CCPImage,
		CCPImageTag:             cl.Spec.CCPImageTag,
		PVCName:                 util.CreatePVCSnippet(cl.Spec.PrimaryStorage.StorageType, primaryPVCName),
		DeploymentLabels:        operator.GetLabelsFromMap(primaryLabels),
		PodLabels:               operator.GetLabelsFromMap(primaryLabels),
		BackupPVCName:           util.CreateBackupPVCSnippet(cl.Spec.BackupPVCName),
		BackupPath:              cl.Spec.BackupPath,
		DataPathOverride:        cl.Spec.Name,
		Database:                cl.Spec.Database,
		ArchiveMode:             archiveMode,
		ArchivePVCName:          util.CreateBackupPVCSnippet(archivePVCName),
		XLOGDir:                 xlogdir,
		ArchiveTimeout:          archiveTimeout,
		SecurityContext:         util.CreateSecContext(cl.Spec.PrimaryStorage.Fsgroup, cl.Spec.PrimaryStorage.SupplementalGroups),
		RootSecretName:          cl.Spec.RootSecretName,
		PrimarySecretName:       cl.Spec.PrimarySecretName,
		UserSecretName:          cl.Spec.UserSecretName,
		NodeSelector:            operator.GetAffinity(cl.Spec.UserLabels["NodeLabelKey"], cl.Spec.UserLabels["NodeLabelValue"], "In"),
		ContainerResources:      operator.GetContainerResourcesJSON(&cl.Spec.ContainerResources),
		ConfVolume:              operator.GetConfVolume(clientset, cl, namespace),
		CollectAddon:            operator.GetCollectAddon(clientset, namespace, &cl.Spec),
		BadgerAddon:             operator.GetBadgerAddon(clientset, namespace, &cl.Spec),
		PgbackrestEnvVars:       operator.GetPgbackrestEnvVars(cl.Spec.UserLabels[util.LABEL_BACKREST], cl.Spec.Name, cl.Spec.Name, cl.Spec.Port),
		PgmonitorEnvVars:        operator.GetPgmonitorEnvVars(cl.Spec.UserLabels[util.LABEL_COLLECT]),
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

	primaryLabels[util.LABEL_CURRENT_PRIMARY] = cl.Spec.Name

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

	//delete the pgbouncer service if exists
	if cl.Spec.UserLabels[util.LABEL_PGBOUNCER] == "true" {
		DeletePgbouncer(clientset, cl.Spec.Name, namespace)
	}

	//delete the primary service
	kubeapi.DeleteService(clientset, cl.Spec.Name, namespace)

	//delete the replica service
	var found bool
	_, found, err = kubeapi.GetService(clientset, cl.Spec.Name+ReplicaSuffix, namespace)
	if found {
		kubeapi.DeleteService(clientset, cl.Spec.Name+ReplicaSuffix, namespace)
	}

	//delete the pgpool deployment if necessary
	if cl.Spec.UserLabels[util.LABEL_PGPOOL] == "true" {
		DeletePgpool(clientset, cl.Spec.Name, namespace)
	}

	//delete the backrest repo deployment if necessary
	if cl.Spec.UserLabels[util.LABEL_BACKREST] == "true" {
		deleteBackrestRepo(clientset, cl.Spec.Name, namespace)
	}

	//delete the pgreplicas if necessary
	DeletePgreplicas(restclient, cl.Spec.Name, namespace)

	//delete the pgbackups if necessary
	pgback := crv1.Pgbackup{}
	found, err = kubeapi.Getpgbackup(restclient, &pgback, cl.Spec.Name, namespace)
	if found {
		kubeapi.Deletepgbackup(restclient, cl.Spec.Name, namespace)
	}

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

// Scale ...
func (r Strategy1) Scale(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace, pvcName string, cluster *crv1.Pgcluster) error {
	var err error
	log.Debug("Scale called for " + replica.Name)
	log.Debug("Scale called pvcName " + pvcName)
	log.Debug("Scale called namespace " + namespace)

	var replicaDoc bytes.Buffer

	serviceName := replica.Spec.ClusterName + "-replica"
	replicaFlag := true

	replicaLabels := operator.GetPrimaryLabels(serviceName, replica.Spec.ClusterName, replicaFlag, cluster.Spec.UserLabels)
	replicaLabels[util.LABEL_REPLICA_NAME] = replica.Spec.Name

	archivePVCName := ""
	archiveMode := "off"
	archiveTimeout := "60"
	xlogdir := "false"
	if cluster.Spec.UserLabels[util.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
		archiveTimeout = cluster.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = replica.Spec.Name + "-xlog"
		//	xlogdir = "true"
	}

	if cluster.Spec.UserLabels[util.LABEL_BACKREST] == "true" {
		//backrest requires archive mode be set to on
		archiveMode = "on"
		archiveTimeout = cluster.Spec.UserLabels[util.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = replica.Spec.Name + "-xlog"
		xlogdir = "false"
	}

	image := cluster.Spec.CCPImage

	//check for --ccp-image-tag at the command line
	imageTag := cluster.Spec.CCPImageTag
	if replica.Spec.UserLabels[util.LABEL_CCP_IMAGE_TAG_KEY] != "" {
		imageTag = replica.Spec.UserLabels[util.LABEL_CCP_IMAGE_TAG_KEY]
	}

	//allow the user to override the replica resources
	cs := replica.Spec.ContainerResources
	if replica.Spec.ContainerResources.LimitsCPU == "" {
		cs = cluster.Spec.ContainerResources
	}

	replicaLabels[util.LABEL_DEPLOYMENT_NAME] = replica.Spec.Name

	//create the replica deployment
	replicaDeploymentFields := operator.DeploymentTemplateFields{
		Name:                    replica.Spec.Name,
		ClusterName:             replica.Spec.ClusterName,
		PgMode:                  "replica",
		Port:                    cluster.Spec.Port,
		CCPImagePrefix:          operator.Pgo.Cluster.CCPImagePrefix,
		LogStatement:            operator.Pgo.Cluster.LogStatement,
		LogMinDurationStatement: operator.Pgo.Cluster.LogMinDurationStatement,
		CCPImageTag:             imageTag,
		CCPImage:                image,
		PVCName:                 util.CreatePVCSnippet(cluster.Spec.ReplicaStorage.StorageType, pvcName),
		BackupPVCName:           util.CreateBackupPVCSnippet(cluster.Spec.BackupPVCName),
		PrimaryHost:             cluster.Spec.PrimaryHost,
		BackupPath:              "",
		Database:                cluster.Spec.Database,
		DataPathOverride:        replica.Spec.Name,
		ArchiveMode:             archiveMode,
		ArchivePVCName:          util.CreateBackupPVCSnippet(archivePVCName),
		XLOGDir:                 xlogdir,
		ArchiveTimeout:          archiveTimeout,
		Replicas:                "1",
		ConfVolume:              operator.GetConfVolume(clientset, cluster, namespace),
		DeploymentLabels:        operator.GetLabelsFromMap(replicaLabels),
		PodLabels:               operator.GetLabelsFromMap(replicaLabels),
		SecurityContext:         util.CreateSecContext(replica.Spec.ReplicaStorage.Fsgroup, replica.Spec.ReplicaStorage.SupplementalGroups),
		RootSecretName:          cluster.Spec.RootSecretName,
		PrimarySecretName:       cluster.Spec.PrimarySecretName,
		UserSecretName:          cluster.Spec.UserSecretName,
		ContainerResources:      operator.GetContainerResourcesJSON(&cs),
		NodeSelector:            operator.GetReplicaAffinity(cluster.Spec.UserLabels, replica.Spec.UserLabels),
		CollectAddon:            operator.GetCollectAddon(clientset, namespace, &cluster.Spec),
		BadgerAddon:             operator.GetBadgerAddon(clientset, namespace, &cluster.Spec),
		PgbackrestEnvVars:       operator.GetPgbackrestEnvVars(cluster.Spec.UserLabels[util.LABEL_BACKREST], replica.Spec.ClusterName, replica.Spec.Name, cluster.Spec.Port),
		PgmonitorEnvVars:        operator.GetPgmonitorEnvVars(cluster.Spec.UserLabels[util.LABEL_COLLECT]),
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

//delete the backrest repo deployment best effort
func deleteBackrestRepo(clientset *kubernetes.Clientset, clusterName, namespace string) error {
	var err error

	depName := clusterName + "-backrest-shared-repo"
	log.Debugf("deleting the backrest repo deployment and service %s", depName)

	err = kubeapi.DeleteDeployment(clientset, depName, namespace)

	//delete the service for the backrest repo
	err = kubeapi.DeleteService(clientset, depName, namespace)

	return err

}
