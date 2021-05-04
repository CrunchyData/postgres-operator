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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/backrest"
	cfg "github.com/crunchydata/postgres-operator/internal/operator/config"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/pkg/events"

	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
	CustomLabels string
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

// systemLabels is a list of the system labels that need to be copied over when
// also applying the custom labels
var systemLabels = []string{
	config.LABEL_PGHA_SCOPE,
	config.LABEL_DEPLOYMENT_NAME,
	config.LABEL_NAME,
	config.LABEL_PG_CLUSTER,
	config.LABEL_POD_ANTI_AFFINITY,
	config.LABEL_PG_DATABASE,
	config.LABEL_PGO_VERSION,
	config.LABEL_PGOUSER,
	config.LABEL_VENDOR,
	config.LABEL_WORKFLOW_ID,
}

// a group of constants that are used as part of the TLS support
const (
	tlsEnvVarEnabled        = "PGHA_TLS_ENABLED"
	tlsEnvVarOnly           = "PGHA_TLS_ONLY"
	tlsMountPathReplication = "/pgconf/tls-replication"
	tlsMountPathServer      = "/pgconf/tls"
	tlsVolumeReplication    = "tls-replication"
	tlsVolumeServer         = "tls-server"
)

var (
	// tlsHBAPattern matches the pattern of what a TLS entry looks like in the
	// Postgres HBA file
	tlsHBAPattern = regexp.MustCompile(`^hostssl`)

	// notlsHBAPattern matches the pattern of what a regular host entry looks like
	// in the Postgres HBA file
	notlsHBAPattern = regexp.MustCompile(`^(host|hostnossl)\s+`)
)

// tlsHBARules is a collection of standard TLS rules that PGO adds to our HBA
// file
var tlsHBARules = []string{
	"hostssl replication " + crv1.PGUserReplication + " 0.0.0.0/0 md5",
	"hostssl all " + crv1.PGUserReplication + " 0.0.0.0/0 reject",
	"hostssl all all 0.0.0.0/0 md5",
}

// tlsVolumeReplicationSizeLimit is the size limit for the optional volume
// for the TLS Secret for replication
var tlsVolumeReplicationSizeLimit = resource.MustParse("2Mi")

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

	// create a copy of the pgBackRest secret for the cluster being restored from
	bootstrapSecret, err := createBootstrapBackRestSecret(clientset, cluster)
	if err != nil {
		publishClusterCreateFailure(cluster, err.Error())
		return err
	}

	if err := addClusterBootstrapJob(clientset, cluster, dataVolume,
		walVolume, tablespaceVolumes, bootstrapSecret); err != nil && !kerrors.IsAlreadyExists(err) {
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

	// get the namespace for the cluster we're restoring from
	restoreClusterNamespace := operator.GetBootstrapNamespace(cluster)

	found := true
	repoDeployment, err := clientset.AppsV1().Deployments(restoreClusterNamespace).
		Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return
		}
		found = false
	}

	if !found {
		if err = backrest.CreateRepoDeployment(clientset, cluster, false, true, 1,
			restoreClusterNamespace); err != nil {
			return
		}
		repoCreated = true
	} else if _, ok := repoDeployment.GetLabels()[config.LABEL_PGHA_BOOTSTRAP]; ok {
		err = fmt.Errorf("unable to create bootstrap repo %s to bootstrap cluster %s "+
			"(namespace %s) because it is already running to bootstrap another cluster",
			repoName, cluster.GetName(), cluster.GetNamespace())
		return
	}

	return
}

