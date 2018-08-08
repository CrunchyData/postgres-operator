package main

/*
Copyright 2017-2018 The Kubernetes Authors.

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

import (
	"context"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	crdclient "github.com/crunchydata/postgres-operator/client"
	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"k8s.io/client-go/kubernetes"
)

var Clientset *kubernetes.Clientset
var Namespace string

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	Namespace = os.Getenv("NAMESPACE")
	log.Debug("setting NAMESPACE to " + Namespace)
	if Namespace == "" {
		log.Error("NAMESPACE env var not set")
		os.Exit(2)
	}

	operator.Initialize()

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating Clientset")
		panic(err.Error())
	}

	// make a new config for our extension's API group, using the first config as a baseline
	crdClient, crdScheme, err := crdclient.NewClient(config)
	if err != nil {
		panic(err)
	}

	// start a controller on instances of our custom resource

	pgTaskcontroller := controller.PgtaskController{
		PgtaskConfig:    config,
		PgtaskClient:    crdClient,
		PgtaskScheme:    crdScheme,
		PgtaskClientset: Clientset,
		Namespace:       Namespace,
	}

	pgIngestcontroller := controller.PgingestController{
		PgingestClient:    crdClient,
		PgingestScheme:    crdScheme,
		PgingestClientset: Clientset,
		Namespace:         Namespace,
	}

	pgClustercontroller := controller.PgclusterController{
		PgclusterClient:    crdClient,
		PgclusterScheme:    crdScheme,
		PgclusterClientset: Clientset,
		Namespace:          Namespace,
	}
	pgReplicacontroller := controller.PgreplicaController{
		PgreplicaClient:    crdClient,
		PgreplicaScheme:    crdScheme,
		PgreplicaClientset: Clientset,
		Namespace:          Namespace,
	}
	pgUpgradecontroller := controller.PgupgradeController{
		PgupgradeClientset: Clientset,
		PgupgradeClient:    crdClient,
		PgupgradeScheme:    crdScheme,
		Namespace:          Namespace,
	}
	pgBackupcontroller := controller.PgbackupController{
		PgbackupClient:    crdClient,
		PgbackupScheme:    crdScheme,
		PgbackupClientset: Clientset,
		Namespace:         Namespace,
	}
	pgPolicycontroller := controller.PgpolicyController{
		PgpolicyClient:    crdClient,
		PgpolicyScheme:    crdScheme,
		PgpolicyClientset: Clientset,
		Namespace:         Namespace,
	}
	podcontroller := controller.PodController{
		PodClientset: Clientset,
		PodClient:    crdClient,
		Namespace:    Namespace,
	}
	jobcontroller := controller.JobController{
		JobClientset: Clientset,
		JobClient:    crdClient,
		Namespace:    Namespace,
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go pgTaskcontroller.Run(ctx)
	go pgIngestcontroller.Run(ctx)
	go pgClustercontroller.Run(ctx)
	go pgReplicacontroller.Run(ctx)
	go pgBackupcontroller.Run(ctx)
	go pgUpgradecontroller.Run(ctx)
	go pgPolicycontroller.Run(ctx)
	go podcontroller.Run(ctx)
	go jobcontroller.Run(ctx)

	go cluster.MajorUpgradeProcess(Clientset, crdClient, Namespace)

	cluster.InitializeAutoFailover(Clientset, crdClient, Namespace)

	fmt.Print("at end of setup, beginning wait...")

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

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
