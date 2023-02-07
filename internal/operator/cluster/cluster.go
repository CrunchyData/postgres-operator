// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017 - 2023 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"

	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ServiceTemplateFields ...
type ServiceTemplateFields struct {
	Name         string
	ServiceName  string
	ClusterName  string
	Port         string
	PGBadgerPort string
	ExporterPort string
	ServiceType  v1.ServiceType
}

// ReplicaSuffix is the suffix of the replica Service name
const ReplicaSuffix = "-replica"

const (
	// exporterContainerName is the name of the exporter container
	exporterContainerName = "exporter"

	// pgBadgerContainerName is the name of the pgBadger container
	pgBadgerContainerName = "pgbadger"
)

func AddClusterBase(clientset kubeapi.Interface, cl *crv1.Pgcluster, namespace string) {
	ctx := context.TODO()
	var err error

	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, cl, namespace, cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY], cl.Spec.PrimaryStorage)
	if err != nil {
		log.Error(err)
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	// create any missing user secrets that are required to be part of the
	// bootstrap
	if err := createMissingUserSecrets(clientset, cl); err != nil {
		log.Errorf("error creating missing user secrets: %q", err.Error())
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	if err = addClusterCreateMissingService(clientset, cl, namespace); err != nil {
		log.Error("error in creating primary service " + err.Error())
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	// Create a configMap for the cluster that will be utilized to configure whether or not
	// initialization logic should be executed when the postgres-ha container is run.  This
	// ensures that the original primary in a PG cluster does not attempt to run any initialization
	// logic following a restart of the container.
	// If the configmap already exists, the cluster creation will continue as this is required
	// for certain pgcluster upgrades.
	if err := operator.CreatePGHAConfigMap(clientset, cl,
		namespace); kerrors.IsAlreadyExists(err) {
		if !pghaConigMapHasInitFlag(clientset, cl) {
			log.Infof("found existing pgha ConfigMap for cluster %s without init flag set. "+
				"setting init flag to 'true'", cl.GetName())

			// if the value is not present, update the config map
			if err := operator.UpdatePGHAConfigInitFlag(clientset, true, cl.Name, cl.Namespace); err != nil {
				log.Error(err)
				publishClusterCreateFailure(cl, err.Error())
				return
			}
		}
	} else if err != nil {
		log.Error(err)
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	// ensure the the pgBackRest Secret is created. If this fails, we have to
	// abort
	if err := backrest.CreateRepoSecret(clientset, cl); err != nil {
		log.Error(err)
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	if err := annotateBackrestSecret(clientset, cl); err != nil {
		log.Error(err)
		publishClusterCreateFailure(cl, err.Error())
	}

	if err := addClusterDeployments(clientset, cl, namespace,
		dataVolume, walVolume, tablespaceVolumes); err != nil {
		log.Error(err)
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	// Now scale the repo deployment only to ensure it is initialized prior to the primary DB.
	// Once the repo is ready, the primary database deployment will then also be scaled to 1.
	clusterInfo, err := ScaleClusterDeployments(clientset, *cl, 1, false, false, true, false)
	if err != nil {
		log.Error(err)
		publishClusterCreateFailure(cl, err.Error())
	}
	log.Debugf("Scaled pgBackRest repo deployment %s to 1 to proceed with initializing "+
		"cluster %s", clusterInfo.PrimaryDeployment, cl.GetName())

	patch, err := kubeapi.NewJSONPatch().Add("spec", "status")(crv1.CompletedStatus).Bytes()
	if err == nil {
		log.Debugf("patching cluster %s: %s", cl.Spec.Name, patch)
		_, err = clientset.CrunchydataV1().Pgclusters(namespace).
			Patch(ctx, cl.Spec.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}

	// patch in the correct current primary value to the CRD spec, as well as
	// any updated user labels. This will handle both new and updated clusters.
	// Note: in previous operator versions, this was stored in a user label
	if err := util.PatchClusterCRD(clientset, cl.Spec.UserLabels, cl, cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY], namespace); err != nil {
		log.Error("could not patch primary crv1 with labels")
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	patch, err = kubeapi.NewJSONPatch().Add("spec", "PrimaryStorage", "name")(dataVolume.PersistentVolumeClaimName).Bytes()
	if err == nil {
		log.Debugf("patching cluster %s: %s", cl.Spec.Name, patch)
		_, err = clientset.CrunchydataV1().Pgclusters(namespace).
			Patch(ctx, cl.Spec.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	// publish create cluster event
	// capture the cluster creation event
	pgouser := cl.ObjectMeta.Labels[config.LABEL_PGOUSER]
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventCreateClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: cl.ObjectMeta.Namespace,
			Username:  pgouser,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreateCluster,
		},
		Clustername: cl.ObjectMeta.Name,
		WorkflowID:  cl.ObjectMeta.Labels[config.LABEL_WORKFLOW_ID],
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}

	// determine if a restore
	_, restore := cl.GetAnnotations()[config.ANNOTATION_BACKREST_RESTORE]

	// add replicas if requested, and if not a restore
	if cl.Spec.Replicas != "" && !restore {
		replicaCount, err := strconv.Atoi(cl.Spec.Replicas)
		if err != nil {
			log.Error("error in replicas value " + err.Error())
			publishClusterCreateFailure(cl, err.Error())
			return
		}
		// create a CRD for each replica
		for i := 0; i < replicaCount; i++ {
			spec := crv1.PgreplicaSpec{}
			// get the storage config
			spec.ReplicaStorage = cl.Spec.ReplicaStorage

			spec.UserLabels = cl.Spec.UserLabels

			// if the primary cluster has default node affinity rules set, we need
			// to honor them in the spec. if a different affinity is desired, the
			// replica needs to set its own rules
			if cl.Spec.NodeAffinity.Default != nil {
				spec.NodeAffinity = cl.Spec.NodeAffinity.Default
			}

			labels := make(map[string]string)
			labels[config.LABEL_PG_CLUSTER] = cl.Spec.Name

			spec.ClusterName = cl.Spec.Name
			uniqueName := util.RandStringBytesRmndr(4)
			labels[config.LABEL_NAME] = cl.Spec.Name + "-" + uniqueName
			spec.Name = labels[config.LABEL_NAME]
			newInstance := &crv1.Pgreplica{
				ObjectMeta: metav1.ObjectMeta{
					Name:   labels[config.LABEL_NAME],
					Labels: labels,
				},
				Spec: spec,
				Status: crv1.PgreplicaStatus{
					State:   crv1.PgreplicaStateCreated,
					Message: "Created, not processed yet",
				},
			}

			_, err = clientset.CrunchydataV1().Pgreplicas(namespace).
				Create(ctx, newInstance, metav1.CreateOptions{})
			if err != nil {
				log.Error(" in creating Pgreplica instance" + err.Error())
				publishClusterCreateFailure(cl, err.Error())
			}

		}
	}
}

// AddClusterBootstrap creates the resources needed to bootstrap a new cluster from an existing
// data source.  Specifically, this function creates the bootstrap job that will be run to
// bootstrap the cluster, along with supporting resources (e.g. ConfigMaps and volumes).
func AddClusterBootstrap(clientset kubeapi.Interface, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()
	namespace := cluster.GetNamespace()

	var err error

	if err = operator.CreatePGHAConfigMap(clientset, cluster,
		namespace); kerrors.IsAlreadyExists(err) {
		log.Infof("found existing pgha ConfigMap for cluster %s, setting init flag to 'true'",
			cluster.GetName())
		err = operator.UpdatePGHAConfigInitFlag(clientset, true, cluster.Name, cluster.Namespace)
	}
	if err != nil {
		log.Error(err)
		publishClusterCreateFailure(cluster, err.Error())
		return err
	}

	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, cluster, namespace,
		cluster.Annotations[config.ANNOTATION_CURRENT_PRIMARY], cluster.Spec.PrimaryStorage)
	if err != nil {
		publishClusterCreateFailure(cluster, err.Error())
		return err
	}

	if err := addClusterBootstrapJob(clientset, cluster, namespace, dataVolume,
		walVolume, tablespaceVolumes); err != nil && !kerrors.IsAlreadyExists(err) {
		publishClusterCreateFailure(cluster, err.Error())
		return err
	}

	patch, err := json.Marshal(map[string]interface{}{
		"status": crv1.PgclusterStatus{
			State:   crv1.PgclusterStateBootstrapping,
			Message: "Bootstapping cluster from an existing data source",
		},
	})
	if err == nil {
		log.Debugf("patching cluster %s: %s", cluster.Name, patch)
		_, err = clientset.CrunchydataV1().Pgclusters(namespace).
			Patch(ctx, cluster.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		return err
	}

	return nil
}

// AddBootstrapRepo creates a pgBackRest repository and associated service to use when
// bootstrapping a cluster from an existing data source.  If an existing repo is detected
// and is being used to bootstrap another cluster, then an error is returned.  If an existing
// repo is detected and is not associated with a bootstrap job (but rather an active cluster),
// then no action is taken and the function resturns.  Also, in addition to returning an error
// in the event an error is encountered, the function also returns a 'repoCreated' bool that
// specifies whether or not a repo was actually created.
func AddBootstrapRepo(clientset kubernetes.Interface, cluster *crv1.Pgcluster) (repoCreated bool, err error) {
	ctx := context.TODO()
	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom
	repoName := fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName)

	found := true
	repoDeployment, err := clientset.AppsV1().Deployments(cluster.GetNamespace()).
		Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return
		}
		found = false
	}

	if !found {
		if err = backrest.CreateRepoDeployment(clientset, cluster, false, true, 1); err != nil {
			return
		}
		repoCreated = true
	} else if _, ok := repoDeployment.GetLabels()[config.LABEL_PGHA_BOOTSTRAP]; ok {
		err = fmt.Errorf("Unable to create bootstrap repo %s to bootstrap cluster %s "+
			"(namespace %s) because it is already running to bootstrap another cluster",
			repoName, cluster.GetName(), cluster.GetNamespace())
		return
	}

	return
}

// ScaleBase ...
func ScaleBase(clientset kubeapi.Interface, replica *crv1.Pgreplica, namespace string) {
	ctx := context.TODO()

	if replica.Spec.Status == crv1.CompletedStatus {
		log.Warn("crv1 pgreplica " + replica.Spec.Name + " is already marked complete, will not recreate")
		return
	}

	// get the pgcluster CRD to base the replica off of
	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).
		Get(ctx, replica.Spec.ClusterName, metav1.GetOptions{})
	if err != nil {
		return
	}

	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, cluster, namespace, replica.Spec.Name, replica.Spec.ReplicaStorage)
	if err != nil {
		log.Error(err)
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster)
		return
	}

	// update the replica CRD pvcname
	patch, err := kubeapi.NewJSONPatch().Add("spec", "replicastorage", "name")(dataVolume.PersistentVolumeClaimName).Bytes()
	if err == nil {
		log.Debugf("patching replica %s: %s", replica.Spec.Name, patch)
		_, err = clientset.CrunchydataV1().Pgreplicas(namespace).
			Patch(ctx, replica.Spec.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	// create the replica service if it doesnt exist
	if err = scaleReplicaCreateMissingService(clientset, replica, cluster, namespace); err != nil {
		log.Error(err)
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster)
		return
	}

	// instantiate the replica
	if err = scaleReplicaCreateDeployment(clientset, replica, cluster, namespace, dataVolume, walVolume, tablespaceVolumes); err != nil {
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster)
		return
	}

	// update the replica CRD status
	patch, err = kubeapi.NewJSONPatch().Add("spec", "status")(crv1.CompletedStatus).Bytes()
	if err == nil {
		log.Debugf("patching replica %s: %s", replica.Spec.Name, patch)
		_, err = clientset.CrunchydataV1().Pgreplicas(namespace).
			Patch(ctx, replica.Spec.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	}
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}

	// publish event for replica creation
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventScaleClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  replica.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventScaleCluster,
		},
		Clustername: cluster.Spec.UserLabels[config.LABEL_REPLICA_NAME],
		Replicaname: cluster.Spec.UserLabels[config.LABEL_PG_CLUSTER],
	}

	if err = events.Publish(f); err != nil {
		log.Error(err.Error())
	}
}