// ResizeClusterPVC allows for the resizing of all the PVCs across the cluster.
// This draws a distinction from PVCs in the cluster that may be sized
// independently, e.g. replicas.
//
// If there are instances that are sized differently than the primary, we ensure
// that the size is kept consistent. In other words:
//
// - If instance is sized consistently with the cluster, we will resize it.
// - If the instance has its size set independently, we will check to see if
//   that size is smaller than the cluster PVC resize. if it is smaller, then we
//   will size it to match the cluster PVC
func ResizeClusterPVC(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	log.Debugf("resize cluster PVC on [%s]", deployment.Name)
	ctx := context.TODO()

	// we can ignore the error here as this has to have been validated before
	// reaching this step. However, if you reach this step and are getting an
	// error, you're likely invoking this function improperly. Sorry.
	clusterSize, _ := resource.ParseQuantity(cluster.Spec.PrimaryStorage.Size)

	// determine if this deployment represents an individual instance
	if instance, err := clientset.CrunchydataV1().Pgreplicas(cluster.Namespace).Get(ctx,
		deployment.GetName(), metav1.GetOptions{}); err == nil {

		// get the instanceSize. If there is an error parsing this, then we most
		// certainly will take the clusterSize
		if instanceSize, err := resource.ParseQuantity(instance.Spec.ReplicaStorage.Size); err != nil || clusterSize.Cmp(instanceSize) == 1 {
			// ok, so let's update the instance with the new size, but ensure that we
			// do not try to resize it a second time
			annotations := instance.ObjectMeta.GetAnnotations()
			if annotations == nil {
				annotations = map[string]string{}
			}
			annotations[config.ANNOTATION_CLUSTER_DO_NOT_RESIZE] = config.LABEL_TRUE
			instance.ObjectMeta.SetAnnotations(annotations)

			// set the size
			instance.Spec.ReplicaStorage.Size = cluster.Spec.PrimaryStorage.Size

			// and update
			if _, err := clientset.CrunchydataV1().Pgreplicas(instance.Namespace).Update(ctx,
				instance, metav1.UpdateOptions{}); err != nil {
				// if we cannot update the instance spec, we should warn that we were
				// unable to perform the resize, but not block any other action
				log.Errorf("could not resize instance %q: %s", deployment.GetName(), err.Error())

				return nil
			}
		} else {
			// we are skipping this. we don't need to error, just inform
			msg := "instance size is larger than that of cluster size"
			if err != nil {
				msg = err.Error()
			}

			log.Infof("skipping pvc resize of instance %q: %s", deployment.GetName(), msg)

			return nil
		}
	}

	// OK, let's now perform the resize. In this case, we need to update the value
	// on the PVC.
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(cluster.Namespace).Get(ctx,
		deployment.GetName(), metav1.GetOptions{})

	// if we can't locate the PVC, we can't resize, and we really need to return
	// an error
	if err != nil {
		return err
	}

	// alright, update the PVC size
	pvc.Spec.Resources.Requests[v1.ResourceStorage] = clusterSize

	// and update!
	if _, err := clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx,
		pvc, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// ResizeWALPVC allows for the resizing of the PVCs where WAL are stored.
func ResizeWALPVC(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	log.Debugf("resize cluster PVC on [%s]", deployment.Name)
	ctx := context.TODO()

	// we can ignore the error here as this has to have been validated before
	// reaching this step. However, if you reach this step and are getting an
	// error, you're likely invoking this function improperly. Sorry.
	size, _ := resource.ParseQuantity(cluster.Spec.WALStorage.Size)

	// OK, let's now perform the resize. In this case, we need to update the value
	// on the PVC.
	pvcName := deployment.GetName() + "-wal"
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(cluster.Namespace).Get(ctx,
		pvcName, metav1.GetOptions{})

	// if we can't locate the PVC, we can't resize, and we really need to return
	// an error
	if err != nil {
		return err
	}

	// alright, update the PVC size
	pvc.Spec.Resources.Requests[v1.ResourceStorage] = size

	// and update!
	if _, err := clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx,
		pvc, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
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
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster, replica)
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
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster, replica)
		return
	}

	// instantiate the replica
	if err = scaleReplicaCreateDeployment(clientset, replica, cluster, namespace, dataVolume, walVolume, tablespaceVolumes); err != nil {
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], cluster, replica)
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
		Clustername: cluster.Name,
		Replicaname: replica.Name,
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

