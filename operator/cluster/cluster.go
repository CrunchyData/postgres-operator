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
	"errors"
	log "github.com/Sirupsen/logrus"
	"time"

	"github.com/crunchydata/postgres-operator/operator/pvc"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ClusterStrategy interface {
	AddCluster(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, string) error
	DeleteCluster(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, string) error

	MinorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, *tpr.PgUpgrade, string) error
	MajorUpgrade(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgCluster, *tpr.PgUpgrade, string) error
}

type ServiceTemplateFields struct {
	Name        string
	ClusterName string
	Port        string
}

type DeploymentTemplateFields struct {
	Name               string
	ClusterName        string
	Port               string
	CCP_IMAGE_TAG      string
	PG_MASTER_USER     string
	PG_MASTER_PASSWORD string
	PG_USER            string
	PG_PASSWORD        string
	PG_DATABASE        string
	PG_ROOT_PASSWORD   string
	PVC_NAME           string
	//next 2 are for the replica deployment only
	REPLICAS         string
	PG_MASTER_HOST   string
	SECURITY_CONTEXT string
}

const REPLICA_SUFFIX = "-replica"

var strategyMap map[string]ClusterStrategy

//var strategy1 ClusterStrategy

func init() {
	//strategy1 = ClusterStrategy1{}
	strategyMap = make(map[string]ClusterStrategy)
	strategyMap["1"] = ClusterStrategy1{}
}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgCluster)

	source := cache.NewListWatchFromClient(client, "pgclusters", api.NamespaceAll, fields.Everything())

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

	updateHandler := func(old interface{}, obj interface{}) {
		cluster := obj.(*tpr.PgCluster)
		eventchan <- cluster
		//log.Info("updating PgCluster object")
		//log.Info("updated with Name=" + cluster.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgCluster{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createAddHandler,
			UpdateFunc: updateHandler,
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

func addCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgCluster, namespace string) {
	var err error

	//create the PVC for the master if required
	if db.Spec.PVC_NAME == "" {
		db.Spec.PVC_NAME = db.Spec.Name + "-pvc"
		log.Debug("PVC_NAME=%s PVC_SIZE=%s PVC_ACCESS_MODE=%s\n",
			db.Spec.PVC_NAME, db.Spec.PVC_ACCESS_MODE, db.Spec.PVC_SIZE)
		err = pvc.Create(clientset, db.Spec.PVC_NAME, db.Spec.PVC_ACCESS_MODE, db.Spec.PVC_SIZE, namespace)
		if err != nil {
			log.Error(err.Error())
			return
		}
		log.Info("created PVC =" + db.Spec.PVC_NAME + " in namespace " + namespace)
	}
	log.Debug("creating PgCluster object strategy is [" + db.Spec.STRATEGY + "]")

	if db.Spec.STRATEGY == "" {
		db.Spec.STRATEGY = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[db.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster creation" + db.Spec.STRATEGY)
		return
	}

	strategy.AddCluster(clientset, client, db, namespace)

}

func deleteCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgCluster, namespace string) {

	log.Debug("deleteCluster called with strategy " + db.Spec.STRATEGY)

	if db.Spec.STRATEGY == "" {
		db.Spec.STRATEGY = "1"
	}

	strategy, ok := strategyMap[db.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for cluster creation" + db.Spec.STRATEGY)
		return
	}
	strategy.DeleteCluster(clientset, client, db, namespace)

}

func AddUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, upgrade *tpr.PgUpgrade, namespace string, cluster *tpr.PgCluster) error {
	var err error

	//get the strategy to use
	if cluster.Spec.STRATEGY == "" {
		cluster.Spec.STRATEGY = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[cluster.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		return errors.New("invalid STRATEGY requested for database upgrade" + cluster.Spec.STRATEGY)
	}
	//invoke the strategy
	if upgrade.Spec.UPGRADE_TYPE == "minor" {
		err = strategy.MinorUpgrade(clientset, client, cluster, upgrade, namespace)
	} else if upgrade.Spec.UPGRADE_TYPE == "major" {
		err = strategy.MajorUpgrade(clientset, client, cluster, upgrade, namespace)
	} else {
		return errors.New("invalid UPGRADE_TYPE requested for cluster upgrade " + upgrade.Spec.UPGRADE_TYPE)
	}
	return err

}
