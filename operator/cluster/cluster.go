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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strconv"
	"strings"
)

// Strategy ....
type Strategy interface {
	Scale(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgreplica, string, string, *crv1.Pgcluster) error
	AddCluster(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, string, string) error
	Failover(*kubernetes.Clientset, *rest.RESTClient, string, *crv1.Pgtask, string, *rest.Config) error
	DeleteCluster(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, string) error
	DeleteReplica(*kubernetes.Clientset, *crv1.Pgreplica, string) error

	MinorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *crv1.Pgcluster, *crv1.Pgupgrade, string) error
	UpdatePolicyLabels(*kubernetes.Clientset, string, string, map[string]string) error
}

// ServiceTemplateFields ...
type ServiceTemplateFields struct {
	Name        string
	ServiceName string
	ClusterName string
	Port        string
	ServiceType string
}

// ReplicaSuffix ...
const ReplicaSuffix = "-replica"

var strategyMap map[string]Strategy

func init() {
	strategyMap = make(map[string]Strategy)
	strategyMap["1"] = Strategy1{}
}

// AddClusterBase ...
func AddClusterBase(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) {
	var err error

	if cl.Spec.Status == crv1.UpgradeCompletedStatus {
		log.Warn("crv1 pgcluster " + cl.Spec.ClusterName + " is already marked complete, will not recreate")
		return
	}

	var pvcName string

	_, found, err := kubeapi.GetPVC(clientset, cl.Spec.Name, namespace)
	if found {
		log.Debugf("pvc [%s] already present from previous cluster with this same name, will not recreate", cl.Spec.Name)
		pvcName = cl.Spec.Name
	} else {
		pvcName, err = pvc.CreatePVC(clientset, &cl.Spec.PrimaryStorage, cl.Spec.Name, cl.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debugf("created primary pvc [%s]", pvcName)
	}

	if cl.Spec.UserLabels[util.LABEL_ARCHIVE] == "true" {
		pvcName := cl.Spec.Name + "-xlog"
		_, found, err = kubeapi.GetPVC(clientset, pvcName, namespace)
		if found {
			log.Debugf("pvc [%s] already present from previous cluster with this same name, will not recreate", pvcName)
		} else {
			storage := crv1.PgStorageSpec{}
			pgoStorage := operator.Pgo.Storage[operator.Pgo.XlogStorage]
			storage.StorageClass = pgoStorage.StorageClass
			storage.AccessMode = pgoStorage.AccessMode
			storage.Size = pgoStorage.Size
			storage.StorageType = pgoStorage.StorageType
			storage.MatchLabels = pgoStorage.MatchLabels
			storage.SupplementalGroups = pgoStorage.SupplementalGroups
			storage.Fsgroup = pgoStorage.Fsgroup
			_, err := pvc.CreatePVC(clientset, &storage, pvcName, cl.Spec.Name, namespace)
			if err != nil {
				log.Error(err)
				return
			}
		}
	}

	log.Debugf("creating Pgcluster object strategy is [%s]", cl.Spec.Strategy)
	//allows user to override with their own passwords

	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
		return
	}

	//replaced with ccpimagetag instead of pg version

	strategy.AddCluster(clientset, client, cl, namespace, pvcName)

	err = util.Patch(client, "/spec/status", crv1.UpgradeCompletedStatus, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}
	err = util.Patch(client, "/spec/PrimaryStorage/name", pvcName, crv1.PgclusterResourcePlural, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	log.Debugf("before pgpool check [%s]", cl.Spec.UserLabels[util.LABEL_PGPOOL])
	//add pgpool deployment if requested
	if cl.Spec.UserLabels[util.LABEL_PGPOOL] == "true" {
		log.Debug("pgpool requested")
		//create the pgpool deployment using that credential
		AddPgpool(clientset, cl, namespace, true)
	}
	//add pgbouncer deployment if requested
	if cl.Spec.UserLabels[util.LABEL_PGBOUNCER] == "true" {
		log.Debug("pgbouncer requested")
		//create the pgbouncer deployment using that credential
		AddPgbouncer(clientset, cl, namespace, true)
	}

	//add replicas if requested
	if cl.Spec.Replicas != "" {
		replicaCount, err := strconv.Atoi(cl.Spec.Replicas)
		if err != nil {
			log.Error("error in replicas value " + err.Error())
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
			spec.UserLabels[util.LABEL_NODE_LABEL_KEY] = ""
			spec.UserLabels[util.LABEL_NODE_LABEL_VALUE] = ""

			//check for replica node label in pgo.yaml
			if operator.Pgo.Cluster.ReplicaNodeLabel != "" {
				parts := strings.Split(operator.Pgo.Cluster.ReplicaNodeLabel, "=")
				spec.UserLabels[util.LABEL_NODE_LABEL_KEY] = parts[0]
				spec.UserLabels[util.LABEL_NODE_LABEL_VALUE] = parts[1]
				log.Debug("using pgo.yaml ReplicaNodeLabel for replica creation")
			}

			labels := make(map[string]string)
			labels[util.LABEL_PG_CLUSTER] = cl.Spec.Name

			spec.ClusterName = cl.Spec.Name
			uniqueName := util.RandStringBytesRmndr(4)
			labels[util.LABEL_NAME] = cl.Spec.Name + "-" + uniqueName
			spec.Name = labels[util.LABEL_NAME]
			newInstance := &crv1.Pgreplica{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   labels[util.LABEL_NAME],
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
			}

		}
	}

}

// DeleteClusterBase ...
func DeleteClusterBase(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *crv1.Pgcluster, namespace string) {

	log.Debugf("deleteCluster called with strategy %s", cl.Spec.Strategy)

	pgtask := crv1.Pgtask{}
	found, _ := kubeapi.Getpgtask(client, &pgtask, cl.Spec.Name+"-"+util.LABEL_AUTOFAIL, namespace)
	if found {
		aftask := AutoFailoverTask{}
		aftask.Clear(client, cl.Spec.Name, namespace)
	}

	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok == false {
		log.Error("invalid Strategy requested for cluster creation" + cl.Spec.Strategy)
		return
	}

	strategy.DeleteCluster(clientset, client, cl, namespace)

	upgrade := crv1.Pgupgrade{}
	found, _ = kubeapi.Getpgupgrade(client, &upgrade, cl.Spec.Name, namespace)
	if found {
		err := kubeapi.Deletepgupgrade(client, cl.Spec.Name, namespace)
		if err == nil {
			log.Debug("deleted pgupgrade " + cl.Spec.Name)
		} else if kerrors.IsNotFound(err) {
			log.Debug("will not delete pgupgrade, not found for " + cl.Spec.Name)
		} else {
			log.Error("error deleting pgupgrade " + cl.Spec.Name + err.Error())
		}
	}

}

// AddUpgradeBase ...
func AddUpgradeBase(clientset *kubernetes.Clientset, client *rest.RESTClient, upgrade *crv1.Pgupgrade, namespace string, cl *crv1.Pgcluster) error {
	var err error

	//get the strategy to use
	if cl.Spec.Strategy == "" {
		cl.Spec.Strategy = "1"
		log.Info("using default cluster strategy")
	}

	strategy, ok := strategyMap[cl.Spec.Strategy]
	if ok {
		log.Debug("strategy found")
	} else {
		log.Error("invalid Strategy requested for cluster upgrade" + cl.Spec.Strategy)
		return err
	}

	//invoke the strategy
	if upgrade.Spec.UpgradeType == "minor" {
		err = strategy.MinorUpgrade(clientset, client, cl, upgrade, namespace)
		if err == nil {
			/**
			err = util.Patch(client, "/spec/upgradestatus", crv1.UpgradeCompletedStatus, crv1.PgupgradeResourcePlural, upgrade.Spec.Name, namespace)
			if err != nil {
				log.Error(err)
				log.Error("could not patch the ugpradestatus")
			}
			log.Debug("jeff updated pgupgrade to completed")
			*/
		} else {
			log.Error(err)
			log.Error("error in doing minor upgrade")
		}
	} else {
		log.Error("invalid UPGRADE_TYPE requested for cluster upgrade" + upgrade.Spec.UpgradeType)
		return err
	}
	if err == nil {
		log.Info("updating the pg version after cluster upgrade")
		fullVersion := upgrade.Spec.CCPImageTag
		err = util.Patch(client, "/spec/ccpimagetag", fullVersion, crv1.PgclusterResourcePlural, upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error(err.Error())
		}
	}

	return err

}

// ScaleBase ...
func ScaleBase(clientset *kubernetes.Clientset, client *rest.RESTClient, replica *crv1.Pgreplica, namespace string) {
	var err error

	if replica.Spec.Status == crv1.UpgradeCompletedStatus {
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

	//create the PVC
	pvcName, err := pvc.CreatePVC(clientset, &replica.Spec.ReplicaStorage, replica.Spec.Name, cluster.Spec.Name, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	if cluster.Spec.UserLabels[util.LABEL_ARCHIVE] == "true" {
		//_, err := pvc.CreatePVC(clientset, &cluster.Spec.PrimaryStorage, replica.Spec.Name+"-xlog", cluster.Spec.Name, namespace)
		storage := crv1.PgStorageSpec{}
		pgoStorage := operator.Pgo.Storage[operator.Pgo.XlogStorage]
		storage.StorageClass = pgoStorage.StorageClass
		storage.AccessMode = pgoStorage.AccessMode
		storage.Size = pgoStorage.Size
		storage.StorageType = pgoStorage.StorageType
		storage.MatchLabels = pgoStorage.MatchLabels
		storage.SupplementalGroups = pgoStorage.SupplementalGroups
		storage.Fsgroup = pgoStorage.Fsgroup
		_, err := pvc.CreatePVC(clientset, &storage, replica.Spec.Name+"-xlog", cluster.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}

	/**
	//the -backrestrepo pvc is now an emptydir volume to be backward
	//compatible with the postgres container only, it is not used
	//with the shared backrest repo design
	if cluster.Spec.UserLabels[util.LABEL_BACKREST] == "true" {
		_, err := pvc.CreatePVC(clientset, &cluster.Spec.BackrestStorage, replica.Spec.Name+"-backrestrepo", cluster.Spec.Name, namespace)
		if err != nil {
			log.Error(err)
			return
		}
	}
	*/

	log.Debugf("created replica pvc [%s]", pvcName)

	//update the replica CRD pvcname
	err = util.Patch(client, "/spec/replicastorage/name", pvcName, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
	if err != nil {
		log.Error("error in pvcname patch " + err.Error())
	}

	log.Debugf("creating Pgreplica object strategy is [%s]", cluster.Spec.Strategy)

	if cluster.Spec.Strategy == "" {
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cluster.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for replica creation" + cluster.Spec.Strategy)
		return
	}

	//create the replica service if it doesnt exist

	st := operator.Pgo.Cluster.ServiceType

	if replica.Spec.UserLabels[util.LABEL_SERVICE_TYPE] != "" {
		st = replica.Spec.UserLabels[util.LABEL_SERVICE_TYPE]
	} else if cluster.Spec.UserLabels[util.LABEL_SERVICE_TYPE] != "" {
		st = cluster.Spec.UserLabels[util.LABEL_SERVICE_TYPE]
	}

	serviceName := replica.Spec.ClusterName + "-replica"
	serviceFields := ServiceTemplateFields{
		Name:        serviceName,
		ServiceName: serviceName,
		ClusterName: replica.Spec.ClusterName,
		Port:        cluster.Spec.Port,
		ServiceType: st,
	}

	err = CreateService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return
	}

	//instantiate the replica
	strategy.Scale(clientset, client, replica, namespace, pvcName, &cluster)

	//update the replica CRD status
	err = util.Patch(client, "/spec/status", crv1.UpgradeCompletedStatus, crv1.PgreplicaResourcePlural, replica.Spec.Name, namespace)
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

	log.Debugf("creating Pgreplica object strategy is [%s]", cluster.Spec.Strategy)

	if cluster.Spec.Strategy == "" {
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cluster.Spec.Strategy]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid Strategy requested for replica creation" + cluster.Spec.Strategy)
		return
	}

	strategy.DeleteReplica(clientset, replica, namespace)

}