// ToggleTLS will toggle the appropriate Postgres configuration in a DCS file
// around the TLS settings
func ToggleTLS(clientset kubeapi.Interface, cluster *crv1.Pgcluster) error {
	// get the ConfigMap that stores the configuration
	cm, err := getClusterConfigMap(clientset, cluster)

	if err != nil {
		return err
	}

	dcs, dcsConfig, err := getDCSConfig(clientset, cluster, cm)

	if err != nil {
		return err
	}

	// alright, the great toggle.
	if cluster.Spec.TLS.IsTLSEnabled() {
		dcsConfig.PostgreSQL.Parameters["ssl"] = "on"
		dcsConfig.PostgreSQL.Parameters["ssl_cert_file"] = "/pgconf/tls/tls.crt"
		dcsConfig.PostgreSQL.Parameters["ssl_key_file"] = "/pgconf/tls/tls.key"
		dcsConfig.PostgreSQL.Parameters["ssl_ca_file"] = "/pgconf/tls/ca.crt"
	} else {
		dcsConfig.PostgreSQL.Parameters["ssl"] = "off"
		delete(dcsConfig.PostgreSQL.Parameters, "ssl_cert_file")
		delete(dcsConfig.PostgreSQL.Parameters, "ssl_key_file")
		delete(dcsConfig.PostgreSQL.Parameters, "ssl_ca_file")
	}

	return dcs.Update(dcsConfig)
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

// UpdateBackrestS3 updates any pgBackRest settings that may have been updated.
// Presently this is just the bucket name.
func UpdateBackrestS3(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	// go through the environmetnal variables on the "database" container and edit
	// the appropriate S3 related ones
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name != "database" {
			continue
		}

		for j, envVar := range deployment.Spec.Template.Spec.Containers[i].Env {
			if envVar.Name == "PGBACKREST_REPO1_S3_BUCKET" {
				deployment.Spec.Template.Spec.Containers[i].Env[j].Value = cluster.Spec.BackrestS3Bucket
			}
		}
	}

	return nil
}

// UpdateLabels updates the labels on the template to match those of the custom
// labels
func UpdateLabels(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	log.Debugf("update labels on [%s]", deployment.Name)

	labels := map[string]string{}

	// ...so, try to get all of the "system labels" copied over
	for _, k := range systemLabels {
		labels[k] = deployment.Spec.Template.ObjectMeta.Labels[k]
	}

	// now get the custom labels
	for k, v := range util.GetCustomLabels(cluster) {
		labels[k] = v
	}

	log.Debugf("new labels: %v", labels)

	// set the labels on the deployment object
	deployment.Spec.Template.ObjectMeta.SetLabels(labels)

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

// UpdateTLS updates whether or not TLS is enabled, and if certain attributes
// are set (i.e. TLSOnly), ensures that the changes are properly reflected on
// the containers
func UpdateTLS(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment) error {
	// get the ConfigMap that stores the configuration
	cm, err := getClusterConfigMap(clientset, cluster)

	// if we can't edit the ConfigMap, we can't proceed as the cluster can end up
	// in a weird state
	if err != nil {
		log.Error(err)
		return err
	}

	// it's easier to remove all of the settings associated with a TLS cluster
	// before applying the new settings. So no matter what remove
	if err := disableTLS(clientset, deployment, cm); err != nil {
		log.Error(err)
		return err
	}

	// determine if TLS needs to be enabled
	if !cluster.Spec.TLS.IsTLSEnabled() {
		return nil
	}

	return enableTLS(clientset, cluster, deployment, cm)
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
		config.ANNOTATION_GCS_BUCKET:          cfg(cl.BackrestGCSBucket, op.BackrestGCSBucket),
		config.ANNOTATION_GCS_ENDPOINT:        cfg(cl.BackrestGCSEndpoint, op.BackrestGCSEndpoint),
		config.ANNOTATION_GCS_KEY_TYPE:        cfg(cl.BackrestGCSKeyType, op.BackrestGCSKeyType),
	}).Bytes()

	if err == nil {
		log.Debugf("patching secret %s: %s", secretName, patch)
		_, err = clientset.CoreV1().Secrets(namespace).
			Patch(ctx, secretName, types.MergePatchType, patch, metav1.PatchOptions{})
	}
	return err
}

