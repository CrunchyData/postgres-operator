// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	ServiceType  string
}

// ReplicaSuffix ...
const ReplicaSuffix = "-replica"

func AddClusterBase(clientset kubeapi.Interface, cl *crv1.Pgcluster, namespace string) {
	var err error

	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, cl, namespace, cl.Annotations[config.ANNOTATION_CURRENT_PRIMARY], cl.Spec.PrimaryStorage)
	if err != nil {
		log.Error(err)
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
	if err = operator.CreatePGHAConfigMap(clientset, cl, namespace); err != nil &&
		!kerrors.IsAlreadyExists(err) {
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

	err = util.Patch(clientset.CrunchydataV1().RESTClient(), "/spec/status", crv1.CompletedStatus, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
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

	err = util.Patch(clientset.CrunchydataV1().RESTClient(), "/spec/PrimaryStorage/name", dataVolume.PersistentVolumeClaimName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)

	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	//publish create cluster event
	//capture the cluster creation event
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
		//create a CRD for each replica
		for i := 0; i < replicaCount; i++ {
			spec := crv1.PgreplicaSpec{}
			//get the storage config
			spec.ReplicaStorage = cl.Spec.ReplicaStorage

			spec.UserLabels = cl.Spec.UserLabels

			//the replica should not use the same node labels as the primary
			spec.UserLabels[config.LABEL_NODE_LABEL_KEY] = ""
			spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] = ""

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

			_, err = clientset.CrunchydataV1().Pgreplicas(namespace).Create(newInstance)
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

	namespace := cluster.GetNamespace()

	if err := operator.CreatePGHAConfigMap(clientset, cluster, namespace); err != nil &&
		!kerrors.IsAlreadyExists(err) {
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
		_, err = clientset.CrunchydataV1().Pgclusters(namespace).Patch(cluster.Name, types.MergePatchType, patch)
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

	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom
	repoName := fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName)

	found := true
	repoDeployment, err := clientset.AppsV1().Deployments(cluster.GetNamespace()).Get(
		repoName, metav1.GetOptions{})
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

// DeleteClusterBase ...
func DeleteClusterBase(clientset kubernetes.Interface, cl *crv1.Pgcluster, namespace string) {

	DeleteCluster(clientset, cl, namespace)

	//delete any existing configmaps
	if err := deleteConfigMaps(clientset, cl.Spec.Name, namespace); err != nil {
		log.Error(err)
	}

	//delete any existing pgtasks ???

	//publish delete cluster event
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventDeleteClusterFormat{
		EventHeader: events.EventHeader{
			Namespace: namespace,
			Username:  cl.ObjectMeta.Labels[config.LABEL_PGOUSER],
			Topic:     topics,
			Timestamp: time.Now(),
			EventType: events.EventDeleteCluster,
		},
		Clustername: cl.Spec.Name,
	}

	if err := events.Publish(f); err != nil {
		log.Error(err)
	}
}

// ScaleBase ...
func ScaleBase(clientset kubeapi.Interface, replica *crv1.Pgreplica, namespace string) {

	if replica.Spec.Status == crv1.CompletedStatus {
		log.Warn("crv1 pgreplica " + replica.Spec.Name + " is already marked complete, will not recreate")
		return
	}

	//get the pgcluster CRD to base the replica off of
	cluster, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(replica.Spec.ClusterName, metav1.GetOptions{})
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

	//update the replica CRD pvcname
	err = util.Patch(clientset.CrunchydataV1().RESTClient(), "/spec/replicastorage/name", dataVolume.PersistentVolumeClaimName, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	//create the replica service if it doesnt exist
	if err = scaleReplicaCreateMissingService(clientset, replica, cluster, namespace); err != nil {
		log.Error(err)
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster)
		return
	}

	//instantiate the replica
	if err = scaleReplicaCreateDeployment(clientset, replica, cluster, namespace, dataVolume, walVolume, tablespaceVolumes); err != nil {
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster)
		return
	}

	//update the replica CRD status
	err = util.Patch(clientset.CrunchydataV1().RESTClient(), "/spec/status", crv1.CompletedStatus, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}

	//publish event for replica creation
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

	//get the pgcluster CRD for this replica
	_, err := clientset.CrunchydataV1().Pgclusters(namespace).Get(replica.Spec.ClusterName, metav1.GetOptions{})
	if err != nil {
		return
	}

	DeleteReplica(clientset, replica, namespace)

	//publish event for scale down
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
func UpdateAnnotations(clientset kubernetes.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, annotations map[string]string) error {
	var updateError error

	// first, get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(clientset, cluster)

	if err != nil {
		return err
	}

	// now update each deployment with the new annotations
	for _, deployment := range deployments.Items {
		log.Debugf("update annotations on [%s]", deployment.Name)
		log.Debugf("new annotations: %v", annotations)

		deployment.Spec.Template.ObjectMeta.SetAnnotations(annotations)

		// Before applying the update, we want to explicitly stop PostgreSQL on each
		// instance. This prevents PostgreSQL from having to boot up in crash
		// recovery mode.
		//
		// If an error is returned, we only issue a warning
		if err := stopPostgreSQLInstance(clientset, restConfig, deployment); err != nil {
			log.Warn(err)
		}

		// finally, update the Deployment. If something errors, we'll log that there
		// was an error, but continue with processing the other deployments
		if _, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(&deployment); err != nil {
			log.Error(err)
			updateError = err
		}
	}

	return updateError
}

// UpdateResources updates the PostgreSQL instance Deployments to reflect the
// update resources (i.e. CPU, memory)
func UpdateResources(clientset kubernetes.Interface, restConfig *rest.Config, cluster *crv1.Pgcluster) error {
	// get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(clientset, cluster)

	if err != nil {
		return err
	}

	// iterate through each PostgreSQL instance deployment and update the
	// resource values for the database or exporter containers
	//
	// NOTE: a future version (near future) will first try to detect the primary
	// so that all the replicas are updated first, and then the primary gets the
	// update
	for _, deployment := range deployments.Items {
		// now, iterate through each container within that deployment
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
		// Before applying the update, we want to explicitly stop PostgreSQL on each
		// instance. This prevents PostgreSQL from having to boot up in crash
		// recovery mode.
		//
		// If an error is returned, we only issue a warning
		if err := stopPostgreSQLInstance(clientset, restConfig, deployment); err != nil {
			log.Warn(err)
		}
		// update the deployment with the new values
		if _, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(&deployment); err != nil {
			return err
		}
	}

	return nil
}

// UpdateTablespaces updates the PostgreSQL instance Deployments to update
// what tablespaces are mounted.
// Though any new tablespaces are present in the CRD, to attempt to do less work
// this function takes a map of the new tablespaces that are being added, so we
// only have to check and create the PVCs that are being mounted at this time
//
// To do this, iterate through the tablespace mount map that is present in the
// new cluster.
func UpdateTablespaces(clientset kubernetes.Interface, restConfig *rest.Config,
	cluster *crv1.Pgcluster, newTablespaces map[string]crv1.PgStorageSpec) error {
	// first, get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(clientset, cluster)

	if err != nil {
		return err
	}

	tablespaceVolumes := make([]map[string]operator.StorageResult, len(deployments.Items))

	// now we can start creating the new tablespaces! First, create the new
	// PVCs. The PVCs are created for each **instance** in the cluster, as every
	// instance needs to have a distinct PVC for each tablespace
	for i, deployment := range deployments.Items {
		tablespaceVolumes[i] = make(map[string]operator.StorageResult)

		for tablespaceName, storageSpec := range newTablespaces {
			// get the name of the tablespace PVC for that instance
			tablespacePVCName := operator.GetTablespacePVCName(deployment.Name, tablespaceName)

			log.Debugf("creating tablespace PVC [%s] for [%s]", tablespacePVCName, deployment.Name)

			// and now create it! If it errors, we just need to return, which
			// potentially leaves things in an inconsistent state, but at this point
			// only PVC objects have been created
			tablespaceVolumes[i][tablespaceName], err = pvc.CreateIfNotExists(clientset,
				storageSpec, tablespacePVCName, cluster.Name, cluster.Namespace)
			if err != nil {
				return err
			}
		}
	}

	// now the fun step: update each deployment with the new volumes
	for i, deployment := range deployments.Items {
		log.Debugf("attach tablespace volumes to [%s]", deployment.Name)

		// iterate through each table space and prepare the Volume and
		// VolumeMount clause for each instance
		for tablespaceName := range newTablespaces {
			// this is the volume to be added for the tablespace
			volume := v1.Volume{
				Name:         operator.GetTablespaceVolumeName(tablespaceName),
				VolumeSource: tablespaceVolumes[i][tablespaceName].VolumeSource(),
			}

			// add the volume to the list of volumes
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

			// now add the volume mount point to that of the database container
			volumeMount := v1.VolumeMount{
				MountPath: fmt.Sprintf("%s%s", config.VOLUME_TABLESPACE_PATH_PREFIX, tablespaceName),
				Name:      operator.GetTablespaceVolumeName(tablespaceName),
			}

			// we can do this as we always know that the "database" container is the
			// first container in the list
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)

			// add any supplemental groups specified in storage configuration.
			// SecurityContext is always initialized because we use fsGroup.
			deployment.Spec.Template.Spec.SecurityContext.SupplementalGroups = append(
				deployment.Spec.Template.Spec.SecurityContext.SupplementalGroups,
				tablespaceVolumes[i][tablespaceName].SupplementalGroups...)
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

		// Before applying the update, we want to explicitly stop PostgreSQL on each
		// instance. This prevents PostgreSQL from having to boot up in crash
		// recovery mode.
		//
		// If an error is returned, we only issue a warning
		if err := stopPostgreSQLInstance(clientset, restConfig, deployment); err != nil {
			log.Warn(err)
		}

		// finally, update the Deployment. Potential to put things into an
		// inconsistent state if any of these updates fail
		if _, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(&deployment); err != nil {
			return err
		}
	}

	return nil
}

// annotateBackrestSecret annotates the pgBackRest repository secret with relevant cluster
// configuration as needed to support bootstrapping from the repository after the cluster
// has been deleted
func annotateBackrestSecret(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {

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
	values := map[string]string{
		config.ANNOTATION_PG_PORT:             cluster.Spec.Port,
		config.ANNOTATION_REPO_PATH:           util.GetPGBackRestRepoPath(*cluster),
		config.ANNOTATION_S3_BUCKET:           cfg(cl.BackrestS3Bucket, op.BackrestS3Bucket),
		config.ANNOTATION_S3_ENDPOINT:         cfg(cl.BackrestS3Endpoint, op.BackrestS3Endpoint),
		config.ANNOTATION_S3_REGION:           cfg(cl.BackrestS3Region, op.BackrestS3Region),
		config.ANNOTATION_SSHD_PORT:           strconv.Itoa(operator.Pgo.Cluster.BackrestPort),
		config.ANNOTATION_SUPPLEMENTAL_GROUPS: cluster.Spec.BackrestStorage.SupplementalGroups,
		config.ANNOTATION_S3_URI_STYLE:        cfg(cl.BackrestS3URIStyle, op.BackrestS3URIStyle),
		config.ANNOTATION_S3_VERIFY_TLS:       cfg(cl.BackrestS3VerifyTLS, op.BackrestS3VerifyTLS),
	}
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return err
	}

	secretName := fmt.Sprintf(util.BackrestRepoSecretName, clusterName)
	patchString := fmt.Sprintf(`{"metadata":{"annotations":%s}}`, string(valuesJSON))

	log.Debugf("About to patch secret %s (namespace %s) using:\n%s", secretName, namespace,
		patchString)
	if _, err := clientset.CoreV1().Secrets(namespace).Patch(secretName, types.MergePatchType,
		[]byte(patchString)); err != nil {
		return err
	}

	return nil
}

func deleteConfigMaps(clientset kubernetes.Interface, clusterName, ns string) error {
	label := fmt.Sprintf("pg-cluster=%s", clusterName)
	list, err := clientset.CoreV1().ConfigMaps(ns).List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return fmt.Errorf("No configMaps found for selector: %s", label)
	}

	for _, configmap := range list.Items {
		err := clientset.CoreV1().ConfigMaps(ns).Delete(configmap.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
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

	//capture the cluster creation event
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
	// First, attempt to get the PostgreSQL instance Pod attachd to this
	// particular deployment
	selector := fmt.Sprintf("%s=%s", config.LABEL_DEPLOYMENT_NAME, deployment.Name)
	pods, err := clientset.CoreV1().Pods(deployment.Namespace).List(metav1.ListOptions{LabelSelector: selector})

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