// ScaleDownBase ...
func ScaleDownBase(clientset kubeapi.Interface, replica *crv1.Pgreplica, namespace string) {
	ctx := context.TODO()

	// get the pgcluster CRD for this replica
	_, err := clientset.CrunchydataV1().Pgclusters(namespace).
		Get(ctx, replica.Spec.ClusterName, metav1.GetOptions{})
	if err != nil {
		return
	}

	_ = DeleteReplica(clientset, replica, namespace)

	// publish event for scale down
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventScaleDownClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  replica.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventScaleDownCluster,
		},
		Clustername: replica.Spec.ClusterName,
	}

	err = events.Publish(f)
	if err != nil {
		log.Error(err.Error())
		return
	}
}

// UpdateAnnotations updates the annotations in the "template" portion of a
// PostgreSQL deployment
func UpdateAnnotations(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	log.Debugf("update annotations on [%s]", deployment.Name)
	annotations := map[string]string{}

	// store the global annotations first
	for k, v := range cluster.Spec.Annotations.Global {
		annotations[k] = v
	}

	// then store the postgres specific annotations
	for k, v := range cluster.Spec.Annotations.Postgres {
		annotations[k] = v
	}

	log.Debugf("new annotations: %v", annotations)

	// set the annotations on the deployment object
	deployment.Spec.Template.ObjectMeta.SetAnnotations(annotations)

	return nil
}

