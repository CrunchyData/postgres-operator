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

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
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

	//err = cleanupPreviousTasks(client, cl.Spec.Name, namespace)

	var pvcName string

	_, found, err := kubeapi.GetPVC(clientset, cl.Spec.Name, namespace)
	if found {
		log.Debugf("pvc [%s] already present from previous cluster with this same name, will not recreate", cl.Spec.Name)
		pvcName = cl.Spec.Name
	} else {
		pvcName, err = pvc.CreatePVC(clientset, &cl.Spec.PrimaryStorage, cl.Spec.Name, cl.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			publishClusterCreateFailure(cl, err.Error())
			return
		}
		log.Debugf("created primary pvc [%s]", pvcName)
	}

	// iterate through all of the tablespaces and attempt to create their PVCs
	// for this cluster
	for tablespaceName, storageSpec := range cl.Spec.TablespaceMounts {
		// first, generate the tablespace PVC name from the cluster deployment name
		// and the name of the tablespace
		tablespacePVCName := operator.GetTablespacePVCName(cl.Spec.Name, tablespaceName)
		// attempt to create the tablespace PVC. If it fails to create, log the
		// error and publish the failure event
		if err := CreateTablespacePVC(clientset, namespace, cl.Spec.Name, tablespacePVCName, &storageSpec); err != nil {
			log.Error(err)
			publishClusterCreateFailure(cl, err.Error())
			return
		}
	}

	//replaced with ccpimagetag instead of pg version

	AddCluster(clientset, client, cl, namespace, pvcName)

	err = util.Patch(client, "/spec/status", crv1.CompletedStatus, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}
	err = util.Patch(client, "/spec/PrimaryStorage/name", pvcName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
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

	//add pgbouncer deployment if requested
	// if cl.Spec.UserLabels[config.LABEL_PGBOUNCER] == "true" {
	if cl.Labels[config.LABEL_PGBOUNCER] == "true" {
		log.Debug("pgbouncer requested")
		//create the pgbouncer deployment using that credential
		AddPgbouncer(clientset, client, cl, namespace, true, false)

		// create the task to update db authorizations after pg container goes ready....
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
			//get the resource config
			spec.ContainerResources = cl.Spec.ContainerResources
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

	//delete any existing pgbackups
	pgback := crv1.Pgbackup{}
	found, err := kubeapi.Getpgbackup(restclient, &pgback, cl.Spec.Name, namespace)
	if found {
		kubeapi.Deletepgbackup(restclient, cl.Spec.Name, namespace)
	}

	//delete any existing configmaps
	if err = deleteConfigMaps(clientset, cl.Spec.Name, namespace); err != nil {
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

	err = events.Publish(f)
	if err != nil {
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

	var pvcName string
	// create the PVC if necessary.  When a replica is being created during a restore, the PVC will already exist.
	// Otherwise a new PVC will be created.
	_, found, err := kubeapi.GetPVC(clientset, replica.Spec.Name, namespace)
	if found {
		log.Debugf("pvc [%s] already present for replica from previous cluster with this same name, will not recreate",
			replica.Spec.Name)
		pvcName = replica.Spec.Name
	} else {
		pvcName, err = pvc.CreatePVC(clientset, &replica.Spec.ReplicaStorage, replica.Spec.Name, cluster.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debugf("created replica pvc [%s]", pvcName)
	}

	//update the replica CRD pvcname
	err = util.Patch(client, "/spec/replicastorage/name", pvcName, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	//create the replica service if it doesnt exist

	st := operator.Pgo.Cluster.ServiceType

	if replica.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = replica.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	} else if cluster.Spec.UserLabels[config.LABEL_SERVICE_TYPE] != "" {
		st = cluster.Spec.UserLabels[config.LABEL_SERVICE_TYPE]
	}

	serviceName := replica.Spec.ClusterName + "-replica"
	serviceFields := ServiceTemplateFields{
		Name:         serviceName,
		ServiceName:  serviceName,
		ClusterName:  replica.Spec.ClusterName,
		Port:         cluster.Spec.Port,
		PGBadgerPort: cluster.Spec.PGBadgerPort,
		ExporterPort: cluster.Spec.ExporterPort,
		ServiceType:  st,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	//instantiate the replica
	Scale(clientset, client, replica, namespace, pvcName, &cluster)

	//update the replica CRD status
	err = util.Patch(client, "/spec/status", crv1.CompletedStatus, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
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

func cleanupPreviousTasks(client *rest.RESTClient, clusterName, namespace string) error {

	selector := config.LABEL_PG_CLUSTER + "=" + clusterName
	taskList := crv1.PgtaskList{}

	err := kubeapi.GetpgtasksBySelector(client, &taskList, selector, namespace)
	if err != nil {
		return err
	}

	for _, t := range taskList.Items {
		err = kubeapi.Deletepgtask(client, t.Name, namespace)
		if err != nil {
			log.Error(err)
		}
	}
	return err
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
