// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"os"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/backrest"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// addClusterCreateMissingService creates a service for the cluster primary if
// it does not yet exist.
func addClusterCreateMissingService(clientset *kubernetes.Clientset, cl *crv1.Pgcluster, namespace string) error {
	st := operator.Pgo.Cluster.ServiceType
	if cl.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = cl.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	}

	//create the primary service
	serviceFields := ServiceTemplateFields{
		Name:         cl.Spec.Name,
		ServiceName:  cl.Spec.Name,
		ClusterName:  cl.Spec.Name,
		Port:         cl.Spec.Port,
		PGBadgerPort: cl.Spec.PGBadgerPort,
		ExporterPort: cl.Spec.ExporterPort,
		ServiceType:  st,
	}

	return CreateService(clientset, &serviceFields, namespace)
}

// addClusterCreateDeployments creates deployments for pgBackRest and PostgreSQL.
func addClusterCreateDeployments(clientset *kubernetes.Clientset, client *rest.RESTClient,
	cl *crv1.Pgcluster, namespace string,
	dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult,
) error {
	var primaryDoc bytes.Buffer
	var err error

	log.Info("creating Pgcluster object  in namespace " + namespace)
	log.Info("created with Name=" + cl.Spec.Name + " in namespace " + namespace)

	cl.Spec.UserLabels["name"] = cl.Spec.Name
	cl.Spec.UserLabels[config.LABEL_PG_CLUSTER] = cl.Spec.ClusterName

	archiveMode := "off"
	if cl.Spec.UserLabels[config.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
	}
	if cl.Labels[config.LABEL_BACKREST] == "true" {
		//backrest requires us to turn on archive mode
		archiveMode = "on"
		if err := backrest.CreateRepoDeployment(clientset, namespace, cl, true,
			0); err != nil {
			log.Error("could not create backrest repo deployment")
			return err
		}
	}

	cl.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = cl.Spec.Name
	cl.Spec.UserLabels[config.LABEL_PGOUSER] = cl.ObjectMeta.Labels[config.LABEL_PGOUSER]
	cl.Spec.UserLabels[config.LABEL_PG_CLUSTER_IDENTIFIER] = cl.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER]

	// Set the Patroni scope to the name of the primary deployment.  Replicas will get scope using the
	// 'crunchy-pgha-scope' label on the pgcluster
	cl.Spec.UserLabels[config.LABEL_PGHA_SCOPE] = cl.Spec.Name

	// set up a map of the names of the tablespaces as well as the storage classes
	tablespaceStorageTypeMap := operator.GetTablespaceStorageTypeMap(cl.Spec.TablespaceMounts)

	// combine supplemental groups from all volumes
	var supplementalGroups []int64
	supplementalGroups = append(supplementalGroups, dataVolume.SupplementalGroups...)
	for _, v := range tablespaceVolumes {
		supplementalGroups = append(supplementalGroups, v.SupplementalGroups...)
	}

	//create the primary deployment
	deploymentFields := operator.DeploymentTemplateFields{
		Name:               cl.Spec.Name,
		IsInit:             true,
		Replicas:           "0",
		ClusterName:        cl.Spec.Name,
		PrimaryHost:        cl.Spec.Name,
		Port:               cl.Spec.Port,
		CCPImagePrefix:     util.GetValueOrDefault(cl.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImage:           cl.Spec.CCPImage,
		CCPImageTag:        cl.Spec.CCPImageTag,
		PVCName:            dataVolume.InlineVolumeSource(),
		DeploymentLabels:   operator.GetLabelsFromMap(cl.Spec.UserLabels),
		PodLabels:          operator.GetLabelsFromMap(cl.Spec.UserLabels),
		DataPathOverride:   cl.Spec.Name,
		Database:           cl.Spec.Database,
		ArchiveMode:        archiveMode,
		SecurityContext:    util.GetPodSecurityContext(supplementalGroups),
		RootSecretName:     cl.Spec.RootSecretName,
		PrimarySecretName:  cl.Spec.PrimarySecretName,
		UserSecretName:     cl.Spec.UserSecretName,
		NodeSelector:       operator.GetAffinity(cl.Spec.UserLabels["NodeLabelKey"], cl.Spec.UserLabels["NodeLabelValue"], "In"),
		PodAntiAffinity:    operator.GetPodAntiAffinity(cl, crv1.PodAntiAffinityDeploymentDefault, cl.Spec.PodAntiAffinity.Default),
		ContainerResources: operator.GetResourcesJSON(cl.Spec.Resources),
		ConfVolume:         operator.GetConfVolume(clientset, cl, namespace),
		CollectAddon:       operator.GetCollectAddon(clientset, namespace, &cl.Spec),
		CollectVolume:      operator.GetCollectVolume(clientset, cl, namespace),
		BadgerAddon:        operator.GetBadgerAddon(clientset, namespace, cl, cl.Spec.Name),
		PgmonitorEnvVars:   operator.GetPgmonitorEnvVars(cl.Spec.UserLabels[config.LABEL_COLLECT]),
		ScopeLabel:         config.LABEL_PGHA_SCOPE,
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cl, cl.Labels[config.LABEL_BACKREST], cl.Spec.Name,
			cl.Spec.Port, cl.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:      operator.GetPgbackrestS3EnvVars(*cl, clientset, namespace),
		EnableCrunchyadm:         operator.Pgo.Cluster.EnableCrunchyadm,
		ReplicaReinitOnStartFail: !operator.Pgo.Cluster.DisableReplicaStartFailReinit,
		SyncReplication:          operator.GetSyncReplication(cl.Spec.SyncReplication),
		Tablespaces:              operator.GetTablespaceNames(cl.Spec.TablespaceMounts),
		TablespaceVolumes:        operator.GetTablespaceVolumesJSON(cl.Spec.Name, tablespaceStorageTypeMap),
		TablespaceVolumeMounts:   operator.GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap),
		TLSEnabled:               cl.Spec.TLS.IsTLSEnabled(),
		TLSOnly:                  cl.Spec.TLSOnly,
		TLSSecret:                cl.Spec.TLS.TLSSecret,
		CASecret:                 cl.Spec.TLS.CASecret,
		Standby:                  cl.Spec.Standby,
	}

	// Create a configMap for the cluster that will be utilized to configure whether or not
	// initialization logic should be executed when the postgres-ha container is run.  This
	// ensures that the original primary in a PG cluster does not attempt to run any initialization
	// logic following a restart of the container.
	if err = operator.CreatePGHAConfigMap(clientset, cl, namespace); err != nil {
		log.Error(err.Error())
		return err
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

	if cl.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToPostgreSQL(&deployment.Spec.Template.Spec, walVolume, cl.Spec.Name)
	}

	// determine if any of the container images need to be overridden
	operator.OverrideClusterContainerImages(deployment.Spec.Template.Spec.Containers)

	if _, found, _ := kubeapi.GetDeployment(clientset, cl.Spec.Name, namespace); !found {
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

	//create rmdata job
	isReplica := false
	isBackup := false
	removeData := true
	removeBackup := false
	err = CreateRmdataJob(clientset, cl, namespace, removeData, removeBackup, isReplica, isBackup)
	if err != nil {
		log.Error(err)
		return err
	} else {
		publishDeleteCluster(namespace, cl.ObjectMeta.Labels[config.LABEL_PGOUSER], cl.Spec.Name, cl.ObjectMeta.Labels[config.LABEL_PG_CLUSTER_IDENTIFIER])
	}

	return err

}