// UpdateResources updates the PostgreSQL instance Deployments to reflect the
// update resources (i.e. CPU, memory)
func UpdateResources(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	// iterate through each PostgreSQL instance deployment and update the
	// resource values for the database or exporter containers
	for index, container := range deployment.Spec.Template.Spec.Containers {
		// first check for the database container
		if container.Name == "database" {
			// first, initialize the requests/limits resource to empty Resource Lists
			deployment.Spec.Template.Spec.Containers[index].Resources.Requests = v1.ResourceList{}
			deployment.Spec.Template.Spec.Containers[index].Resources.Limits = v1.ResourceList{}

			// now, simply deep copy the values from the CRD
			if cluster.Spec.Resources != nil {
				deployment.Spec.Template.Spec.Containers[index].Resources.Requests = cluster.Spec.Resources.DeepCopy()
			}

			if cluster.Spec.Limits != nil {
				deployment.Spec.Template.Spec.Containers[index].Resources.Limits = cluster.Spec.Limits.DeepCopy()
			}
			// next, check for the exporter container
		} else if container.Name == "exporter" {
			// first, initialize the requests/limits resource to empty Resource Lists
			deployment.Spec.Template.Spec.Containers[index].Resources.Requests = v1.ResourceList{}
			deployment.Spec.Template.Spec.Containers[index].Resources.Limits = v1.ResourceList{}

			// now, simply deep copy the values from the CRD
			if cluster.Spec.ExporterResources != nil {
				deployment.Spec.Template.Spec.Containers[index].Resources.Requests = cluster.Spec.ExporterResources.DeepCopy()
			}

			if cluster.Spec.ExporterLimits != nil {
				deployment.Spec.Template.Spec.Containers[index].Resources.Limits = cluster.Spec.ExporterLimits.DeepCopy()
			}
		}
	}

	return nil
}

