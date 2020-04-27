// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// contstants defining the names of the various sidecar containers
const (
	collectCCPImage    = "crunchy-collect"
	pgBadgerCCPImage   = "crunchy-pgbadger"
	crunchyadmCCPImage = "crunchy-admin"
)

func AddClusterBase(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) {
	var err error

	if cl.Spec.Status == crv1.CompletedStatus {
		errorMsg := "crv1 pgcluster " + cl.Spec.ClusterName + " is already marked complete, will not recreate"
		log.Warn(errorMsg)
		publishClusterCreateFailure(cl, errorMsg)
		return
	}

	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, cl, namespace, cl.Spec.Name, cl.Spec.PrimaryStorage)
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

	if err = addClusterCreateDeployments(clientset, client, cl, namespace, dataVolume, walVolume, tablespaceVolumes); err != nil {
		publishClusterCreateFailure(cl, err.Error())
		return
	}

	err = util.Patch(client, "/spec/status", crv1.CompletedStatus, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}
	err = util.Patch(client, "/spec/PrimaryStorage/name", dataVolume.PersistentVolumeClaimName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
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

	//add replicas if requested
	if cl.Spec.Replicas != "" {
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

			//check for replica node label in pgo.yaml
			if operator.Pgo.Cluster.ReplicaNodeLabel != "" {
				parts := strings.Split(operator.Pgo.Cluster.ReplicaNodeLabel, "=")
				spec.UserLabels[config.LABEL_NODE_LABEL_KEY] = parts[0]
				spec.UserLabels[config.LABEL_NODE_LABEL_VALUE] = parts[1]
				log.Debug("using pgo.yaml ReplicaNodeLabel for replica creation")
			}

			labels := make(map[string]string)
			labels[config.LABEL_PG_CLUSTER] = cl.Spec.Name

			spec.ClusterName = cl.Spec.Name
			uniqueName := util.RandStringBytesRmndr(4)
			labels[config.LABEL_NAME] = cl.Spec.Name + "-" + uniqueName
			spec.Name = labels[config.LABEL_NAME]
			newInstance := &crv1.Pgreplica{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   labels[config.LABEL_NAME],
					Labels: labels,
				},
				Spec: spec,
				Status: crv1.PgreplicaStatus{
					State:   crv1.PgreplicaStateCreated,
					Message: "Created, not processed yet",
				},
			}
			result := crv1.Pgreplica{}

			err = client.Post().
				Resource(crv1.PgreplicaResourcePlural).
				Namespace(namespace).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error(" in creating Pgreplica instance" + err.Error())
				publishClusterCreateFailure(cl, err.Error())
			}

		}
	}

}