// scaleReplicaCreateMissingService creates a service for cluster replicas if
// it does not yet exist.
func scaleReplicaCreateMissingService(clientset *kubernetes.Clientset, replica *crv1.Pgreplica, cluster *crv1.Pgcluster, namespace string) error {
	st := operator.Pgo.Cluster.ServiceType
	if replica.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = replica.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	} else if cluster.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = cluster.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	}

	serviceName := fmt.Sprintf("%s-replica", replica.Spec.ClusterName)
	serviceFields := ServiceTemplateFields{
		Name:         serviceName,
		ServiceName:  serviceName,
		ClusterName:  replica.Spec.ClusterName,
		Port:         cluster.Spec.Port,
		PGBadgerPort: cluster.Spec.PGBadgerPort,
		ExporterPort: cluster.Spec.ExporterPort,
		ServiceType:  st,
	}

	return CreateService(clientset, &serviceFields, namespace)
}

// scaleReplicaCreateDeployment creates a deployment for the cluster replica.
func scaleReplicaCreateDeployment(clientset *kubernetes.Clientset, client *rest.RESTClient,
	replica *crv1.Pgreplica, cluster *crv1.Pgcluster, namespace string,
	dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult,
) error {
	var err error
	log.Debugf("Scale called for %s in %s", replica.Name, namespace)

	var replicaDoc bytes.Buffer

	serviceName := replica.Spec.ClusterName + "-replica"
	//replicaFlag := true

	//	replicaLabels := operator.GetPrimaryLabels(serviceName, replica.Spec.ClusterName, replicaFlag, cluster.Spec.UserLabels)
	cluster.Spec.UserLabels[config.LABEL_REPLICA_NAME] = replica.Spec.Name
	cluster.Spec.UserLabels["name"] = serviceName
	cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER] = replica.Spec.ClusterName

	archiveMode := "off"
	if cluster.Spec.UserLabels[config.LABEL_ARCHIVE] == "true" {
		archiveMode = "on"
	}
	if cluster.Labels[config.LABEL_BACKREST] == "true" {
		//backrest requires archive mode be set to on
		archiveMode = "on"
	}

	image := cluster.Spec.CCPImage

	//check for --ccp-image-tag at the command line
	imageTag := cluster.Spec.CCPImageTag
	if replica.Spec.UserLabels[config.LABEL_CCP_IMAGE_TAG_KEY] != "" {
		imageTag = replica.Spec.UserLabels[config.LABEL_CCP_IMAGE_TAG_KEY]
	}

	cluster.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = replica.Spec.Name

	// set up a map of the names of the tablespaces as well as the storage classes
	tablespaceStorageTypeMap := operator.GetTablespaceStorageTypeMap(cluster.Spec.TablespaceMounts)

	// combine supplemental groups from all volumes
	var supplementalGroups []int64
	supplementalGroups = append(supplementalGroups, dataVolume.SupplementalGroups...)
	for _, v := range tablespaceVolumes {
		supplementalGroups = append(supplementalGroups, v.SupplementalGroups...)
	}

	//create the replica deployment
	replicaDeploymentFields := operator.DeploymentTemplateFields{
		Name:               replica.Spec.Name,
		ClusterName:        replica.Spec.ClusterName,
		Port:               cluster.Spec.Port,
		CCPImagePrefix:     util.GetValueOrDefault(cluster.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImageTag:        imageTag,
		CCPImage:           image,
		PVCName:            dataVolume.InlineVolumeSource(),
		PrimaryHost:        cluster.Spec.PrimaryHost,
		Database:           cluster.Spec.Database,
		DataPathOverride:   replica.Spec.Name,
		ArchiveMode:        archiveMode,
		Replicas:           "1",
		ConfVolume:         operator.GetConfVolume(clientset, cluster, namespace),
		DeploymentLabels:   operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		PodLabels:          operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		SecurityContext:    util.GetPodSecurityContext(supplementalGroups),
		RootSecretName:     cluster.Spec.RootSecretName,
		PrimarySecretName:  cluster.Spec.PrimarySecretName,
		UserSecretName:     cluster.Spec.UserSecretName,
		ContainerResources: operator.GetResourcesJSON(cluster.Spec.Resources),
		NodeSelector:       operator.GetAffinity(replica.Spec.UserLabels["NodeLabelKey"], replica.Spec.UserLabels["NodeLabelValue"], "In"),
		PodAntiAffinity:    operator.GetPodAntiAffinity(cluster, crv1.PodAntiAffinityDeploymentDefault, cluster.Spec.PodAntiAffinity.Default),
		CollectAddon:       operator.GetCollectAddon(clientset, namespace, &cluster.Spec),
		CollectVolume:      operator.GetCollectVolume(clientset, cluster, namespace),
		BadgerAddon:        operator.GetBadgerAddon(clientset, namespace, cluster, replica.Spec.Name),
		PgmonitorEnvVars:   operator.GetPgmonitorEnvVars(cluster.Spec.UserLabels[config.LABEL_COLLECT]),
		ScopeLabel:         config.LABEL_PGHA_SCOPE,
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cluster, cluster.Labels[config.LABEL_BACKREST], replica.Spec.Name,
			cluster.Spec.Port, cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:      operator.GetPgbackrestS3EnvVars(*cluster, clientset, namespace),
		EnableCrunchyadm:         operator.Pgo.Cluster.EnableCrunchyadm,
		ReplicaReinitOnStartFail: !operator.Pgo.Cluster.DisableReplicaStartFailReinit,
		SyncReplication:          operator.GetSyncReplication(cluster.Spec.SyncReplication),
		Tablespaces:              operator.GetTablespaceNames(cluster.Spec.TablespaceMounts),
		TablespaceVolumes:        operator.GetTablespaceVolumesJSON(replica.Spec.Name, tablespaceStorageTypeMap),
		TablespaceVolumeMounts:   operator.GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap),
		TLSEnabled:               cluster.Spec.TLS.IsTLSEnabled(),
		TLSOnly:                  cluster.Spec.TLSOnly,
		TLSSecret:                cluster.Spec.TLS.TLSSecret,
		CASecret:                 cluster.Spec.TLS.CASecret,
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

	if cluster.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToPostgreSQL(&replicaDeployment.Spec.Template.Spec, walVolume, replica.Spec.Name)
	}

	// determine if any of the container images need to be overridden
	operator.OverrideClusterContainerImages(replicaDeployment.Spec.Template.Spec.Containers)

	// set the replica scope to the same scope as the primary, i.e. the scope defined using label
	// 'crunchy-pgha-scope'
	replicaDeployment.Labels[config.LABEL_PGHA_SCOPE] = cluster.Labels[config.LABEL_PGHA_SCOPE]
	replicaDeployment.Spec.Template.Labels[config.LABEL_PGHA_SCOPE] = cluster.Labels[config.LABEL_PGHA_SCOPE]

	return kubeapi.CreateDeployment(clientset, &replicaDeployment, namespace)
}