// UpdateTablespaces updates the PostgreSQL instance Deployments to update
// what tablespaces are mounted.
func UpdateTablespaces(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	// update the volume portion of the Deployment spec to reflect all of the
	// available tablespaces
	for tablespaceName, storageSpec := range cluster.Spec.TablespaceMounts {
		// go through the volume list and see if there is already a volume for this
		// if there is, skip
		found := false
		volumeName := operator.GetTablespaceVolumeName(tablespaceName)

		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Name == volumeName {
				found = true
				break
			}
		}

		if found {
			continue
		}

		// create the volume definition for the tablespace
		storageResult := operator.StorageResult{
			PersistentVolumeClaimName: operator.GetTablespacePVCName(deployment.Name, tablespaceName),
			SupplementalGroups:        storageSpec.GetSupplementalGroups(),
		}

		volume := v1.Volume{
			Name:         volumeName,
			VolumeSource: storageResult.VolumeSource(),
		}

		// add the volume to the list of volumes
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

		// now add the volume mount point to that of the database container
		volumeMount := v1.VolumeMount{
			MountPath: fmt.Sprintf("%s%s", config.VOLUME_TABLESPACE_PATH_PREFIX, tablespaceName),
			Name:      volumeName,
		}

		// we can do this as we always know that the "database" container is the
		// first container in the list
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)

		// add any supplemental groups specified in storage configuration.
		// SecurityContext is always initialized because we use fsGroup.
		deployment.Spec.Template.Spec.SecurityContext.SupplementalGroups = append(
			deployment.Spec.Template.Spec.SecurityContext.SupplementalGroups,
			storageResult.SupplementalGroups...)
	}

	// find the "PGHA_TABLESPACES" value and update it with the new tablespace
	// name list
	ok := false
	for i, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
		// yup, it's an old fashioned linear time lookup
		if envVar.Name == "PGHA_TABLESPACES" {
			deployment.Spec.Template.Spec.Containers[0].Env[i].Value = operator.GetTablespaceNames(
				cluster.Spec.TablespaceMounts)
			ok = true
		}
	}

	// if its not found, we need to add it to the env
	if !ok {
		envVar := v1.EnvVar{
			Name:  "PGHA_TABLESPACES",
			Value: operator.GetTablespaceNames(cluster.Spec.TablespaceMounts),
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, envVar)
	}

	return nil
}

