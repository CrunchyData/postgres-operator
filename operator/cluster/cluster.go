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

// Package cluster holds the cluster TPR logic and definitions
// A cluster is comprised of a master service, replica service,
// master deployment, and replica deployment
package cluster

import (
	log "github.com/Sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"

	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ClusterStrategy interface {
	AddCluster(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, string, string) error
	CreateReplica(string, *kubernetes.Clientset, *tpr.PgCluster, string, string, string, bool) error
	DeleteCluster(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, string) error

	MinorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, *tpr.PgUpgrade, string) error
	MajorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, *tpr.PgUpgrade, string) error
	MajorUpgradeFinalize(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, *tpr.PgUpgrade, string) error
	PrepareClone(*kubernetes.Clientset, *rest.RESTClient, string, *tpr.PgCluster, string) error
	UpdatePolicyLabels(*kubernetes.Clientset, string, string, map[string]string) error
}

type ServiceTemplateFields struct {
	Name        string
	ClusterName string
	Port        string
}

type DeploymentTemplateFields struct {
	Name                 string
	ClusterName          string
	Port                 string
	CCP_IMAGE_TAG        string
	PG_DATABASE          string
	OPERATOR_LABELS      string
	PGDATA_PATH_OVERRIDE string
	PVC_NAME             string
	BACKUP_PVC_NAME      string
	BACKUP_PATH          string
	PGROOT_SECRET_NAME   string
	PGUSER_SECRET_NAME   string
	PGMASTER_SECRET_NAME string
	SECURITY_CONTEXT     string
	//next 2 are for the replica deployment only
	REPLICAS       string
	PG_MASTER_HOST string
}

const REPLICA_SUFFIX = "-replica"

var StrategyMap map[string]ClusterStrategy

func init() {
	StrategyMap = make(map[string]ClusterStrategy)
	StrategyMap["1"] = ClusterStrategy1{}
}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgCluster)

	source := cache.NewListWatchFromClient(client, tpr.CLUSTER_RESOURCE, namespace, fields.Everything())

	createAddHandler := func(obj interface{}) {
		cluster := obj.(*tpr.PgCluster)
		eventchan <- cluster
		addCluster(clientset, client, cluster, namespace)
	}
	createDeleteHandler := func(obj interface{}) {
		cluster := obj.(*tpr.PgCluster)
		eventchan <- cluster
		deleteCluster(clientset, client, cluster, namespace)
	}

	createUpdateHandler := func(old interface{}, obj interface{}) {
		oldcluster := old.(*tpr.PgCluster)
		cluster := obj.(*tpr.PgCluster)
		eventchan <- cluster
		updateCluster(clientset, client, cluster, oldcluster, namespace)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgCluster{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createAddHandler,
			UpdateFunc: createUpdateHandler,
			DeleteFunc: createDeleteHandler,
		})

	go controller.Run(stopchan)

	for {
		select {
		case event := <-eventchan:
			if event == nil {
				log.Infof("%#v\n", event)
			}
		}
	}

}

func addCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string) {
	var err error

	if cl.Spec.STATUS == tpr.UPGRADE_COMPLETED_STATUS {
		log.Warn("tpr pgcluster " + cl.Spec.ClusterName + " is already marked complete, will not recreate")
		return
	}

	//create the PVC for the master if required
	//var pvcName string
	/**

	switch cl.Spec.MasterStorage.StorageType {
	case "":
		log.Debug("MasterStorage.StorageType is empty")
	case "emptydir":
		log.Debug("MasterStorage.StorageType is emptydir")
	case "existing":
		log.Debug("MasterStorage.StorageType is existing")
		pvcName = cl.Spec.MasterStorage.PvcName
	case "create":
		log.Debug("MasterStorage.StorageType is create")
		pvcName = cl.Spec.Name + "-pvc"
		log.Debug("PVC_NAME=%s PVC_SIZE=%s PVC_ACCESS_MODE=%s\n",
			pvcName, cl.Spec.MasterStorage.PvcAccessMode, cl.Spec.MasterStorage.PvcSize)
		err = pvc.Create(clientset, pvcName, cl.Spec.MasterStorage.PvcAccessMode, cl.Spec.MasterStorage.PvcSize, cl.Spec.MasterStorage.StorageType, cl.Spec.MasterStorage.StorageClass, namespace)
		if err != nil {
			log.Error("error in pvc create " + err.Error())
			return
		}
		log.Info("created PVC =" + pvcName + " in namespace " + namespace)
	case "dynamic":
		log.Debug("MasterStorage.StorageType is dynamic, not supported yet")
	}
	*/

	pvcName, err := createPVC(clientset, cl.Spec.Name, &cl.Spec.MasterStorage, namespace)
	log.Debug("created master pvc [" + pvcName + "]")

	log.Debug("creating PgCluster object strategy is [" + cl.Spec.STRATEGY + "]")

	var err1, err2, err3 error
	if cl.Spec.SECRET_FROM != "" {
		cl.Spec.PG_ROOT_PASSWORD, err1 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SECRET_FROM+tpr.PGROOT_SECRET_SUFFIX)
		cl.Spec.PG_PASSWORD, err2 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SECRET_FROM+tpr.PGUSER_SECRET_SUFFIX)
		cl.Spec.PG_MASTER_PASSWORD, err3 = util.GetPasswordFromSecret(clientset, namespace, cl.Spec.SECRET_FROM+tpr.PGMASTER_SECRET_SUFFIX)
		if err1 != nil || err2 != nil || err3 != nil {
			log.Error("error getting secrets using SECRET_FROM " + cl.Spec.SECRET_FROM)
			return
		}
	}

	err = util.CreateDatabaseSecrets(clientset, client, cl, namespace)
	if err != nil {
		log.Error("error in create secrets " + err.Error())
		return
	}

	if cl.Spec.STRATEGY == "" {
		cl.Spec.STRATEGY = "1"
		log.Info("using default strategy")
	}

	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster creation" + cl.Spec.STRATEGY)
		return
	}

	setFullVersion(client, cl, namespace)

	strategy.AddCluster(clientset, client, cl, namespace, pvcName)

	err = util.Patch(client, "/spec/status", tpr.UPGRADE_COMPLETED_STATUS, tpr.CLUSTER_RESOURCE, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in status patch " + err.Error())
	}

}

func setFullVersion(tprclient *rest.RESTClient, cl *tpr.PgCluster, namespace string) {
	//get full version from image tag
	fullVersion := util.GetFullVersion(cl.Spec.CCP_IMAGE_TAG)

	//update the tpr
	err := util.Patch(tprclient, "/spec/postgresfullversion", fullVersion, tpr.CLUSTER_RESOURCE, cl.Spec.Name, namespace)
	if err != nil {
		log.Error("error in version patch " + err.Error())
	}

}

func deleteCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, namespace string) {

	log.Debug("deleteCluster called with strategy " + cl.Spec.STRATEGY)

	if cl.Spec.STRATEGY == "" {
		cl.Spec.STRATEGY = "1"
	}

	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster creation" + cl.Spec.STRATEGY)
		return
	}
	util.DeleteDatabaseSecrets(clientset, cl.Spec.Name, namespace)
	strategy.DeleteCluster(clientset, client, cl, namespace)
	err := client.Delete().
		Resource(tpr.UPGRADE_RESOURCE).
		Namespace(namespace).
		Name(cl.Spec.Name).
		Do().
		Error()
	if err == nil {
		log.Info("deleted pgupgrade " + cl.Spec.Name)
	} else if kerrors.IsNotFound(err) {
		log.Info("will not delete pgupgrade, not found for " + cl.Spec.Name)
	} else {
		log.Error("error deleting pgupgrade " + cl.Spec.Name + err.Error())
	}

}

func AddUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, upgrade *tpr.PgUpgrade, namespace string, cl *tpr.PgCluster) error {
	var err error

	//get the strategy to use
	if cl.Spec.STRATEGY == "" {
		cl.Spec.STRATEGY = "1"
		log.Info("using default cluster strategy")
	}

	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster upgrade" + cl.Spec.STRATEGY)
		return err
	}

	//invoke the strategy
	if upgrade.Spec.UPGRADE_TYPE == "minor" {
		err = strategy.MinorUpgrade(clientset, client, cl, upgrade, namespace)
		if err == nil {
			err = util.Patch(client, "/spec/upgradestatus", tpr.UPGRADE_COMPLETED_STATUS, tpr.UPGRADE_RESOURCE, upgrade.Spec.Name, namespace)
		}
	} else if upgrade.Spec.UPGRADE_TYPE == "major" {
		err = strategy.MajorUpgrade(clientset, client, cl, upgrade, namespace)
	} else {
		log.Error("invalid UPGRADE_TYPE requested for cluster upgrade" + upgrade.Spec.UPGRADE_TYPE)
		return err
	}
	if err == nil {
		log.Info("updating the pg version after cluster upgrade")
		fullVersion := util.GetFullVersion(upgrade.Spec.CCP_IMAGE_TAG)
		err = util.Patch(client, "/spec/postgresfullversion", fullVersion, tpr.CLUSTER_RESOURCE, upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error(err.Error())
		}
	}

	return err

}

func updateCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, oldcluster *tpr.PgCluster, namespace string) {

	if oldcluster.Spec.REPLICAS != cl.Spec.REPLICAS {
		log.Debug("detected change to REPLICAS for " + cl.Spec.Name + " from " + oldcluster.Spec.REPLICAS + " to " + cl.Spec.REPLICAS)
		ScaleReplicas(clientset, client, cl, oldcluster, namespace)
	}

}

func ScaleReplicas(clientset *kubernetes.Clientset, client *rest.RESTClient, cl *tpr.PgCluster, oldcluster *tpr.PgCluster, namespace string) {

	oldCount, err := strconv.Atoi(oldcluster.Spec.REPLICAS)
	if err != nil {
		log.Error(err)
	}
	newCount, err := strconv.Atoi(cl.Spec.REPLICAS)
	if err != nil {
		log.Error(err)
	}

	//get the strategy to use
	if cl.Spec.STRATEGY == "" {
		cl.Spec.STRATEGY = "1"
		log.Info("using default cluster strategy")
	}

	strategy, ok := StrategyMap[cl.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster upgrade" + cl.Spec.STRATEGY)
		return
	}

	if oldCount > newCount {
		log.Debug("scale down not implemented yet")
	} else {
		//scale up
		log.Debug("scale up called ")
		newReplicas := newCount - oldCount

		for i := 0; i < newReplicas; i++ {
			//generate a unique name suffix
			uniqueName := RandStringBytesRmndr(4)
			depName := cl.Spec.Name + "-replica-" + uniqueName

			//create a PVC
			pvcName, err := createPVC(clientset, depName, &cl.Spec.ReplicaStorage, namespace)
			if err != nil {
				log.Error(err)
				return
			}
			//create a Deployment and its service
			serviceName := depName + "-replica"
			replicaServiceFields := ServiceTemplateFields{
				Name:        serviceName,
				ClusterName: cl.Spec.Name,
				Port:        cl.Spec.Port,
			}

			err = CreateService(clientset, &replicaServiceFields, namespace)
			if err != nil {
				log.Error(err)
				return
			}
			strategy.CreateReplica(serviceName, clientset, cl, depName, pvcName, namespace, false)
		}
	}
}

func createPVC(clientset *kubernetes.Clientset, name string, storageSpec *tpr.PgStorageSpec, namespace string) (string, error) {
	var pvcName string
	var err error

	switch storageSpec.StorageType {
	case "":
		log.Debug("StorageType is empty")
	case "emptydir":
		log.Debug("StorageType is emptydir")
	case "existing":
		log.Debug("StorageType is existing")
		pvcName = storageSpec.PvcName
	case "create":
		log.Debug("StorageType is create")
		pvcName = name + "-pvc"
		log.Debug("PVC_NAME=%s PVC_SIZE=%s PVC_ACCESS_MODE=%s\n",
			pvcName, storageSpec.PvcAccessMode, storageSpec.PvcSize)
		err = pvc.Create(clientset, pvcName, storageSpec.PvcAccessMode, storageSpec.PvcSize, storageSpec.StorageType, storageSpec.StorageClass, namespace)
		if err != nil {
			log.Error("error in pvc create " + err.Error())
			return pvcName, err
		}
		log.Info("created PVC =" + pvcName + " in namespace " + namespace)
	case "dynamic":
		log.Debug("StorageType is dynamic, not supported yet")
	}

	return pvcName, err
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

}
func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
