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
	"bytes"
	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

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
	REPLICAS       string
	PG_MASTER_HOST string
}

const SERVICE_PATH = "/pgconf/cluster-service.json"
const DEPLOYMENT_PATH = "/pgconf/cluster-deployment.json"
const REPLICA_DEPLOYMENT_PATH = "/pgconf/cluster-replica-deployment.json"

var DeploymentTemplate *template.Template
var ReplicaDeploymentTemplate *template.Template
var ServiceTemplate *template.Template

const REPLICA_SUFFIX = "-replica"

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(DEPLOYMENT_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	DeploymentTemplate = template.Must(template.New("deployment template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(REPLICA_DEPLOYMENT_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	ReplicaDeploymentTemplate = template.Must(template.New("replica deployment template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(SERVICE_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}

	ServiceTemplate = template.Must(template.New("service template").Parse(string(buf)))
}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.PgCluster)

	source := cache.NewListWatchFromClient(client, "pgclusters", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		cluster := obj.(*tpr.PgCluster)
		eventchan <- cluster
		addCluster(clientset, client, cluster)
	}
	createDeleteHandler := func(obj interface{}) {
		cluster := obj.(*tpr.PgCluster)
		eventchan <- cluster
		deleteCluster(clientset, client, cluster)
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

func addCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgCluster) {
	var serviceDoc, replicaServiceDoc, masterDoc, replicaDoc bytes.Buffer
	var err error
	var replicaServiceResult, serviceResult *v1.Service
	var replicaDeploymentResult, deploymentResult *v1beta1.Deployment

	log.Info("creating PgCluster object")
	log.Info("created with Name=" + db.Spec.Name)

	//create the master service
	serviceFields := ServiceTemplateFields{
		Name:        db.Spec.Name,
		ClusterName: db.Spec.Name,
		Port:        db.Spec.Port,
	}

	err = ServiceTemplate.Execute(&serviceDoc, serviceFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	serviceDocString := serviceDoc.String()
	log.Info(serviceDocString)

	service := v1.Service{}
	err = json.Unmarshal(serviceDoc.Bytes(), &service)
	if err != nil {
		log.Error("error unmarshalling json into Service " + err.Error())
		return
	}

	serviceResult, err = clientset.Services(v1.NamespaceDefault).Create(&service)
	if err != nil {
		log.Error("error creating Service " + err.Error())
		return
	}
	log.Info("created master service " + serviceResult.Name)

	//create the replica service
	replicaServiceFields := ServiceTemplateFields{
		Name:        db.Spec.Name + REPLICA_SUFFIX,
		ClusterName: db.Spec.Name,
		Port:        db.Spec.Port,
	}

	err = ServiceTemplate.Execute(&replicaServiceDoc, replicaServiceFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	replicaServiceDocString := replicaServiceDoc.String()
	log.Info(replicaServiceDocString)

	replicaService := v1.Service{}
	err = json.Unmarshal(replicaServiceDoc.Bytes(), &replicaService)
	if err != nil {
		log.Error("error unmarshalling json into replica Service " + err.Error())
		return
	}

	replicaServiceResult, err = clientset.Services(v1.NamespaceDefault).Create(&replicaService)
	if err != nil {
		log.Error("error creating replica Service " + err.Error())
		return
	}
	log.Info("created replica service " + replicaServiceResult.Name)

	//create the master deployment
	deploymentFields := DeploymentTemplateFields{
		Name:               db.Spec.Name,
		ClusterName:        db.Spec.Name,
		Port:               db.Spec.Port,
		CCP_IMAGE_TAG:      db.Spec.CCP_IMAGE_TAG,
		PVC_NAME:           db.Spec.PVC_NAME,
		PG_MASTER_USER:     db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD: db.Spec.PG_MASTER_PASSWORD,
		PG_USER:            db.Spec.PG_USER,
		PG_PASSWORD:        db.Spec.PG_PASSWORD,
		PG_DATABASE:        db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:   db.Spec.PG_ROOT_PASSWORD,
	}

	err = DeploymentTemplate.Execute(&masterDoc, deploymentFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	deploymentDocString := masterDoc.String()
	log.Info(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(masterDoc.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling master json into Deployment " + err.Error())
		return
	}

	deploymentResult, err = clientset.Deployments(v1.NamespaceDefault).Create(&deployment)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		return
	}
	log.Info("created master Deployment " + deploymentResult.Name)

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:               db.Spec.Name + REPLICA_SUFFIX,
		ClusterName:        db.Spec.Name,
		Port:               db.Spec.Port,
		CCP_IMAGE_TAG:      db.Spec.CCP_IMAGE_TAG,
		PVC_NAME:           db.Spec.PVC_NAME,
		PG_MASTER_HOST:     db.Spec.PG_MASTER_HOST,
		PG_MASTER_USER:     db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD: db.Spec.PG_MASTER_PASSWORD,
		PG_USER:            db.Spec.PG_USER,
		PG_PASSWORD:        db.Spec.PG_PASSWORD,
		PG_DATABASE:        db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:   db.Spec.PG_ROOT_PASSWORD,
		REPLICAS:           db.Spec.REPLICAS,
	}

	err = ReplicaDeploymentTemplate.Execute(&replicaDoc, replicaDeploymentFields)
	if err != nil {
		log.Error(err.Error())
		return
	}
	replicaDeploymentDocString := replicaDoc.String()
	log.Info(replicaDeploymentDocString)

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		log.Error("error unmarshalling replica json into Deployment " + err.Error())
		return
	}

	replicaDeploymentResult, err = clientset.Deployments(v1.NamespaceDefault).Create(&replicaDeployment)
	if err != nil {
		log.Error("error creating replica Deployment " + err.Error())
		return
	}
	log.Info("created replica Deployment " + replicaDeploymentResult.Name)

}

func deleteCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.PgCluster) {
	log.Info("deleting PgCluster object")
	log.Info("deleting with Name=" + db.Spec.Name)

	//delete the master service

	err := clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Service " + err.Error())
	}
	log.Info("deleted master service " + db.Spec.Name)

	//delete the replica service
	err = clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Service " + err.Error())
	}
	log.Info("deleted replica service " + db.Spec.Name + REPLICA_SUFFIX)

	//delete the master deployment
	err = clientset.Deployments(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting master Deployment " + err.Error())
	}

	log.Info("deleted master Deployment " + db.Spec.Name)
	//delete the master replicaset

	//find the replicaset pod name
	options := v1.ListOptions{}
	options.LabelSelector = "name=" + db.Spec.Name

	var reps *v1beta1.ReplicaSetList
	reps, err = clientset.ReplicaSets(v1.NamespaceDefault).List(options)
	if err != nil {
		log.Error("error getting master replicaset name" + err.Error())
	} else {
		if len(reps.Items) > 0 {
			err = clientset.ReplicaSets(v1.NamespaceDefault).Delete(reps.Items[0].Name,
				&v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting master replicaset " + err.Error())
			}

			log.Info("deleted master replicaset " + reps.Items[0].Name)
		}
	}

	//delete the replica deployment
	err = clientset.Deployments(v1.NamespaceDefault).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		log.Error("error deleting replica Deployment " + err.Error())
	}
	log.Info("deleted replica Deployment " + db.Spec.Name + REPLICA_SUFFIX)
	//delete the replica ReplicaSet
	options.LabelSelector = "name=" + db.Spec.Name + REPLICA_SUFFIX

	reps, err = clientset.ReplicaSets(v1.NamespaceDefault).List(options)
	if err != nil {
		log.Error("error getting replica replicaset name" + err.Error())
	} else {
		if len(reps.Items) > 0 {
			err = clientset.ReplicaSets(v1.NamespaceDefault).Delete(reps.Items[0].Name,
				&v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting replica replicaset " + err.Error())
			}
			log.Info("deleted replica replicaset " + reps.Items[0].Name)
		}
	}

	//lastly, delete any remaining pods
	listOptions := v1.ListOptions{}
	listOptions.LabelSelector = "name=" + db.Spec.Name
	pods, err := clientset.Core().Pods(v1.NamespaceDefault).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name)
		err = clientset.Pods(v1.NamespaceDefault).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name)

	}
	listOptions.LabelSelector = "name=" + db.Spec.Name + REPLICA_SUFFIX
	pods, err = clientset.Core().Pods(v1.NamespaceDefault).List(listOptions)
	for _, pod := range pods.Items {
		log.Info("deleting pod " + pod.Name)
		err = clientset.Pods(v1.NamespaceDefault).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			log.Error("error deleting pod " + pod.Name + err.Error())
		}
		log.Info("deleted pod " + pod.Name)

	}

}
