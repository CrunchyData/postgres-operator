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

// Package main is the main function for the crunchy operator
package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/crunchydata/crunchy-operator/tpr"

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
}

const SERVICE_PATH = "/pgconf/database-service.json"
const POD_PATH = "/pgconf/database-pod.json"

var PodTemplate *template.Template
var ServiceTemplate *template.Template

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(POD_PATH)
	if err != nil {
		fmt.Println(err.Error())
		panic(err.Error())
	}
	PodTemplate = template.Must(template.New("pod template").Parse(string(buf)))

	buf, err = ioutil.ReadFile(SERVICE_PATH)
	if err != nil {
		fmt.Println(err.Error())
		panic(err.Error())
	}

	ServiceTemplate = template.Must(template.New("service template").Parse(string(buf)))
}

func Process(clientset *kubernetes.Clientset, client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.CrunchyDatabase)

	source := cache.NewListWatchFromClient(client, "crunchydatabases", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		db := obj.(*tpr.CrunchyDatabase)
		eventchan <- db
		addDatabase(clientset, client, db)
	}
	createDeleteHandler := func(obj interface{}) {
		db := obj.(*tpr.CrunchyDatabase)
		eventchan <- db
		deleteDatabase(clientset, client, db)
	}

	updateHandler := func(old interface{}, obj interface{}) {
		db := obj.(*tpr.CrunchyDatabase)
		eventchan <- db
		fmt.Println("updating CrunchyDatabase object")
		fmt.Println("updated with Name=" + db.Spec.Name)
	}

	_, controller := cache.NewInformer(
		source,
		&tpr.CrunchyDatabase{},
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

// database consists of a Service and a Pod
func addDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.CrunchyDatabase) {
	fmt.Println("creating CrunchyDatabase object")
	fmt.Println("created with Name=" + db.Spec.Name)

	//create the service - TODO get these fields from
	//the TPR instance
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
	fmt.Println("created service " + svc.Name)

	podFields := PodTemplateFields{
		Name:               db.Spec.Name,
		Port:               db.Spec.Port,
		PVC_NAME:           db.Spec.PVC_NAME,
		CCP_IMAGE_TAG:      db.Spec.CCP_IMAGE_TAG,
		PG_MASTER_USER:     db.Spec.PG_MASTER_USER,
		PG_MASTER_PASSWORD: db.Spec.PG_MASTER_PASSWORD,
		PG_USER:            db.Spec.PG_USER,
		PG_PASSWORD:        db.Spec.PG_PASSWORD,
		PG_DATABASE:        db.Spec.PG_DATABASE,
		PG_ROOT_PASSWORD:   db.Spec.PG_ROOT_PASSWORD,
	}

	var doc2 bytes.Buffer
	err = PodTemplate.Execute(&doc2, podFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	podDocString := doc2.String()
	fmt.Println(podDocString)

	pod := v1.Pod{}
	err = json.Unmarshal(doc2.Bytes(), &pod)
	if err != nil {
		fmt.Println("error unmarshalling json into Pod ")
		fmt.Println(err.Error())
		return
	}

	resultPod, err := clientset.Pods(v1.NamespaceDefault).Create(&pod)
	if err != nil {
		fmt.Println("error creating Pod ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("created pod " + resultPod.Name)

}

func deleteDatabase(clientset *kubernetes.Clientset, client *rest.RESTClient, db *tpr.CrunchyDatabase) {
	fmt.Println("deleting CrunchyDatabase object")
	fmt.Println("deleting with Name=" + db.Spec.Name)

	//delete the service
	err := clientset.Services(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting Service ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted service " + db.Spec.Name)

	//delete the pod
	err = clientset.Pods(v1.NamespaceDefault).Delete(db.Spec.Name,
		&v1.DeleteOptions{})
	if err != nil {
		fmt.Println("error deleting Pod ")
		fmt.Println(err.Error())
		return
	}
	fmt.Println("deleted pod " + db.Spec.Name)
	//delete the pod
}
