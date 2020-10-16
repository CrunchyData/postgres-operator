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
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// addClusterCreateMissingService creates a service for the cluster primary if
// it does not yet exist.
func addClusterCreateMissingService(clientset kubernetes.Interface, cl *crv1.Pgcluster, namespace string) error {
	st := operator.Pgo.Cluster.ServiceType
	if cl.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = cl.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	}

	// create the primary service
	serviceFields := ServiceTemplateFields{
		Name:        cl.Spec.Name,
		ServiceName: cl.Spec.Name,
		ClusterName: cl.Spec.Name,
		Port:        cl.Spec.Port,
		ServiceType: st,
	}

	// only add references to the exporter / pgBadger ports
	clusterLabels := cl.ObjectMeta.GetLabels()

	if val, ok := clusterLabels[config.LABEL_BADGER]; ok && val == config.LABEL_TRUE {
		serviceFields.PGBadgerPort = cl.Spec.PGBadgerPort
	}

	// ...due to legacy reasons, the exporter label may not be available yet in the
	// main labels. so we will check here first, and then check the user labels
	if val, ok := clusterLabels[config.LABEL_EXPORTER]; ok && val == config.LABEL_TRUE {
		serviceFields.ExporterPort = cl.Spec.ExporterPort
	}

	// ...this condition should be targeted for removal in the future
	if cl.Spec.UserLabels != nil {
		if val, ok := cl.Spec.UserLabels[config.LABEL_EXPORTER]; ok && val == config.LABEL_TRUE {
			serviceFields.ExporterPort = cl.Spec.ExporterPort
		}
	}

	return CreateService(clientset, &serviceFields, namespace)
}

// addClusterBootstrapJob creates a job that will be used to bootstrap a PostgreSQL cluster from an
// existing data source
func addClusterBootstrapJob(clientset kubeapi.Interface,
	cl *crv1.Pgcluster, namespace string, dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult) error {

	bootstrapFields, err := getBootstrapJobFields(clientset, cl, dataVolume, walVolume,
		tablespaceVolumes)
	if err != nil {
		return err
	}

	var bootstrapSpec bytes.Buffer
	if err := config.BootstrapTemplate.Execute(&bootstrapSpec, bootstrapFields); err != nil {
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.DeploymentTemplate.Execute(os.Stdout, bootstrapFields)
	}

	job := &batchv1.Job{}
	if err := json.Unmarshal(bootstrapSpec.Bytes(), job); err != nil {
		return err
	}

	if cl.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToPostgreSQL(&job.Spec.Template.Spec, walVolume,
			cl.Spec.Name)
	}

	operator.AddBackRestConfigVolumeAndMounts(&job.Spec.Template.Spec, cl.Name, cl.Spec.BackrestConfig)

	// determine if any of the container images need to be overridden
	operator.OverrideClusterContainerImages(job.Spec.Template.Spec.Containers)

	if _, err := clientset.BatchV1().Jobs(namespace).Create(job); err != nil {
		return err
	}

	return nil
}

// addClusterDeployments creates deployments for pgBackRest and PostgreSQL.
func addClusterDeployments(clientset kubeapi.Interface,
	cl *crv1.Pgcluster, namespace string, dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult) error {

	if err := backrest.CreateRepoDeployment(clientset, cl, true, false, 0); err != nil {
		return err
	}

	deploymentFields := getClusterDeploymentFields(clientset, cl,
		dataVolume, walVolume, tablespaceVolumes)

	var primaryDoc bytes.Buffer
	if err := config.DeploymentTemplate.Execute(&primaryDoc, deploymentFields); err != nil {
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.DeploymentTemplate.Execute(os.Stdout, deploymentFields)
	}

	deployment := &appsv1.Deployment{}
	if err := json.Unmarshal(primaryDoc.Bytes(), deployment); err != nil {
		return err
	}

	if cl.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToPostgreSQL(&deployment.Spec.Template.Spec, walVolume,
			cl.Spec.Name)
	}

	operator.AddBackRestConfigVolumeAndMounts(&deployment.Spec.Template.Spec, cl.Name, cl.Spec.BackrestConfig)

	// determine if any of the container images need to be overridden
	operator.OverrideClusterContainerImages(deployment.Spec.Template.Spec.Containers)

	if _, err := clientset.AppsV1().Deployments(namespace).Create(deployment); err != nil &&
		!kerrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// getBootstrapJobFields obtains the fields needed to populate the cluster bootstrap job template