// DeleteReplica ...
func DeleteReplica(clientset *kubernetes.Clientset, cl *crv1.Pgreplica, namespace string) error {

	var err error
	log.Info("deleting Pgreplica object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)
	err = kubeapi.DeleteDeployment(clientset, cl.Spec.Name, namespace)

	return err

}

func publishScaleError(namespace string, username string, cluster *crv1.Pgcluster) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventScaleClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventScaleCluster,
		},
		Clustername: cluster.Spec.UserLabels[config.LABEL_REPLICA_NAME],
		Replicaname: cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER],
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}

func publishDeleteCluster(namespace, username, clusterName, identifier string) {
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventDeleteClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  username,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeleteCluster,
		},
		Clustername: clusterName,
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}

// ScaleClusterInfo contains information about a cluster obtained when scaling the various
// deployments for a cluster.  This includes the name of the primary deployment, all replica
// deployments, along with the names of the services enabled for the cluster.
type ScaleClusterInfo struct {
	PrimaryDeployment        string
	ReplicaDeployments       []string
	PGBackRestRepoDeployment string
	PGBouncerDeployment      string
}

// ShutdownCluster is responsible for shutting down a cluster that is currently running.  This
// includes changing the replica count for all clusters to 0, and then updating the pgcluster
// with a shutdown status.
func ShutdownCluster(clientset *kubernetes.Clientset, restclient *rest.RESTClient,
	cluster crv1.Pgcluster) error {

	// first ensure the current primary deployment is properly recorded in the pg cluster.  This
	// is needed to ensure the cluster can be
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGHA_ROLE, "master")
	pods, err := kubeapi.GetPods(clientset, selector, cluster.Namespace)
	if err != nil {
		return err
	}
	if len(pods.Items) > 1 {
		return fmt.Errorf("Cluster Operator: Invalid number of primary pods (%d) found when "+
			"shutting down cluster %s", len(pods.Items), cluster.Name)
	}
	primaryPod := pods.Items[0]
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations[config.ANNOTATION_PRIMARY_DEPLOYMENT] =
		primaryPod.Labels[config.LABEL_DEPLOYMENT_NAME]

	if err := kubeapi.Updatepgcluster(restclient, &cluster, cluster.Name,
		cluster.Namespace); err != nil {
		return fmt.Errorf("Cluster Operator: Unable to update the current primary deployment "+
			"in the pgcluster when shutting down cluster %s", cluster.Name)
	}

	// disable autofailover to prevent failovers while shutting down deployments
	if err := util.ToggleAutoFailover(clientset, false, cluster.Labels[config.LABEL_PGHA_SCOPE],
		cluster.Namespace); err != nil {
		return fmt.Errorf("Cluster Operator: Unable to toggle autofailover when shutting "+
			"down cluster %s", cluster.Name)
	}

	clusterInfo, err := ScaleClusterDeployments(clientset, cluster, 0, true, true, true, true)
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Database shutdown along with the following services: %v",
		append(make([]string, 0), clusterInfo.PGBackRestRepoDeployment,
			clusterInfo.PGBouncerDeployment))
	if err := kubeapi.PatchpgclusterStatus(restclient, crv1.PgclusterStateShutdown,
		message, &cluster, cluster.Namespace); err != nil {
		return err
	}

	if err := kubeapi.DeleteConfigMap(clientset, fmt.Sprintf("%s-leader",
		cluster.Labels[config.LABEL_PGHA_SCOPE]), cluster.Namespace); err != nil {
		return err
	}

	publishClusterShutdown(cluster)

	return nil
}