// UpdateTolerations updates the Toleration definition for a Deployment.
// However, we have to check if the Deployment is based on a pgreplica Spec --
// if it is, we need to determine if there are any instance specific tolerations
// defined on that
func UpdateTolerations(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	ctx := context.TODO()

	// determine if this instance is based on the pgcluster or a pgreplica. if
	// it is based on the pgcluster, we can apply the tolerations and exit early
	if deployment.Name == cluster.Name {
		deployment.Spec.Template.Spec.Tolerations = cluster.Spec.Tolerations
		return nil
	}

	// ok, so this is based on a pgreplica. Let's try to find it.
	instance, err := clientset.CrunchydataV1().Pgreplicas(cluster.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})

	// if we error, log it and return, as this error will interrupt a rolling update
	if err != nil {
		log.Error(err)
		return err
	}

	// "replica" instances can have toleration overrides. these get managed as
	// part of the pgreplicas controller, not here. as such, if this "replica"
	// instance has specific toleration overrides, we will exit here so we do not
	// apply the cluster-wide tolerations
	if len(instance.Spec.Tolerations) != 0 {
		return nil
	}

	// otherwise, the tolerations set on the cluster instance are available to
	// all instances, so set the value and return
	deployment.Spec.Template.Spec.Tolerations = cluster.Spec.Tolerations

	return nil
}

// annotateBackrestSecret annotates the pgBackRest repository secret with relevant cluster
// configuration as needed to support bootstrapping from the repository after the cluster
// has been deleted
func annotateBackrestSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()
	clusterName := cluster.GetName()
	namespace := cluster.GetNamespace()

	// simple helper that takes two config options, returning the first if populated, and
	// if not the returning the second (which also might now be populated)
	cfg := func(cl, op string) string {
		if cl != "" {
			return cl
		}
		return op
	}
	cl := cluster.Spec
	op := operator.Pgo.Cluster
	secretName := fmt.Sprintf(util.BackrestRepoSecretName, clusterName)
	patch, err := kubeapi.NewMergePatch().Add("metadata", "annotations")(map[string]string{
		config.ANNOTATION_PG_PORT:             cluster.Spec.Port,
		config.ANNOTATION_REPO_PATH:           operator.GetPGBackRestRepoPath(cluster),
		config.ANNOTATION_S3_BUCKET:           cfg(cl.BackrestS3Bucket, op.BackrestS3Bucket),
		config.ANNOTATION_S3_ENDPOINT:         cfg(cl.BackrestS3Endpoint, op.BackrestS3Endpoint),
		config.ANNOTATION_S3_REGION:           cfg(cl.BackrestS3Region, op.BackrestS3Region),
		config.ANNOTATION_SSHD_PORT:           strconv.Itoa(operator.Pgo.Cluster.BackrestPort),
		config.ANNOTATION_SUPPLEMENTAL_GROUPS: cluster.Spec.BackrestStorage.SupplementalGroups,
		config.ANNOTATION_S3_URI_STYLE:        cfg(cl.BackrestS3URIStyle, op.BackrestS3URIStyle),
		config.ANNOTATION_S3_VERIFY_TLS:       cfg(cl.BackrestS3VerifyTLS, op.BackrestS3VerifyTLS),
	}).Bytes()

	if err == nil {
		log.Debugf("patching secret %s: %s", secretName, patch)
		_, err = clientset.CoreV1().Secrets(namespace).
			Patch(ctx, secretName, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	return err
}

// createMissingUserSecret is the heart of trying to determine if a user secret
// is missing, and if it is, creating it. Requires the appropriate secretName
// suffix for a given secret, as well as the user name
// createUserSecret(request, newInstance, crv1.RootSecretSuffix,
// 	crv1.PGUserSuperuser, request.PasswordSuperuser)
func createMissingUserSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster, username string) error {
	ctx := context.TODO()

	// derive the secret name
	secretName := crv1.UserSecretName(cluster, username)

	// if the secret already exists, skip it
	// if it returns an error other than "not found" return an error
	if _, err := clientset.CoreV1().Secrets(cluster.Spec.Namespace).Get(
		ctx, secretName, metav1.GetOptions{}); err == nil {
		log.Infof("user secret %q exists for user %q for cluster %q",
			secretName, username, cluster.Spec.Name)
		return nil
	} else if !kerrors.IsNotFound(err) {
		return err
	}

	// alright, so we have to create the secret
	// if the password fails to generate, return an error
	passwordLength := util.GeneratedPasswordLength(operator.Pgo.Cluster.PasswordLength)
	password, err := util.GeneratePassword(passwordLength)
	if err != nil {
		return err
	}

	// great, now we can create the secret! if we can't, return an error
	return util.CreateSecret(clientset, cluster.Spec.Name, secretName,
		username, password, cluster.Spec.Namespace)
}