func getBootstrapJobFields(clientset kubeapi.Interface,
	cluster *crv1.Pgcluster, dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult) (operator.BootstrapJobTemplateFields, error) {

	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom
	restoreOpts := strconv.Quote(cluster.Spec.PGDataSource.RestoreOpts)

	bootstrapFields := operator.BootstrapJobTemplateFields{
		DeploymentTemplateFields: getClusterDeploymentFields(clientset, cluster, dataVolume,
			walVolume, tablespaceVolumes),
		RestoreFrom: cluster.Spec.PGDataSource.RestoreFrom,
		RestoreOpts: restoreOpts[1 : len(restoreOpts)-1],
	}

	// A recovery target should also have a recovery target action. The PostgreSQL
	// and pgBackRest defaults are `pause` which requires the user to execute SQL
	// before the cluster will accept any writes. If no action has been specified,
	// use `promote` which accepts writes as soon as recovery succeeds.
	//
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html#RUNTIME-CONFIG-WAL-RECOVERY-TARGET
	// - https://pgbackrest.org/command.html#command-restore/category-command/option-target-action
	//
	if strings.Contains(restoreOpts, "--target") &&
		!strings.Contains(restoreOpts, "--target-action") {
		bootstrapFields.RestoreOpts =
			strings.TrimSpace(bootstrapFields.RestoreOpts + " --target-action=promote")
	}

	// Grab the pgBackRest secret from the "restore from" cluster to obtain the annotations
	// containing the additional configuration details needed to bootstrap from the clusters
	// pgBackRest repository
	restoreFromSecret, err := clientset.CoreV1().Secrets(cluster.GetNamespace()).Get(
		fmt.Sprintf(util.BackrestRepoSecretName, restoreClusterName), metav1.GetOptions{})
	if err != nil {
		return bootstrapFields, err
	}

	// Grab the cluster to restore from to see if it still exists
	restoreCluster, err := clientset.CrunchydataV1().Pgclusters(cluster.GetNamespace()).Get(restoreClusterName, metav1.GetOptions{})
	found := true
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return bootstrapFields, err
		}
		found = false
	}

	// If the cluster exists, only proceed if it isnt shutdown
	if found && (restoreCluster.Status.State == crv1.PgclusterStateShutdown) {
		return bootstrapFields, fmt.Errorf("Unable to bootstrap cluster %s from cluster %s "+
			"(namespace %s) because it has a %s status", cluster.GetName(),
			restoreClusterName, cluster.GetNamespace(),
			string(restoreCluster.Status.State))
	}

	// Now override any backrest env vars for the bootstrap job
	bootstrapBackrestVars, err := operator.GetPgbackrestBootstrapEnvVars(restoreClusterName,
		cluster.GetName(), restoreFromSecret)
	if err != nil {
		return bootstrapFields, err
	}
	bootstrapFields.PgbackrestEnvVars = bootstrapBackrestVars

	// if an s3 restore is detected, override or set the pgbackrest S3 env vars, otherwise do
	// not set the s3 env vars at all
	s3Restore := backrest.S3RepoTypeCLIOptionExists(cluster.Spec.PGDataSource.RestoreOpts)
	if s3Restore {
		// Now override any backrest S3 env vars for the bootstrap job
		bootstrapFields.PgbackrestS3EnvVars = operator.GetPgbackrestBootstrapS3EnvVars(
			cluster.Spec.PGDataSource.RestoreFrom, restoreFromSecret)
	} else {
		bootstrapFields.PgbackrestS3EnvVars = ""
	}

	return bootstrapFields, nil
}