// createBootstrapBackRestSecret creates a copy of the pgBackRest secret from the source cluster
// being utilized to bootstrap a new cluster.  This ensures the required Secret (and therefore the
// required pgBackRest cofiguration) as needed to bootstrap a new cluster via a 'pgbackrest
// restore' is always present in the namespace of the cluster being created (e.g. when
// bootstrapping from the pgBackRest backups of a cluster in another namespace)
func createBootstrapBackRestSecret(clientset kubernetes.Interface,
	cluster *crv1.Pgcluster) (*v1.Secret, error) {
	ctx := context.TODO()

	restoreFromCluster := cluster.Spec.PGDataSource.RestoreFrom

	// Get the proper namespace depending on where we're restoring from.  If no namespace is
	// specified in the PGDataSource then assume the same namespace as the pgcluster.
	restoreFromNamespace := operator.GetBootstrapNamespace(cluster)

	// get a copy of the pgBackRest repo secret for the cluster we're restoring from
	restoreFromSecretName := fmt.Sprintf("%s-%s", restoreFromCluster,
		config.LABEL_BACKREST_REPO_SECRET)
	restoreFromSecret, err := clientset.CoreV1().Secrets(restoreFromNamespace).Get(ctx,
		restoreFromSecretName, metav1.GetOptions{})
	if err != nil {
		publishClusterCreateFailure(cluster, err.Error())
		return nil, err
	}

	// Create a copy of the secret for the cluster being recreated.  This ensures a copy of the
	// required pgBackRest Secret is always present is the namespace of the cluster being created.
	secretCopyName := fmt.Sprintf(util.BootstrapConfigPrefix, cluster.GetName(),
		config.LABEL_BACKREST_REPO_SECRET)
	secretCopy := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: restoreFromSecret.GetAnnotations(),
			Labels: map[string]string{
				config.LABEL_VENDOR:            config.LABEL_CRUNCHY,
				config.LABEL_PG_CLUSTER:        cluster.GetName(),
				config.LABEL_PGO_BACKREST_REPO: "true",
				config.LABEL_PGHA_BOOTSTRAP:    cluster.GetName(),
			},
			Name:      secretCopyName,
			Namespace: cluster.GetNamespace(),
		},
		Data: restoreFromSecret.Data,
	}

	for k, v := range util.GetCustomLabels(cluster) {
		secretCopy.ObjectMeta.Labels[k] = v
	}

	return clientset.CoreV1().Secrets(cluster.GetNamespace()).Create(ctx, secretCopy,
		metav1.CreateOptions{})
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
	if _, err := clientset.CoreV1().Secrets(cluster.Namespace).Get(
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
		username, password, cluster.Namespace, util.GetCustomLabels(cluster))
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

// disableTLS unmounts any TLS Secrets from a Postgres cluster and will ensure
// that TLSOnly is set to false
func disableTLS(clientset kubeapi.Interface, deployment *apps_v1.Deployment, cm *v1.ConfigMap) error {
	// first, set the environmental variables that are associated with TLS
	// enablement to "false"
	for i, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
		switch envVar.Name {
		default:
			continue
		case tlsEnvVarEnabled, tlsEnvVarOnly:
			deployment.Spec.Template.Spec.Containers[0].Env[i].Value = "false"
		}
	}

	// next, remove any of the TLS secrets from the volume mounts
	volumeMounts := make([]v1.VolumeMount, 0)

	for _, volumeMount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		switch volumeMount.Name {
		default:
			volumeMounts = append(volumeMounts, volumeMount)
		case tlsVolumeServer, tlsVolumeReplication:
			continue
		}
	}

	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	// finally, remove any of the TLS volumes
	volumes := make([]v1.Volume, 0)

	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		switch volume.Name {
		default:
			volumes = append(volumes, volume)
		case tlsVolumeServer, tlsVolumeReplication:
			continue
		}
	}

	deployment.Spec.Template.Spec.Volumes = volumes

	// disable TLS in the instance configuration settings, particularly the HBA
	// portion
	localDB, localConfig, err := getLocalConfig(clientset, cm, deployment.GetName())

	if err != nil {
		log.Error(err)
		return err
	}

	// remove any entry that has "hostssl" in it...but check to see if there is a
	// corresponding entry with "host". So this is kind of a costly operation, but
	// it's small
	hba := make([]string, 0)

	for _, rule := range localConfig.PostgreSQL.PGHBA {
		// if this is not a TLS entry, append and continue on
		if !strings.HasPrefix(rule, "hostssl") {
			hba = append(hba, rule)
			continue
		}

		// ok, if this is a TLS entry, we are going to remove it as it is, but we
		// we may need to convert it to a "host" entry if there is no corresponding
		// entry
		expectedHostRule := tlsHBAPattern.ReplaceAllLiteralString(rule, "host")

		if !findHBARule(localConfig.PostgreSQL.PGHBA, expectedHostRule) {
			hba = append(hba, expectedHostRule)
		}
	}

	// update the HBA rules for the instance
	localConfig.PostgreSQL.PGHBA = hba

	// and push the update into the ConfigMap
	return localDB.Update(getLocalConfigName(deployment.GetName()), localConfig)
}

