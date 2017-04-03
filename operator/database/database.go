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

package database

import (
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

type ServiceTemplateFields struct {
	Name string
	Port string
}

type PodTemplateFields struct {
	Name               string
	Port               string
	PVC_NAME           string
	CCP_IMAGE_TAG      string
	PG_MASTER_USER     string
	PG_MASTER_PASSWORD string
	PG_USER            string
	PG_PASSWORD        string
	PG_DATABASE        string
	PG_ROOT_PASSWORD   string
	BACKUP_PVC_NAME    string
	BACKUP_PATH        string
	SECURITY_CONTEXT   string
}

type DatabaseStrategy interface {
	AddDatabase(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgDatabase, string) error
	DeleteDatabase(*kubernetes.Clientset, *rest.RESTClient, *tpr.PgDatabase, string) error
}

var strategyMap map[string]DatabaseStrategy

func init() {
	strategyMap = make(map[string]DatabaseStrategy)
	strategyMap["1"] = DatabaseStrategy1{}

}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgDatabase)

	source := cache.NewListWatchFromClient(client, "pgdatabases", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		db := obj.(*tpr.PgDatabase)
		eventchan <- db
		addDatabase(clientset, client, db, namespace)
	}
	createDeleteHandler := func(obj interface{}) {
		db := obj.(*tpr.PgDatabase)
		eventchan <- db
		deleteDatabase(clientset, client, db, namespace)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		db := obj.(*tpr.PgDatabase)
		eventchan <- db
		//log.Info("updating PgDatabase object")
		//log.Info("updated with Name=" + db.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgDatabase{},
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
				//log.Infof("%#v\n", event)
			}
		}
	}

}

func addDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase, namespace string) {
	var err error
	var strategy DatabaseStrategy

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

	log.Debug("creating PgDatabase object " + db.Spec.STRATEGY + " in namespace " + namespace)

	if db.Spec.STRATEGY == "" {
		db.Spec.STRATEGY = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[db.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")

	} else {
		log.Error("invalid STRATEGY requested for Database creation" + db.Spec.STRATEGY)
		return
	}

	strategy.AddDatabase(clientset, client, db, namespace)

}

func deleteDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgDatabase, namespace string) {
	log.Debug("deleting PgDatabase object with strategy " + db.Spec.STRATEGY + " in namespace " + namespace)

	if db.Spec.STRATEGY == "" {
		db.Spec.STRATEGY = "1"
		log.Info("using default strategy")
	}

	strategy, ok := strategyMap[db.Spec.STRATEGY]
	if ok {
		log.Info("strategy found")
	} else {
		log.Error("invalid STRATEGY requested for database creation" + db.Spec.STRATEGY)
		return
	}

	strategy.DeleteDatabase(clientset, client, db, namespace)

}