// getClusterDeploymentFields obtains the fields needed to populate the cluster deployment template
func getClusterDeploymentFields(clientset kubernetes.Interface,
	cl *crv1.Pgcluster, dataVolume, walVolume operator.StorageResult,
	tablespaceVolumes map[string]operator.StorageResult) operator.DeploymentTemplateFields {

	namespace := cl.GetNamespace()

	log.Infof("creating Pgcluster %s in namespace %s", cl.Name, namespace)

	cl.Spec.UserLabels["name"] = cl.Spec.Name
	cl.Spec.UserLabels[config.LABEL_PG_CLUSTER] = cl.Spec.ClusterName

	// if the current deployment label value does not match current primary name
	// update the label so that the new deployment will match the existing PVC
	// as determined previously
	// Note that the use of this value brings the initial deployment creation in line with
	// the paradigm used during cluster restoration, as in operator/backrest/restore.go
	if cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY] != cl.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] {
		cl.Spec.UserLabels[config.LABEL_DEPLOYMENT_NAME] = cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY]
	}

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
		Name:               cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY],
		IsInit:             true,
		Replicas:           "0",
		ClusterName:        cl.Spec.Name,
		Port:               cl.Spec.Port,
		CCPImagePrefix:     util.GetValueOrDefault(cl.Spec.CCPImagePrefix, operator.Pgo.Cluster.CCPImagePrefix),
		CCPImage:           cl.Spec.CCPImage,
		CCPImageTag:        cl.Spec.CCPImageTag,
		PVCName:            dataVolume.InlineVolumeSource(),
		DeploymentLabels:   operator.GetLabelsFromMap(cl.Spec.UserLabels),
		PodAnnotations:     operator.GetAnnotations(cl, crv1.ClusterAnnotationPostgres),
		PodLabels:          operator.GetLabelsFromMap(cl.Spec.UserLabels),
		DataPathOverride:   cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY],
		Database:           cl.Spec.Database,
		SecurityContext:    operator.GetPodSecurityContext(supplementalGroups),
		RootSecretName:     cl.Spec.RootSecretName,
		PrimarySecretName:  cl.Spec.PrimarySecretName,
		UserSecretName:     cl.Spec.UserSecretName,
		NodeSelector:       operator.GetAffinity(cl.Spec.UserLabels["NodeLabelKey"], cl.Spec.UserLabels["NodeLabelValue"], "In"),
		PodAntiAffinity:    operator.GetPodAntiAffinity(cl, crv1.PodAntiAffinityDeploymentDefault, cl.Spec.PodAntiAffinity.Default),
		ContainerResources: operator.GetResourcesJSON(cl.Spec.Resources, cl.Spec.Limits),
		ConfVolume:         operator.GetConfVolume(clientset, cl, namespace),
		ExporterAddon:      operator.GetExporterAddon(clientset, namespace, &cl.Spec),
		BadgerAddon:        operator.GetBadgerAddon(clientset, namespace, cl, cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY]),
		PgmonitorEnvVars:   operator.GetPgmonitorEnvVars(cl.Spec.UserLabels[config.LABEL_EXPORTER], cl.Spec.CollectSecretName),
		ScopeLabel:         config.LABEL_PGHA_SCOPE,
		PgbackrestEnvVars: operator.GetPgbackrestEnvVars(cl, cl.Labels[config.LABEL_BACKREST], cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY],
			cl.Spec.Port, cl.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:      operator.GetPgbackrestS3EnvVars(*cl, clientset, namespace),
		EnableCrunchyadm:         operator.Pgo.Cluster.EnableCrunchyadm,
		ReplicaReinitOnStartFail: !operator.Pgo.Cluster.DisableReplicaStartFailReinit,
		SyncReplication:          operator.GetSyncReplication(cl.Spec.SyncReplication),
		Tablespaces:              operator.GetTablespaceNames(cl.Spec.TablespaceMounts),
		TablespaceVolumes:        operator.GetTablespaceVolumesJSON(cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY], tablespaceStorageTypeMap),
		TablespaceVolumeMounts:   operator.GetTablespaceVolumeMountsJSON(tablespaceStorageTypeMap),
		TLSEnabled:               cl.Spec.TLS.IsTLSEnabled(),
		TLSOnly:                  cl.Spec.TLSOnly,
		TLSSecret:                cl.Spec.TLS.TLSSecret,
		ReplicationTLSSecret:     cl.Spec.TLS.ReplicationTLSSecret,
		CASecret:                 cl.Spec.TLS.CASecret,
		Standby:                  cl.Spec.Standby,
	}

	return deploymentFields
}

