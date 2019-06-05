// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"os"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/util"

	//jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"

	//"k8s.io/apimachinery/pkg/api/meta"
	//"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AddCluster ...
func AddCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string, primaryPVCName string) error {
	var primaryDoc bytes.Buffer
	var err error

	log.Info("creating Pgcluster object  in namespace " + namespace)
	log.Info("created with Name=" + cl.Spec.Name + " in namespace " + namespace)

	st := operator.Pgo.Cluster.ServiceType
	if cl.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = cl.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
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

	//refactor section
	//primaryLabels := make(map[string]string)
	//primaryLabels := operator.GetPrimaryLabels(cl.Spec.Name, cl.Spec.ClusterName, false, cl.Spec.UserLabels)
	cl.Spec.UserLabels["name"] = cl.Spec.Name
	cl.Spec.UserLabels[config.LABEL_PG_CLUSTER] = cl.Spec.ClusterName
	//end refactor section

	archivePVCName := ""
	archiveMode := "off"
	archiveTimeout := "60"
	xlogdir := "false"
	if cl.Spec.UserLabels[config.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
		archiveTimeout = cl.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = cl.Spec.Name + "-xlog"
	}

	if cl.Labels[config.LABEL_BACKREST] == "true" {
		//backrest requires us to turn on archive mode
		archiveMode = "on"
		archiveTimeout = cl.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]
		//archivePVCName = cl.Spec.Name + "-xlog"
		//backrest doesn't use xlog, so we make the pvc an emptydir
		//by setting the name to empty string
		archivePVCName = ""
		xlogdir = "false"
		err = backrest.CreateRepoDeployment(clientset, namespace, cl)
		if err != nil {
			log.Error("could not create backrest repo deployment")
			return err
		}
	}

	cl.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = cl.Spec.Name

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
		DeploymentLabels:        operator.GetLabelsFromMap(cl.Spec.UserLabels),
		PodLabels:               operator.GetLabelsFromMap(cl.Spec.UserLabels),
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
		BadgerAddon:             operator.GetBadgerAddon(clientset, namespace, cl, cl.Spec.Name),
		PgmonitorEnvVars:        operator.GetPgmonitorEnvVars(cl.Spec.UserLabels[config.LABEL_COLLECT]),
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cl.Labels[config.LABEL_BACKREST], cl.Spec.ClusterName, cl.Spec.Name,
			cl.Spec.Port, cl.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(cl.Labels[config.LABEL_BACKREST],
			cl.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], clientset, namespace),
	}

	log.Debug("collectaddon value is [" + deploymentFields.CollectAddon + "]")
	err = config.DeploymentTemplate.Execute(&primaryDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	//a form of debugging
	if operator.CRUNCHY_DEBUG {
		config.DeploymentTemplate.Execute(os.Stdout, deploymentFields)
	}

	deployment := v1.Deployment{}
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

	cl.Spec.UserLabels[config.LABEL_CURRENT_PRIMARY] = cl.Spec.Name

	err = util.PatchClusterCRD(client, cl.Spec.UserLabels, cl, namespace)
	if err != nil {
		log.Error("could not patch primary crv1 with labels")
		return err
	}

	return err

}