// StartupCluster is responsible for starting a cluster that was previsouly shutdown.  This
// includes changing the replica count for all clusters to 1, and then updating the pgcluster
// with a shutdown status.
func StartupCluster(clientset *kubernetes.Clientset, cluster crv1.Pgcluster) error {

	log.Debugf("Cluster Operator: starting cluster %s", cluster.Name)

	// ensure autofailover is enabled to ensure proper startup of the cluster
	if err := util.ToggleAutoFailover(clientset, true, cluster.Labels[config.LABEL_PGHA_SCOPE],
		cluster.Namespace); err != nil {
		return fmt.Errorf("Cluster Operator: Unable to toggle autofailover when shutting "+
			"down cluster %s", cluster.Name)
	}

	// Scale up the primary and supporting services, but not the replicas.  Replicas will be
	// scaled up after the primary is ready.  This ensures the primary at the time of shutdown
	// is the primary when the cluster comes back online.
	clusterInfo, err := ScaleClusterDeployments(clientset, cluster, 1, true, false, true, true)
	if err != nil {
		return err
	}

	log.Debugf("Cluster Operator: primary deployment %s started for cluster %s along with "+
		"services %v.  The following replicas will be started once the primary has initialized: "+
		"%v", clusterInfo.PrimaryDeployment, cluster.Name, append(make([]string, 0),
		clusterInfo.PGBackRestRepoDeployment, clusterInfo.PGBouncerDeployment),
		clusterInfo.ReplicaDeployments)

	return nil
}