// enableTLS performs all of the actions required to add TLS to a Postgres
// cluster. This includes mounting the Secrets and ensuring that any env vars
// that are required are set.
func enableTLS(clientset kubeapi.Interface, cluster *crv1.Pgcluster, deployment *apps_v1.Deployment, cm *v1.ConfigMap) error {
	// first, set the environmental variables that are associated with TLS
	// enablement to "true" as needed. if the variables are not set, ensure they
	// are set
	var foundEnabled, foundOnly bool

	for i, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
		switch envVar.Name {
		default:
			continue
		case tlsEnvVarEnabled:
			deployment.Spec.Template.Spec.Containers[0].Env[i].Value = "true"
			foundEnabled = true
		case tlsEnvVarOnly:
			deployment.Spec.Template.Spec.Containers[0].Env[i].Value = "false"
			if cluster.Spec.TLSOnly {
				deployment.Spec.Template.Spec.Containers[0].Env[i].Value = "true"
			}
			foundOnly = true
		}
	}

	if !foundEnabled {
		envVar := v1.EnvVar{
			Name:  tlsEnvVarEnabled,
			Value: "true",
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, envVar)
	}

	if !foundOnly {
		envVar := v1.EnvVar{
			Name:  tlsEnvVarOnly,
			Value: "false",
		}
		if cluster.Spec.TLSOnly {
			envVar.Value = "true"
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, envVar)
	}

	// next, add the required TLS volume mounts
	volumeMounts := make([]v1.VolumeMount, 0)
	volumeMounts = append(volumeMounts,
		v1.VolumeMount{
			Name:      tlsVolumeServer,
			MountPath: tlsMountPathServer,
		},
	)

	if cluster.Spec.TLS.ReplicationTLSSecret != "" {
		volumeMounts = append(volumeMounts,
			v1.VolumeMount{
				Name:      tlsVolumeReplication,
				MountPath: tlsMountPathReplication,
			},
		)
	}

	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		volumeMounts...,
	)

	// finally, mount the actual TLS volumes
	// if there is a replication secret, we mount it as part of the project Secret
	// and on its own in order to handle the libpq stuff
	volume := v1.Volume{
		Name: tlsVolumeServer,
	}
	defaultMode := int32(0o440)
	volume.Projected = &v1.ProjectedVolumeSource{
		DefaultMode: &defaultMode,
		Sources: []v1.VolumeProjection{
			{
				Secret: &v1.SecretProjection{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cluster.Spec.TLS.TLSSecret,
					},
				},
			},
			{
				Secret: &v1.SecretProjection{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cluster.Spec.TLS.CASecret,
					},
				},
			},
		},
	}

	if cluster.Spec.TLS.ReplicationTLSSecret != "" {
		volume.Projected.Sources = append(volume.Projected.Sources,
			v1.VolumeProjection{
				Secret: &v1.SecretProjection{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cluster.Spec.TLS.ReplicationTLSSecret,
					},
					Items: []v1.KeyToPath{
						{
							Key:  "tls.key",
							Path: "tls-replication.key",
						},
						{
							Key:  "tls.crt",
							Path: "tls-replication.crt",
						},
					},
				},
			},
		)
	}

	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

	if cluster.Spec.TLS.ReplicationTLSSecret != "" {
		volume := v1.Volume{
			Name: tlsVolumeReplication,
		}
		volume.EmptyDir = &v1.EmptyDirVolumeSource{
			Medium:    v1.StorageMediumMemory,
			SizeLimit: &tlsVolumeReplicationSizeLimit,
		}
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)
	}

	// enable TLS in the instance configuration settings, particularly the HBA
	// portion
	localDB, localConfig, err := getLocalConfig(clientset, cm, deployment.GetName())

	if err != nil {
		log.Error(err)
		return err
	}

	// need to break out if this is TLS only mode or not. And order matters in the
	// HBA file.

	hba := make([]string, 0)

	// first, extract any of the "local" settings and give them priority
	for _, rule := range localConfig.PostgreSQL.PGHBA {
		// if this is not a TLS entry, append and continue on
		if !strings.HasPrefix(rule, "local") {
			continue
		}

		hba = append(hba, rule)
	}

	// now, insert our predefined TLS rules
	hba = append(hba, tlsHBARules...)

	// OK, insert the rest of the rules, though if this is TLS only, we'll convert
	// a "host" rule to be TLS.
	for _, rule := range localConfig.PostgreSQL.PGHBA {
		// skip local :)
		if strings.HasPrefix(rule, "local") {
			continue
		}

		// if this is rule is already TLS enabled, then add it and continue, though
		// first check to see if it's already in the list
		if !notlsHBAPattern.MatchString(rule) {
			if !findHBARule(hba, rule) {
				hba = append(hba, rule)
			}

			continue
		}

		// so this is now a "host" pattern, so we need to check for TLS only. If this
		// is not TLS only, we can just add the rule, provided it does not
		expectedRule := rule
		if cluster.Spec.TLSOnly {
			expectedRule = notlsHBAPattern.ReplaceAllLiteralString(rule, "hostssl ")
		}

		// check to see if this rule already exists in the updated hba list. If
		// so, then we can ignore, otherwise we convert to TLS
		if !findHBARule(hba, expectedRule) {
			hba = append(hba, expectedRule)
		}
	}

	// update the HBA rules for the instance
	localConfig.PostgreSQL.PGHBA = hba

	// and push the update into the ConfigMap
	return localDB.Update(getLocalConfigName(deployment.GetName()), localConfig)
}

