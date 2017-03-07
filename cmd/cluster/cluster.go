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
// TODO add a crunchy-proxy deployment to the cluster
package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/crunchydata/operator/tpr"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ServiceTemplateFields struct {
	Name string
	Port string
}

type DeploymentTemplateFields struct {
	Name               string
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
		fmt.Println(err.Error())
		panic(err.Error())
	}
	DeploymentTemplate = template.Must(template.New("deployment template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(REPLICA_DEPLOYMENT_PATH)
	if err != nil {
		fmt.Println(err.Error())
		panic(err.Error())
	}
	ReplicaDeploymentTemplate = template.Must(template.New("replica deployment template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(SERVICE_PATH)
	if err != nil {
		fmt.Println(err.Error())
		panic(err.Error())
	}

	ServiceTemplate = template.Must(template.New("service template").Parse(string(buf)))
}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.CrunchyCluster)

	source := cache.NewListWatchFromClient(client, "crunchyclusters", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		cluster := obj.(*tpr.CrunchyCluster)
		eventchan <- cluster
		addCluster(clientset, client, cluster)
	}
	createDeleteHandler := func(obj interface{}) {
		cluster := obj.(*tpr.CrunchyCluster)
		eventchan <- cluster
		deleteCluster(clientset, client, cluster)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		cluster := obj.(*tpr.CrunchyCluster)
		eventchan <- cluster
		fmt.Println("updating CrunchyCluster object")
		fmt.Println("updated with Name=" + cluster.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.CrunchyCluster{},
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
			fmt.Printf("%#v\n", event)
		}
	}

}

func addCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.CrunchyCluster) {
	fmt.Println("creating CrunchyCluster object")
	fmt.Println("created with Name=" + db.Spec.Name)

	//create the master service
	serviceFields := ServiceTemplateFields{
		Name: db.Spec.Name,
		Port: "5432",
	}

	var doc bytes.Buffer
	err := ServiceTemplate.Execute(&doc, serviceFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	serviceDocString := doc.String()
	fmt.Println(serviceDocString)

	service := v1.Service{}
	err = json.Unmarshal(doc.Bytes(), &service)
	if err != nil {
		fmt.Println("error unmarshalling json into Service ")
		fmt.Println(err.Error())
		return
	}

	//var result api.Service

	svc, err := clientset.Services(v1.NamespaceDefault).Create(&service)
	if err != nil {
		fmt.Println("error creating Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created master service " + svc.Name)

	//create the replica service
	serviceFields = ServiceTemplateFields{
		Name: db.Spec.Name + REPLICA_SUFFIX,
		Port: "5432",
	}

	err = ServiceTemplate.Execute(&doc, serviceFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var doc4 bytes.Buffer
	serviceDocString = doc4.String()
	fmt.Println(serviceDocString)

	service = v1.Service{}
	err = json.Unmarshal(doc4.Bytes(), &service)
	if err != nil {
		fmt.Println("error unmarshalling json into Service ")
		fmt.Println(err.Error())
		return
	}

	svc, err = clientset.Services(v1.NamespaceDefault).Create(&service)
	if err != nil {
		fmt.Println("error creating Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created replica service " + svc.Name)

	//create the master deployment
	//create the deployment - TODO get these fields from the
	//TPR instance
	deploymentFields := DeploymentTemplateFields{
		Name:               db.Spec.Name,
		Port:               "5432",
		CCP_IMAGE_TAG:      "centos7-9.5-1.2.8",
		PVC_NAME:           "crunchy-pvc",
		PG_MASTER_USER:     "master",
		PG_MASTER_PASSWORD: "password",
		PG_USER:            "testuser",
		PG_PASSWORD:        "password",
		PG_DATABASE:        "userdb",
		PG_ROOT_PASSWORD:   "password",
	}

	var doc3 bytes.Buffer
	err = DeploymentTemplate.Execute(&doc3, deploymentFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	deploymentDocString := doc3.String()
	fmt.Println(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(doc.Bytes(), &deployment)
	if err != nil {
		fmt.Println("error unmarshalling master json into Deployment ")
		fmt.Println(err.Error())
		return
	}

	resultDeployment, err := clientset.Deployments(v1.NamespaceDefault).Create(&deployment)
	if err != nil {
		fmt.Println("error creating master Deployment ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created master Deployment " + resultDeployment.Name)

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:               db.Spec.Name,
		Port:               "5432",
		CCP_IMAGE_TAG:      "centos7-9.5-1.2.8",
		PVC_NAME:           "crunchy-pvc",
		PG_MASTER_USER:     "master",
		PG_MASTER_PASSWORD: "password",
		PG_USER:            "testuser",
		PG_PASSWORD:        "password",
		PG_DATABASE:        "userdb",
		PG_ROOT_PASSWORD:   "password",
		PG_MASTER_HOST:     db.Spec.Name,
		REPLICAS:           "2",
	}

	var doc2 bytes.Buffer
	err = DeploymentTemplate.Execute(&doc2, replicaDeploymentFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	replicaDeploymentDocString := doc2.String()
	fmt.Println(replicaDeploymentDocString)

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(doc2.Bytes(), &replicaDeployment)
	if err != nil {
		fmt.Println("error unmarshalling replica json into Deployment ")
		fmt.Println(err.Error())
		return
	}

	resultReplicaDeployment, err2 := clientset.Deployments(v1.NamespaceDefault).Create(&replicaDeployment)
	if err2 != nil {
		fmt.Println("error creating replica Deployment ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created replica Deployment " + resultReplicaDeployment.Name)

}

func deleteCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.CrunchyCluster) {
	fmt.Println("deleting CrunchyCluster object")
	fmt.Println("deleting with Name=" + db.Spec.Name)

	//delete the master service

	err := clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting master Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted master service " + db.Spec.Name)

	//delete the replica service
	err = clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting replica Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted replica service " + db.Spec.Name + REPLICA_SUFFIX)

	//delete the master deployment
	err = clientset.Deployments(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting master Deployment ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted master Deployment " + db.Spec.Name)

	//delete the replica deployment
	err = clientset.Deployments(v1.NamespaceDefault).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting replica Deployment ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted replica Deployment " + db.Spec.Name + REPLICA_SUFFIX)

}
