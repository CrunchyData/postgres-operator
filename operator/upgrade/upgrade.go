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

package upgrade

import (
	log "github.com/Sirupsen/logrus"
	"time"

	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/database"
	"github.com/crunchydata/postgres-operator/operator/util"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const COMPLETED_STATUS = "completed"

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}, namespace string) {

	eventchan := make(chan *tpr.PgUpgrade)

	source := cache.NewListWatchFromClient(client, "pgupgrades", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		job := obj.(*tpr.PgUpgrade)
		eventchan <- job
		addUpgrade(clientset, client, job, namespace)
	}
	createDeleteHandler := func(obj interface{}) {
		job := obj.(*tpr.PgUpgrade)
		eventchan <- job
		deleteUpgrade(clientset, client, job, namespace)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		job := obj.(*tpr.PgUpgrade)
		eventchan <- job
		//log.Info("updating PgUpgrade object")
		//log.Info("updated with Name=" + job.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.PgUpgrade{},
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
			//log.Infof("%#v\n", event)
			if event == nil {
				log.Info("event was null")
			}
		}
	}

}

func addUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, upgrade *tpr.PgUpgrade, namespace string) {
	var err error
	db := tpr.PgDatabase{}
	cl := tpr.PgCluster{}

	//get the pgdatabase TPR

	err = client.Get().
		Resource("pgdatabases").
		Namespace(namespace).
		Name(upgrade.Spec.Name).
		Do().
		Into(&db)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Debug("pgdatabase " + upgrade.Spec.Name + " not found ")
		} else {
			log.Error("error getting pgdatabase " + upgrade.Spec.Name + err.Error())
		}
	} else {
		err = database.AddUpgrade(clientset, client, upgrade, namespace, &db)
		if err != nil {
			log.Error(err.Error())
		} else {
			//update the upgrade TPR status to completed
			err = util.Patch(client, "/spec/upgradestatus", COMPLETED_STATUS, "pgupgrades", upgrade.Spec.Name, namespace)
			if err != nil {
				log.Error(err.Error())
			}
		}
		return
	}

	//not a db so get the pgcluster TPR
	err = client.Get().
		Resource("pgclusters").
		Namespace(namespace).
		Name(upgrade.Spec.Name).
		Do().
		Into(&cl)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Debug("pgcluster " + upgrade.Spec.Name + " not found ")
			return
		} else {
			log.Error("error getting pgcluser " + upgrade.Spec.Name + err.Error())
		}
	}

	err = cluster.AddUpgrade(clientset, client, upgrade, namespace, &cl)
	if err != nil {
		log.Error(err.Error())
	} else {
		//update the upgrade TPR status to completed
		err = util.Patch(client, "/spec/upgradestatus", COMPLETED_STATUS, "pgupgrades", upgrade.Spec.Name, namespace)
		if err != nil {
			log.Error(err.Error())
		}
	}

}

func deleteUpgrade(clientset *kubernetes.Clientset, client *rest.RESTClient, upgrade *tpr.PgUpgrade, namespace string) {
	var jobName = "upgrade-" + upgrade.Spec.Name
	log.Debug("deleting Job with Name=" + jobName + " in namespace " + namespace)

	//delete the job
	//err := clientset.ExtensionsV1beta1Client.Jobs(v1.NamespaceDefault).Delete(jobName,
	err := clientset.Batch().Jobs(namespace).Delete(jobName,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting Job " + jobName + err.Error())
		return
	}
	log.Debug("deleted Job " + jobName)
}