// findHBARule sees if a HBA rule exists in a list. If it does, return true
func findHBARule(hba []string, rule string) bool {
	for _, r := range hba {
		if r == rule {
			return true
		}
	}

	return false
}

// getClusterConfigMap returns the configmap that stores the configuration of a
// cluster
func getClusterConfigMap(clientset kubeapi.Interface, cluster *crv1.Pgcluster) (*v1.ConfigMap, error) {
	ctx := context.TODO()
	return clientset.CoreV1().ConfigMaps(cluster.Namespace).Get(ctx,
		cluster.Name+"-pgha-config", metav1.GetOptions{})
}

// getDCSConfig gets the global configuration for a cluster, which can be used
// to update said configuration
func getDCSConfig(clientset kubeapi.Interface, cluster *crv1.Pgcluster, cm *v1.ConfigMap) (*cfg.DCS, *cfg.DCSConfig, error) {
	dcs := cfg.NewDCS(cm, clientset, cluster.Name)

	// now, get the local configuration for that cluster. We get this
	// from the deployment name
	dcsConfig, _, err := dcs.GetDCSConfig()

	if err != nil {
		return nil, nil, err
	}

	return dcs, dcsConfig, nil
}

// getLocalConfig gets the local configuration for a specific instance in a
// cluster, which can be used to update said configuration
func getLocalConfig(clientset kubeapi.Interface, cm *v1.ConfigMap, instanceName string) (*cfg.LocalDB, *cfg.LocalDBConfig, error) {
	// need to load the rest configuration
	restConfig, err := kubeapi.LoadClientConfig()
	if err != nil {
		return nil, nil, err
	}

	localDB, err := cfg.NewLocalDB(cm, restConfig, clientset)

	if err != nil {
		return nil, nil, err
	}

	// now, get the local configuration for that cluster. We get this
	// from the deployment name
	localConfig, err := localDB.GetLocalConfigFromCluster(getLocalConfigName(instanceName))

	if err != nil {
		return nil, nil, err
	}

	return localDB, localConfig, nil
}

// getLocalConfigName gets the name of the entry in the local configuration file
func getLocalConfigName(instanceName string) string {
	return fmt.Sprintf(cfg.PGHALocalConfigName, instanceName)
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
func StopPostgreSQLInstance(clientset kubernetes.Interface, restConfig *rest.Config, deployment apps_v1.Deployment) error {
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
