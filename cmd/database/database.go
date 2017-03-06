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
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/crunchydata/operator/tpr"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ServiceTemplate struct {
	Name string
	Port string
}

type PodTemplate struct {
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
	service := ServiceTemplate{
		Name: db.Spec.Name,
		Port: "5432",
	}

	t, err := template.New("service template").ParseFiles("/pgconf/database-service.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	t.Execute(os.Stdout, service)

	//create the pod
	pod := PodTemplate{
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

	t, err = template.New("pod template").ParseFiles("/pgconf/database-pod.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	t.Execute(os.Stdout, pod)
}

func deleteDatabase(db *tpr.CrunchyDatabase) {
	fmt.Println("deleting CrunchyDatabase object")
	fmt.Println("deleting with Name=" + db.Spec.Name)

	//delete the service
	//delete the pod
}