// DeleteClusterBase ...
func DeleteClusterBase(clientset *kubernetes.Clientset, restclient *rest.RESTClient, cl *crv1.Pgcluster, namespace string) {

	DeleteCluster(clientset, restclient, cl, namespace)

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
func ScaleBase(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace string) {
	var err error

	if replica.Spec.Status == crv1.CompletedStatus {
		log.Warn("crv1 pgreplica " + replica.Spec.Name + " is already marked complete, will not recreate")
		return
	}

	//get the pgcluster CRD to base the replica off of
	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(client, &cluster,
		replica.Spec.ClusterName, namespace)
	if err != nil {
		return
	}

	dataVolume, walVolume, tablespaceVolumes, err := pvc.CreateMissingPostgreSQLVolumes(
		clientset, &cluster, namespace, replica.Spec.Name, replica.Spec.ReplicaStorage)
	if err != nil {
		log.Error(err)
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], &cluster)
		return
	}

	//update the replica CRD pvcname
	err = util.Patch(client, "/spec/replicastorage/name", dataVolume.PersistentVolumeClaimName, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	//create the replica service if it doesnt exist
	if err = scaleReplicaCreateMissingService(clientset, replica, &cluster, namespace); err != nil {
		log.Error(err)
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], &cluster)
		return
	}

	//instantiate the replica
	if err = scaleReplicaCreateDeployment(clientset, client, replica, &cluster, namespace, dataVolume, walVolume, tablespaceVolumes); err != nil {
		publishScaleError(namespace, replica.ObjectMeta.Labels[config.LABEL_PGOUSER], &cluster)
		return
	}

	//update the replica CRD status
	err = util.Patch(client, "/spec/status", crv1.CompletedStatus, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
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
func ScaleDownBase(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace string) {
	var err error

	//get the pgcluster CRD for this replica
	cluster := crv1.Pgcluster{}
	_, err = kubeapi.Getpgcluster(client, &cluster,
		replica.Spec.ClusterName, namespace)
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

// UpdateResources updates the PostgreSQL instance Deployments to reflect the
// update resources (i.e. CPU, memory)
func UpdateResources(clientset *kubernetes.Clientset, restConfig *rest.Config, cluster *crv1.Pgcluster) error {
	// get a list of all of the instance deployments for the cluster
	deployments, err := operator.GetInstanceDeployments(clientset, cluster)

	if err != nil {
		return err
	}

	// iterate through each PostgreSQL instance deployment and update the
	// resource values for the database container
	//
	// NOTE: a future version (near future) will first try to detect the primary
	// so that all the replicas are updated first, and then the primary gets the
	// update
	for _, deployment := range deployments.Items {
		requestResourceList := v1.ResourceList{}
		limitResourceList := v1.ResourceList{}

		// if there is a request / limit resource list available already, use that
		// one
		// NOTE: this works as the "database" container is always first
		if deployment.Spec.Template.Spec.Containers[0].Resources.Requests != nil {
			requestResourceList = deployment.Spec.Template.Spec.Containers[0].Resources.Requests
		}
		if deployment.Spec.Template.Spec.Containers[0].Resources.Limits != nil {
			limitResourceList = deployment.Spec.Template.Spec.Containers[0].Resources.Limits
		}

		// handle the CPU update. For the CPU updates, we set both set/unset the
		// request and the limit
		if resource, ok := cluster.Spec.Resources[v1.ResourceCPU]; ok {
			requestResourceList[v1.ResourceCPU] = resource
			limitResourceList[v1.ResourceCPU] = resource
		} else {
			delete(requestResourceList, v1.ResourceCPU)
			delete(limitResourceList, v1.ResourceCPU)
		}

		// handle the memory update
		if resource, ok := cluster.Spec.Resources[v1.ResourceMemory]; ok {
			requestResourceList[v1.ResourceMemory] = resource
		} else {
			delete(requestResourceList, v1.ResourceMemory)
		}

		// update the requests resourcelist
		deployment.Spec.Template.Spec.Containers[0].Resources.Requests = requestResourceList
		deployment.Spec.Template.Spec.Containers[0].Resources.Limits = limitResourceList

		// Before applying the update, we want to explicitly stop PostgreSQL on each
		// instance. This prevents PostgreSQL from having to boot up in crash
		// recovery mode.
		//
		// If an error is returned, we only issue a warning
		if err := stopPostgreSQLInstance(clientset, restConfig, deployment); err != nil {
			log.Warn(err)
		}

		// update the deployment with the new values
		if err := kubeapi.UpdateDeployment(clientset, &deployment); err != nil {
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
func UpdateTablespaces(clientset *kubernetes.Clientset, restConfig *rest.Config,
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
		if err := kubeapi.UpdateDeployment(clientset, &deployment); err != nil {
			return err
		}
	}

	return nil
}

func deleteConfigMaps(clientset *kubernetes.Clientset, clusterName, ns string) error {
	label := fmt.Sprintf("pg-cluster=%s", clusterName)
	list, ok := kubeapi.ListConfigMap(clientset, label, ns)
	if !ok {
		return fmt.Errorf("No configMaps found for selector: %s", label)
	}

	for _, configmap := range list.Items {
		err := kubeapi.DeleteConfigMap(clientset, configmap.Name, ns)
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
func stopPostgreSQLInstance(clientset *kubernetes.Clientset, restConfig *rest.Config, deployment apps_v1.Deployment) error {
	// First, attempt to get the PostgreSQL instance Pod attachd to this
	// particular deployment
	selector := fmt.Sprintf("%s=%s", config.LABEL_DEPLOYMENT_NAME, deployment.Name)
	pods, err := kubeapi.GetPods(clientset, selector, deployment.ObjectMeta.Namespace)

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
