package main

/*
Copyright 2019 Crunchy Data
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
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/operatorupgrade"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
)

var Clientset *kubernetes.Clientset

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

	namespaceList := util.GetNamespaces()
	log.Debugf("watching the following namespaces: [%v]", namespaceList)

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating Clientset")
		log.Error(err)
		os.Exit(2)
	}

	// make a new config for our extension's API group, using the first config as a baseline
	crdClient, crdScheme, err := util.NewClient(config)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	operator.Initialize(Clientset)

	//validate the NAMESPACE env var
	err = util.ValidateNamespaces(Clientset)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	// start a controller on instances of our custom resource

	pgTaskcontroller := controller.PgtaskController{
		PgtaskConfig:    config,
		PgtaskClient:    crdClient,
		PgtaskScheme:    crdScheme,
		PgtaskClientset: Clientset,
		Namespace:       namespaceList,
	}

	pgClustercontroller := controller.PgclusterController{
		PgclusterClient:    crdClient,
		PgclusterScheme:    crdScheme,
		PgclusterClientset: Clientset,
		Namespace:          namespaceList,
	}
	pgReplicacontroller := controller.PgreplicaController{
		PgreplicaClient:    crdClient,
		PgreplicaScheme:    crdScheme,
		PgreplicaClientset: Clientset,
		Namespace:          namespaceList,
	}
	pgBackupcontroller := controller.PgbackupController{
		PgbackupClient:    crdClient,
		PgbackupScheme:    crdScheme,
		PgbackupClientset: Clientset,
		Namespace:         namespaceList,
	}
	pgPolicycontroller := controller.PgpolicyController{
		PgpolicyClient:    crdClient,
		PgpolicyScheme:    crdScheme,
		PgpolicyClientset: Clientset,
		Namespace:         namespaceList,
	}
	podcontroller := controller.PodController{
		PodClientset: Clientset,
		PodClient:    crdClient,
		Namespace:    namespaceList,
	}
	jobcontroller := controller.JobController{
		JobClientset: Clientset,
		JobClient:    crdClient,
		Namespace:    namespaceList,
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go pgTaskcontroller.Run(ctx)
	go pgClustercontroller.Run(ctx)
	go pgReplicacontroller.Run(ctx)
	go pgBackupcontroller.Run(ctx)
	go pgPolicycontroller.Run(ctx)
	go podcontroller.Run(ctx)
	go jobcontroller.Run(ctx)

	cluster.InitializeAutoFailover(Clientset, crdClient, namespaceList)

	operatorupgrade.OperatorUpgrade(Clientset, crdClient, namespaceList)

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