// DeleteCluster ...
func DeleteCluster(clientset kubernetes.Interface, cl *crv1.Pgcluster, namespace string) error {

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
func scaleReplicaCreateMissingService(clientset kubernetes.Interface, replica *crv1.Pgreplica, cluster *crv1.Pgcluster, namespace string) error {
	st := operator.Pgo.Cluster.ServiceType
	if replica.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = replica.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	} else if cluster.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = cluster.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	}

	serviceName := fmt.Sprintf("%s-replica", replica.Spec.ClusterName)
	serviceFields := ServiceTemplateFields{
		Name:        serviceName,
		ServiceName: serviceName,
		ClusterName: replica.Spec.ClusterName,
		Port:        cluster.Spec.Port,
		ServiceType: st,
	}

	// only add references to the exporter / pgBadger ports
	clusterLabels := cluster.ObjectMeta.GetLabels()

	if val, ok := clusterLabels[config.LABEL_EXPORTER]; ok && val == config.LABEL_TRUE {
		serviceFields.ExporterPort = cluster.Spec.ExporterPort
	}

	if val, ok := clusterLabels[config.LABEL_BADGER]; ok && val == config.LABEL_TRUE {
		serviceFields.PGBadgerPort = cluster.Spec.PGBadgerPort
	}

	return CreateService(clientset, &serviceFields, namespace)
}

// scaleReplicaCreateDeployment creates a deployment for the cluster replica.
func scaleReplicaCreateDeployment(clientset kubernetes.Interface,
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
		Database:           cluster.Spec.Database,
		DataPathOverride:   replica.Spec.Name,
		ArchiveMode:        archiveMode,
		Replicas:           "1",
		ConfVolume:         operator.GetConfVolume(clientset, cluster, namespace),
		DeploymentLabels:   operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		PodAnnotations:     operator.GetAnnotations(cluster, crv1.ClusterAnnotationPostgres),
		PodLabels:          operator.GetLabelsFromMap(cluster.Spec.UserLabels),
		SecurityContext:    operator.GetPodSecurityContext(supplementalGroups),
		RootSecretName:     cluster.Spec.RootSecretName,
		PrimarySecretName:  cluster.Spec.PrimarySecretName,
		UserSecretName:     cluster.Spec.UserSecretName,
		ContainerResources: operator.GetResourcesJSON(cluster.Spec.Resources, cluster.Spec.Limits),
		NodeSelector:       operator.GetAffinity(replica.Spec.UserLabels["NodeLabelKey"], replica.Spec.UserLabels["NodeLabelValue"], "In"),
		PodAntiAffinity:    operator.GetPodAntiAffinity(cluster, crv1.PodAntiAffinityDeploymentDefault, cluster.Spec.PodAntiAffinity.Default),
		ExporterAddon:      operator.GetExporterAddon(clientset, namespace, &cluster.Spec),
		BadgerAddon:        operator.GetBadgerAddon(clientset, namespace, cluster, replica.Spec.Name),
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
		ReplicationTLSSecret:     cluster.Spec.TLS.ReplicationTLSSecret,
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

	replicaDeployment := appsv1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling replica json into Deployment " + err.Error())
		return err
	}

	if cluster.Spec.WALStorage.StorageType != "" {
		operator.AddWALVolumeAndMountsToPostgreSQL(&replicaDeployment.Spec.Template.Spec, walVolume, replica.Spec.Name)
	}

	operator.AddBackRestConfigVolumeAndMounts(&replicaDeployment.Spec.Template.Spec, cluster.Name, cluster.Spec.BackrestConfig)

	// determine if any of the container images need to be overridden
	operator.OverrideClusterContainerImages(replicaDeployment.Spec.Template.Spec.Containers)

	// set the replica scope to the same scope as the primary, i.e. the scope defined using label
	// 'crunchy-pgha-scope'
	replicaDeployment.Labels[config.LABEL_PGHA_SCOPE] = cluster.Labels[config.LABEL_PGHA_SCOPE]
	replicaDeployment.Spec.Template.Labels[config.LABEL_PGHA_SCOPE] = cluster.Labels[config.LABEL_PGHA_SCOPE]

	_, err = clientset.AppsV1().Deployments(namespace).Create(&replicaDeployment)
	return err
}

