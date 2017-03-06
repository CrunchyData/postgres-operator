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
package cluster

import (
	"fmt"
	"time"

	"github.com/crunchydata/operator/tpr"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func Process(client *rest.RESTClient, stopchan chan struct{}) {

	eventchan := make(chan *tpr.CrunchyCluster)

	source := cache.NewListWatchFromClient(client, "crunchyclusters", api.NamespaceAll, fields.Everything())

	createAddHandler := func(obj interface{}) {
		cluster := obj.(*tpr.CrunchyCluster)
		eventchan <- cluster
		addCluster(cluster)
	}
	createDeleteHandler := func(obj interface{}) {
		cluster := obj.(*tpr.CrunchyCluster)
		eventchan <- cluster
		deleteCluster(cluster)
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

func addCluster(db *tpr.CrunchyCluster) {
	fmt.Println("creating CrunchyCluster object")
	fmt.Println("created with Name=" + db.Spec.Name)
}

func deleteCluster(db *tpr.CrunchyCluster) {
	fmt.Println("deleting CrunchyCluster object")
	fmt.Println("deleting with Name=" + db.Spec.Name)
}
