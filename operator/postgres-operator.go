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
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crunchydata/postgres-operator/operator/backup"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/database"
	"github.com/crunchydata/postgres-operator/operator/upgrade"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
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
	var debug = flag.Bool("debug", false, "defaults to false")
	flag.Parse()

	var debugEnv = os.Getenv("DEBUG")
	var namespace = os.Getenv("NAMESPACE")

	if namespace == "" {
		namespace = "default"
	}

	if *debug || debugEnv != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	tprclient, err := buildClientFromFlags(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Info("error creating cluster config ")
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating kube client ")
		panic(err.Error())
	}

	initializeResources(clientset)

	//wait a bit to let the resources be created
	time.Sleep(2000 * time.Millisecond)

	/**
	exampleList := tpr.CrunchyDatabaseList{}
	err = tprclient.Get().
		Resource("pgdatabases").
		Do().Into(&exampleList)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%#v\n", exampleList)

	example := tpr.CrunchyDatabase{}
	err = tprclient.Get().
		Namespace("default").
		Resource("pgdatabases").
		Name("example1").
		Do().Into(&example)
	if err != nil {
		log.Info("example1 not found")
	} else {
		fmt.Printf("%#v\n", example)
	}
	*/

	log.Info("---------------------------------------------------------")

	stopchan := make(chan struct{}, 1)

	go database.Process(clientset, tprclient, stopchan, namespace)
	go cluster.Process(clientset, tprclient, stopchan, namespace)
	go backup.Process(clientset, tprclient, stopchan, namespace)
	go upgrade.Process(clientset, tprclient, stopchan, namespace)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-signals:
			log.Infof("received signal %#v, exiting...\n", s)
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
		&tpr.PgDatabase{},
		&tpr.PgDatabaseList{},
		&tpr.PgCluster{},
		&tpr.PgClusterList{},
		&tpr.PgBackup{},
		&tpr.PgBackupList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)

	return nil
}

func initializeResources(clientset *kubernetes.Clientset) {
	// initialize third party resources if they do not exist

	tpr, err := clientset.Extensions().ThirdPartyResources().Get("pg-database.crunchydata.com")
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: v1.ObjectMeta{
					Name: "pg-database.crunchydata.com",
				},
				Versions: []v1beta1.APIVersion{
					{Name: "v1"},
				},
				Description: "A postgres database ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get("pg-cluster.crunchydata.com")
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: v1.ObjectMeta{
					Name: "pg-cluster.crunchydata.com",
				},
				Versions: []v1beta1.APIVersion{
					{Name: "v1"},
				},
				Description: "A postgres cluster ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get("pg-backup.crunchydata.com")
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: v1.ObjectMeta{
					Name: "pg-backup.crunchydata.com",
				},
				Versions: []v1beta1.APIVersion{
					{Name: "v1"},
				},
				Description: "A postgres backup ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get("pg-upgrade.crunchydata.com")
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: v1.ObjectMeta{
					Name: "pg-upgrade.crunchydata.com",
				},
				Versions: []v1beta1.APIVersion{
					{Name: "v1"},
				},
				Description: "A postgres upgrade ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

}