// createMissingUserSecrets checks to see if there are secrets for the
// superuser (postgres), replication user (primaryuser), and a standard postgres
// user for the given cluster. Each of these are created if they do not
// currently exist
func createMissingUserSecrets(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {
	// first, determine if we need to create a user secret for the postgres
	// superuser
	if err := createMissingUserSecret(clientset, cluster, crv1.PGUserSuperuser); err != nil {
		return err
	}

	// next, determine if we need to create a user secret for the replication user
	if err := createMissingUserSecret(clientset, cluster, crv1.PGUserReplication); err != nil {
		return err
	}

	// finally, determine if we need to create a user secret for the regular user
	return createMissingUserSecret(clientset, cluster, cluster.Spec.User)
}

// pghaConigMapHasInitFlag checks to see if the PostgreSQL ConfigMap has the
// PGHA init flag. Returns true if it does have it set, false otherwise.
// If any function calls have an error, we will log that error and return false
func pghaConigMapHasInitFlag(clientset kubernetes.Interface, cluster *crv1.Pgcluster) bool {
	ctx := context.TODO()

	// load the PGHA config map for this cluster. This more or less assumes that
	// it exists
	configMapName := fmt.Sprintf("%s-%s", cluster.Name, operator.PGHAConfigMapSuffix)
	configMap, err := clientset.CoreV1().ConfigMaps(cluster.Namespace).Get(ctx, configMapName, metav1.GetOptions{})

	// if there is an error getting the ConfigMap, log the error and return
	if err != nil {
		log.Error(err)
		return false
	}

	// determine if the init flag is set, regardless of it's true or false
	_, ok := configMap.Data[operator.PGHAConfigInitSetting]

	return ok
}

func publishClusterCreateFailure(cl *crv1.Pgcluster, errorMsg string) {
	pgouser := cl.ObjectMeta.Labels[config.LABEL_PGOUSER]
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventCreateClusterFailureFormat{
		EventHeader: events.EventHeader{
			Namespace: cl.ObjectMeta.Namespace,
			Username:  pgouser,
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventCreateClusterFailure,
		},
		Clustername:  cl.ObjectMeta.Name,
		ErrorMessage: errorMsg,
		WorkflowID:   cl.ObjectMeta.Labels[config.LABEL_WORKFLOW_ID],
	}

	err := events.Publish(f)
	if err != nil {
		log.Error(err.Error())
	}
}

func publishClusterShutdown(cluster crv1.Pgcluster) error {
	clusterName := cluster.Name

	// capture the cluster creation event
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventShutdownClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: cluster.Namespace,
			Username:  cluster.Spec.UserLabels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventShutdownCluster,
		},
		Clustername: clusterName,
	}

	if err := events.Publish(f); err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}

// stopPostgreSQLInstance is a proxy function for the main
// StopPostgreSQLInstance function, as it preps a Deployment to have its
// PostgreSQL instance shut down. This helps to ensure that a PostgreSQL
// instance will launch and not be in crash recovery mode
func stopPostgreSQLInstance(clientset kubernetes.Interface, restConfig *rest.Config, deployment apps_v1.Deployment) error {
	ctx := context.TODO()

	// First, attempt to get the PostgreSQL instance Pod attachd to this
	// particular deployment
	selector := fmt.Sprintf("%s=%s", config.LABEL_DEPLOYMENT_NAME, deployment.Name)
	pods, err := clientset.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
		LabelSelector: selector,
	})

	// if there is a bona fide error, return.
	// However, if no Pods are found, issue a warning, but do not return an error
	// This likely means that PostgreSQL is already shutdown, but hey, it's the
	// cloud
	if err != nil {
		return err
	} else if len(pods.Items) == 0 {
		log.Infof("not shutting down PostgreSQL instance [%s] as the Pod cannot be found", deployment.Name)
		return nil
	}

	// get the first pod off the items list
	pod := pods.Items[0]

	// now we can shut down the cluster
	if err := util.StopPostgreSQLInstance(clientset, restConfig, &pod, deployment.Name); err != nil {
		return err
	}

	return nil
}