// DeleteCluster ...
func DeleteCluster(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {

	var err error
	log.Info("deleting Pgcluster object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)

	//delete the primary and replica deployments and replica sets
	err = shutdownCluster(clientset, restclient, cl, namespace)
	if err != nil {
		log.Error("error deleting primary Deployment " + err.Error())
	}

	//delete the pgbouncer service if exists
	//	if cl.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true" {
	if cl.Labels[config.LABEL_PGBOUNCER] == "true" {
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
	if cl.Spec.UserLabels[config.LABEL_PGPOOL] == "true" {
		DeletePgpool(clientset, cl.Spec.Name, namespace)
	}

	//delete the backrest repo deployment if necessary
	if cl.Labels[config.LABEL_BACKREST] == "true" {
		deleteBackrestRepo(clientset, cl.Spec.Name, namespace)
	}

	//delete the pgreplicas if necessary
	DeletePgreplicas(restclient, cl.Spec.Name, namespace)

	//delete any pgtasks for this cluster
	deletePgtasks(restclient, cl.Spec.Name, namespace)
	return err

}

// shutdownCluster ...
func shutdownCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) error {
	var err error

	deployments, err := kubeapi.GetDeployments(clientset,
		config.LABEL_PG_CLUSTER+"="+cl.Spec.Name, namespace)
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

// Scale ...
func Scale(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace, pvcName string, cluster *crv1.Pgcluster) error {
	var err error
	log.Debug("Scale called for " + replica.Name)
	log.Debug("Scale called pvcName " + pvcName)
	log.Debug("Scale called namespace " + namespace)

	var replicaDoc bytes.Buffer

	serviceName := replica.Spec.ClusterName + "-replica"
	//replicaFlag := true

	//	replicaLabels := operator.GetPrimaryLabels(serviceName, replica.Spec.ClusterName, replicaFlag, cluster.Spec.UserLabels)
	cluster.Spec.UserLabels[config.LABEL_REPLICA_NAME] = replica.Spec.Name
	cluster.Spec.UserLabels["name"] = serviceName
	cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER] = replica.Spec.ClusterName

	archivePVCName := ""
	archiveMode := "off"
	archiveTimeout := "60"
	xlogdir := "false"
	if cluster.Spec.UserLabels[config.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
		archiveTimeout = cluster.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]
		archivePVCName = replica.Spec.Name + "-xlog"
	}

	if cluster.Labels[config.LABEL_BACKREST] == "true" {
		//backrest requires archive mode be set to on
		archiveMode = "on"
		archiveTimeout = cluster.Spec.UserLabels[config.LABEL_ARCHIVE_TIMEOUT]
		//set to emptystring to force emptyDir to be used
		archivePVCName = ""
		xlogdir = "false"
	}

	image := cluster.Spec.CCPImage

	//check for --ccp-image-tag at the command line
	imageTag := cluster.Spec.CCPImageTag
	if replica.Spec.UserLabels[config.LABEL_CCP_IMAGE_TAG_KEY] != "" {
		imageTag = replica.Spec.UserLabels[config.LABEL_CCP_IMAGE_TAG_KEY]
	}

	//allow the user to override the replica resources
	cs := replica.Spec.ContainerResources
	if replica.Spec.ContainerResources.LimitsCPU == "" {
		cs = cluster.Spec.ContainerResources
	}

	cluster.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = replica.Spec.Name

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
		PrimaryHost:             cluster.Spec.PrimaryHost,
		Database:                cluster.Spec.Database,
		DataPathOverride:        replica.Spec.Name,
		ArchiveMode:             archiveMode,
		ArchivePVCName:          util.CreateBackupPVCSnippet(archivePVCName),
		XLOGDir:                 xlogdir,
		ArchiveTimeout:          archiveTimeout,
		Replicas:                "1",
		ConfVolume:              operator.GetConfVolume(clientset, cluster, namespace),
		DeploymentLabels:        operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		PodLabels:               operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		SecurityContext:         util.CreateSecContext(replica.Spec.ReplicaStorage.Fsgroup, replica.Spec.ReplicaStorage.SupplementalGroups),
		RootSecretName:          cluster.Spec.RootSecretName,
		PrimarySecretName:       cluster.Spec.PrimarySecretName,
		UserSecretName:          cluster.Spec.UserSecretName,
		ContainerResources:      operator.GetContainerResourcesJSON(&cs),
		NodeSelector:            operator.GetReplicaAffinity(cluster.Spec.UserLabels, replica.Spec.UserLabels),
		CollectAddon:            operator.GetCollectAddon(clientset, namespace, &cluster.Spec),
		BadgerAddon:             operator.GetBadgerAddon(clientset, namespace, cluster, replica.Spec.Name),
		PgmonitorEnvVars:        operator.GetPgmonitorEnvVars(cluster.Spec.UserLabels[config.LABEL_COLLECT]),
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cluster.Labels[config.LABEL_BACKREST], replica.Spec.ClusterName, replica.Spec.Name,
			cluster.Spec.Port, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars: operator.GetPgbackrestS3EnvVars(cluster.Labels[config.LABEL_BACKREST],
			cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], clientset, namespace),
	}

	switch replica.Spec.ReplicaStorage.StorageType {
	case "", "emptydir":
		log.Debug("PrimaryStorage.StorageType is emptydir")
		err = config.DeploymentTemplate.Execute(&replicaDoc, replicaDeploymentFields)
	case "existing", "create", "dynamic":
		log.Debug("using the shared replica template ")
		err = config.DeploymentTemplate.Execute(&replicaDoc, replicaDeploymentFields)
	}

	if err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.DeploymentTemplate.Execute(os.Stdout, replicaDeploymentFields)
	}

	replicaDeployment := v1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling replica json into Deployment " + err.Error())
		return err
	}

	err = kubeapi.CreateDeployment(clientset, &replicaDeployment, namespace)

	return err
}

// DeleteReplica ...
func DeleteReplica(clientset *kubernetes.Clientset, cl *crv1.Pgreplica, namespace string) error {

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

// deletePgtasks
func deletePgtasks(restclient *rest.RESTClient, clusterName, namespace string) {

	taskList := crv1.PgtaskList{}

	//get a list of pgtasks for this cluster
	err := kubeapi.GetpgtasksBySelector(restclient,
		&taskList, config.LABEL_PG_CLUSTER+"="+clusterName,
		namespace)
	if err != nil {
		return
	}

	log.Debugf("pgtasks to remove is %d\n", len(taskList.Items))

	for _, r := range taskList.Items {
		err = kubeapi.Deletepgtask(restclient, r.Spec.Name, namespace)
	}

}
