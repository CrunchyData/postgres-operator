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
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/crunchydata/postgres-operator/operator/backup"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/database"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	//_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	config *rest.Config
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "the path to a kubeconfig, specifies this tool runs outside the cluster")
	flag.Parse()

	tprclient, err := buildClientFromFlags(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("error creating cluster config ")
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("error creating kube client ")
		panic(err.Error())
	}

	exampleList := tpr.CrunchyDatabaseList{}
	err = tprclient.Get().
		Resource("crunchydatabases").
		Do().Into(&exampleList)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%#v\n", exampleList)

	example := tpr.CrunchyDatabase{}
	err = tprclient.Get().
		Namespace("default").
		Resource("crunchydatabases").
		Name("example1").
		Do().Into(&example)
	if err != nil {
		fmt.Println("example1 not found")
	} else {
		fmt.Printf("%#v\n", example)
	}

	fmt.Println()
	fmt.Println("---------------------------------------------------------")
	fmt.Println()

	stopchan := make(chan struct{}, 1)

	go database.Process(clientset, tprclient, stopchan)
	go cluster.Process(clientset, tprclient, stopchan)
	go backup.Process(clientset, tprclient, stopchan)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-signals:
			fmt.Printf("received signal %#v, exiting...\n", s)
			os.Exit(0)
		}
	}
}

func buildClientFromFlags(kubeconfig string) (*rest.RESTClient, error) {
	config, err := configFromFlags(kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &unversioned.GroupVersion{
		Group:   "crunchydata.com",
		Version: "v1",
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	schemeBuilder.AddToScheme(api.Scheme)

	return rest.RESTClientFor(config)
}

func configFromFlags(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		unversioned.GroupVersion{Group: "crunchydata.com", Version: "v1"},
		&tpr.CrunchyDatabase{},
		&tpr.CrunchyDatabaseList{},
		&tpr.CrunchyCluster{},
		&tpr.CrunchyClusterList{},
		&tpr.CrunchyBackup{},
		&tpr.CrunchyBackupList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)

	return nil
}