// DeleteReplica ...
func DeleteReplica(clientset kubernetes.Interface, cl *crv1.Pgreplica, namespace string) error {

	var err error
	log.Info("deleting Pgreplica object" + " in namespace " + namespace)
	log.Info("deleting with Name=" + cl.Spec.Name + " in namespace " + namespace)
	deletePropagation := metav1.DeletePropagationForeground
	err = clientset.
		AppsV1().Deployments(namespace).
		Delete(cl.Spec.Name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		})

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
func ShutdownCluster(clientset kubeapi.Interface, cluster crv1.Pgcluster) error {

	// first ensure the current primary deployment is properly recorded in the pg
	// cluster. Only consider primaries that are running, as there could be
	// evicted, etc. pods hanging around
	selector := fmt.Sprintf("%s=%s,%s=%s", config.LABEL_PG_CLUSTER, cluster.Name,
		config.LABEL_PGHA_ROLE, config.LABEL_PGHA_ROLE_PRIMARY)

	options := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	}

	// only consider pods that are running
	pods, err := clientset.CoreV1().Pods(cluster.Namespace).List(options)

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

	if _, err := clientset.CrunchydataV1().Pgclusters(cluster.Namespace).Update(&cluster); err != nil {
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
	patch, err := json.Marshal(map[string]interface{}{
		"status": crv1.PgclusterStatus{
			State: crv1.PgclusterStateShutdown,
			Message: fmt.Sprintf("Database shutdown along with the following services: %v", []string{
				clusterInfo.PGBackRestRepoDeployment,
				clusterInfo.PGBouncerDeployment,
			}),
		},
	})
	if err == nil {
		log.Debugf("patching cluster %s: %s", cluster.Name, patch)
		_, err = clientset.CrunchydataV1().Pgclusters(cluster.Namespace).Patch(cluster.Name, types.MergePatchType, patch)
	}
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().ConfigMaps(cluster.Namespace).Delete(fmt.Sprintf("%s-leader",
		cluster.Labels[config.LABEL_PGHA_SCOPE]), &metav1.DeleteOptions{}); err != nil {
		return err
	}

	publishClusterShutdown(cluster)

	return nil
}

// StartupCluster is responsible for starting a cluster that was previsouly shutdown.  This
// includes changing the replica count for all clusters to 1, and then updating the pgcluster
// with a shutdown status.
func StartupCluster(clientset kubernetes.Interface, cluster crv1.Pgcluster) error {

	log.Debugf("Cluster Operator: starting cluster %s", cluster.Name)

	// ensure autofailover is enabled to ensure proper startup of the cluster
	if err := util.ToggleAutoFailover(clientset, true, cluster.Labels[config.LABEL_PGHA_SCOPE],
		cluster.Namespace); err != nil {
		return fmt.Errorf("Cluster Operator: Unable to toggle autofailover when starting "+
			"cluster %s", cluster.Name)
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
func ScaleClusterDeployments(clientset kubernetes.Interface, cluster crv1.Pgcluster, replicas int,
	scalePrimary, scaleReplicas, scaleBackRestRepo,
	scalePGBouncer bool) (clusterInfo ScaleClusterInfo, err error) {

	clusterName := cluster.Name
	namespace := cluster.Namespace
	// Get *all* remaining deployments for the cluster.  This includes the deployment for the
	// primary, any replicas, the pgBackRest repo and any optional services (e.g. pgBouncer)
	var deploymentList *appsv1.DeploymentList
	selector := fmt.Sprintf("%s=%s", config.LABEL_PG_CLUSTER, clusterName)
	deploymentList, err = clientset.
		AppsV1().Deployments(namespace).
		List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return
	}

	for _, deployment := range deploymentList.Items {

		// determine if the deployment is a primary, replica, or supporting service (pgBackRest,
		// pgBouncer, etc.)
		switch {
		case deployment.Name == cluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY]:
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
			// if the replica total is greater than 0, set number of pgBouncer
			// replicas to the number that is specified in the cluster entry
			if replicas > 0 {
				replicas = int(cluster.Spec.PgBouncer.Replicas)
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

		// Scale the deployment according to the number of replicas specified.  If an error is
		// encountered, log it and move on to scaling the next deployment
		patch, err := kubeapi.NewMergePatch().Add("spec", "replicas")(replicas).Bytes()
		if err == nil {
			log.Debugf("patching deployment %s: %s", deployment.GetName(), patch)
			_, err = clientset.AppsV1().Deployments(namespace).Patch(deployment.GetName(), types.MergePatchType, patch)
		}
		if err != nil {
			log.Errorf("Error scaling deployment %s to %d: %v", deployment.Name, replicas, err)
		}
	}
	return
}
