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

	"github.com/crunchydata/operator/tpr"

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

func Process(client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.CrunchyDatabase)

	source := cache.NewListWatchFromClient(client, "crunchydatabases", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		db := obj.(*tpr.CrunchyDatabase)
		eventchan <- db
		addDatabase(db)
	}
	createDeleteHandler := func(obj interface{}) {
		db := obj.(*tpr.CrunchyDatabase)
		eventchan <- db
		deleteDatabase(db)
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

func addDatabase(db *tpr.CrunchyDatabase) {
	fmt.Println("creating CrunchyDatabase object")
	fmt.Println("created with Name=" + db.Spec.Name)

	//create the service
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

	//create the pod
	podFields := PodTemplateFields{
		Name:               db.Spec.Name,
		Port:               "5432",
		CCP_IMAGE_TAG:      "centos7-9.5-1.2.8",
		PG_MASTER_USER:     "master",
		PG_MASTER_PASSWORD: "password",
		PG_USER:            "testuser",
		PG_PASSWORD:        "password",
		PG_DATABASE:        "userdb",
		PG_ROOT_PASSWORD:   "password",
	}

	var doc2 bytes.Buffer
	err = PodTemplate.Execute(&doc2, podFields)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	podDocString := doc2.String()
	fmt.Println(podDocString)

	pod := api.Pod{}
	err = json.Unmarshal(doc2.Bytes(), &pod)
	if err != nil {
		fmt.Println("error unmarshalling json into Pod ")
		fmt.Println(err.Error())
		return
	}

}

func deleteDatabase(db *tpr.CrunchyDatabase) {
	fmt.Println("deleting CrunchyDatabase object")
	fmt.Println("deleting with Name=" + db.Spec.Name)

	//delete the service
	//delete the pod
}
