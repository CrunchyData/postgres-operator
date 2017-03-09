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
			if event == nil {
				fmt.Printf("%#v\n", event)
			}
		}
	}

}

func addCluster(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.CrunchyCluster) {
	var serviceDoc, replicaServiceDoc, masterDoc, replicaDoc bytes.Buffer
	var err error
	var replicaServiceResult, serviceResult *v1.Service
	var replicaDeploymentResult, deploymentResult *v1beta1.Deployment

	fmt.Println("creating CrunchyCluster object")
	fmt.Println("created with Name=" + db.Spec.Name)

	//create the master service
	serviceFields := ServiceTemplateFields{
		Name:        db.Spec.Name,
		ClusterName: db.Spec.Name,
		Port:        "5432",
	}

	err = ServiceTemplate.Execute(&serviceDoc, serviceFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	serviceDocString := serviceDoc.String()
	fmt.Println(serviceDocString)

	service := v1.Service{}
	err = json.Unmarshal(serviceDoc.Bytes(), &service)
	if err != nil {
		fmt.Println("error unmarshalling json into Service ")
		fmt.Println(err.Error())
		return
	}

	serviceResult, err = clientset.Services(v1.NamespaceDefault).Create(&service)
	if err != nil {
		fmt.Println("error creating Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created master service " + serviceResult.Name)

	//create the replica service
	replicaServiceFields := ServiceTemplateFields{
		Name:        db.Spec.Name + REPLICA_SUFFIX,
		ClusterName: db.Spec.Name,
		Port:        "5432",
	}

	err = ServiceTemplate.Execute(&replicaServiceDoc, replicaServiceFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	replicaServiceDocString := replicaServiceDoc.String()
	fmt.Println(replicaServiceDocString)

	replicaService := v1.Service{}
	err = json.Unmarshal(replicaServiceDoc.Bytes(), &replicaService)
	if err != nil {
		fmt.Println("error unmarshalling json into replica Service ")
		fmt.Println(err.Error())
		return
	}

	replicaServiceResult, err = clientset.Services(v1.NamespaceDefault).Create(&replicaService)
	if err != nil {
		fmt.Println("error creating replica Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created replica service " + replicaServiceResult.Name)

	//create the master deployment
	//create the deployment - TODO get these fields from the
	//TPR instance
	deploymentFields := DeploymentTemplateFields{
		Name:               db.Spec.Name,
		ClusterName:        db.Spec.Name,
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

	err = DeploymentTemplate.Execute(&masterDoc, deploymentFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	deploymentDocString := masterDoc.String()
	fmt.Println(deploymentDocString)

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(masterDoc.Bytes(), &deployment)
	if err != nil {
		fmt.Println("error unmarshalling master json into Deployment ")
		fmt.Println(err.Error())
		return
	}

	deploymentResult, err = clientset.Deployments(v1.NamespaceDefault).Create(&deployment)
	if err != nil {
		fmt.Println("error creating master Deployment ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created master Deployment " + deploymentResult.Name)

	//create the replica deployment
	replicaDeploymentFields := DeploymentTemplateFields{
		Name:               db.Spec.Name + REPLICA_SUFFIX,
		ClusterName:        db.Spec.Name,
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

	err = ReplicaDeploymentTemplate.Execute(&replicaDoc, replicaDeploymentFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	replicaDeploymentDocString := replicaDoc.String()
	fmt.Println(replicaDeploymentDocString)

	replicaDeployment := v1beta1.Deployment{}
	err = json.Unmarshal(replicaDoc.Bytes(), &replicaDeployment)
	if err != nil {
		fmt.Println("error unmarshalling replica json into Deployment ")
		fmt.Println(err.Error())
		return
	}

	replicaDeploymentResult, err = clientset.Deployments(v1.NamespaceDefault).Create(&replicaDeployment)
	if err != nil {
		fmt.Println("error creating replica Deployment ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created replica Deployment " + replicaDeploymentResult.Name)

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
	}
	fmt.Println("deleted master service " + db.Spec.Name)

	//delete the replica service
	err = clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting replica Service ")
		fmt.Println(err.Error())
	}
	fmt.Println("deleted replica service " + db.Spec.Name + REPLICA_SUFFIX)

	//delete the master deployment
	err = clientset.Deployments(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting master Deployment ")
		fmt.Println(err.Error())
	}

	fmt.Println("deleted master Deployment " + db.Spec.Name)
	//delete the master replicaset

	//find the replicaset pod name
	options := v1.ListOptions{}
	options.LabelSelector = "name=" + db.Spec.Name

	var reps *v1beta1.ReplicaSetList
	reps, err = clientset.ReplicaSets(v1.NamespaceDefault).List(options)
	if err != nil {
		fmt.Println("error getting master replicaset name")
		fmt.Println(err.Error())
	} else {
		if len(reps.Items) > 0 {
			err = clientset.ReplicaSets(v1.NamespaceDefault).Delete(reps.Items[0].Name,
				&v1.DeleteOptions{})
			if err != nil {
				fmt.Println("error deleting master replicaset ")
				fmt.Println(err.Error())
			}

			fmt.Println("deleted master replicaset " + reps.Items[0].Name)
		}
	}

	//delete the replica deployment
	err = clientset.Deployments(v1.NamespaceDefault).Delete(db.Spec.Name+REPLICA_SUFFIX,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting replica Deployment ")
		fmt.Println(err.Error())
	}
	fmt.Println("deleted replica Deployment " + db.Spec.Name + REPLICA_SUFFIX)
	//delete the replica ReplicaSet
	options.LabelSelector = "name=" + db.Spec.Name + REPLICA_SUFFIX

	reps, err = clientset.ReplicaSets(v1.NamespaceDefault).List(options)
	if err != nil {
		fmt.Println("error getting replica replicaset name")
		fmt.Println(err.Error())
	} else {
		if len(reps.Items) > 0 {
			err = clientset.ReplicaSets(v1.NamespaceDefault).Delete(reps.Items[0].Name,
				&v1.DeleteOptions{})
			if err != nil {
				fmt.Println("error deleting replica replicaset ")
				fmt.Println(err.Error())
			}
			fmt.Println("deleted replica replicaset " + reps.Items[0].Name)
		}
	}

	//lastly, delete any remaining pods
	listOptions := v1.ListOptions{}
	listOptions.LabelSelector = "name=" + db.Spec.Name
	pods, err := clientset.Core().Pods(v1.NamespaceDefault).List(listOptions)
	for _, pod := range pods.Items {
		fmt.Println("deleting pod " + pod.Name)
		err = clientset.Pods(v1.NamespaceDefault).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			fmt.Println("error deleting pod " + pod.Name)
			fmt.Println(err.Error())
		}
		fmt.Println("deleted pod " + pod.Name)

	}
	listOptions.LabelSelector = "name=" + db.Spec.Name + REPLICA_SUFFIX
	pods, err = clientset.Core().Pods(v1.NamespaceDefault).List(listOptions)
	for _, pod := range pods.Items {
		fmt.Println("deleting pod " + pod.Name)
		err = clientset.Pods(v1.NamespaceDefault).Delete(pod.Name,
			&v1.DeleteOptions{})
		if err != nil {
			fmt.Println("error deleting pod " + pod.Name)
			fmt.Println(err.Error())
		}
		fmt.Println("deleted pod " + pod.Name)

	}

}