// ScaleClusterDeployments scales all deployments for a cluster to the number of replicas
// specified using the 'replicas' parameter.  This is typically used to scale-up or down the
// primary deployment and any supporting services (pgBackRest and pgBouncer) when shutting down
// or starting up the cluster due to a scale or scale-down request.
func ScaleClusterDeployments(clientset *kubernetes.Clientset, cluster crv1.Pgcluster, replicas int,
	scalePrimary, scaleReplicas, scaleBackRestRepo,
	scalePGBouncer bool) (clusterInfo ScaleClusterInfo, err error) {

	clusterName := cluster.Name
	namespace := cluster.Namespace
	// Get *all* remaining deployments for the cluster.  This includes the deployment for the
	// primary, any replicas, the pgBackRest repo and any optional services (e.g. pgBouncer)
	var deploymentList *v1.DeploymentList
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, clusterName)
	if deploymentList, err = kubeapi.GetDeployments(clientset, selector,
		namespace); err != nil {
		return
	}

	for _, deployment := range deploymentList.Items {

		// determine if the deployment is a primary, replica, or supporting service (pgBackRest,
		// pgBouncer, etc.)
		switch {
		case deployment.Name == cluster.Labels[config.LABEL_CURRENT_PRIMARY]:
			clusterInfo.PrimaryDeployment = deployment.Name
			// if not scaling the primary simply move on to the next deployment
			if !scalePrimary {
				continue
			}
		case deployment.Labels[config.LABEL_PGBOUNCER] == "true":
			clusterInfo.PGBouncerDeployment = deployment.Name
			// if not scaling services simply move on to the next deployment
			if !scalePGBouncer {
				continue
			}
		case deployment.Labels[config.LABEL_PGO_BACKREST_REPO] == "true":
			clusterInfo.PGBackRestRepoDeployment = deployment.Name
			// if not scaling services simply move on to the next deployment
			if !scaleBackRestRepo {
				continue
			}
		default:
			clusterInfo.ReplicaDeployments = append(clusterInfo.ReplicaDeployments,
				deployment.Name)
			// if not scaling replicas simply move on to the next deployment
			if !scaleReplicas {
				continue
			}
		}

		log.Debugf("scaling deployment %s to %d for cluster %s", deployment.Name, replicas,
			clusterName)

		// Scale the deployment accoriding to the number of replicas specified.  If an error is
		// encountered, log it and move on to scaling the next deployment.
		if err = kubeapi.ScaleDeployment(clientset, deployment, replicas); err != nil {
			log.Errorf("Error scaling deployment %s to %d: %w", deployment.Name, replicas, err)
		}
	}
	return
}
