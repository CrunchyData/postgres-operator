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

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	Namespace := os.Getenv("NAMESPACE")
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

	//TODO is this needed any longer?
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating Clientset")
		panic(err.Error())
	}

	clustercrd, err := crdclient.PgclusterCreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if clustercrd != nil {
		fmt.Println(clustercrd.Name + " exists ")
	}
	//defer apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(clustercrd.Name, nil)

	backupcrd, err := crdclient.PgbackupCreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if backupcrd != nil {
		fmt.Println(backupcrd.Name + " exists ")
	}
	//defer apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(clustercrd.Name, nil)
	upgradecrd, err := crdclient.PgupgradeCreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if upgradecrd != nil {
		fmt.Println(upgradecrd.Name + " exists ")
	}
	//defer apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(clustercrd.Name, nil)
	policycrd, err := crdclient.PgpolicyCreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if policycrd != nil {
		fmt.Println(policycrd.Name + " exists ")
	}
	taskcrd, err := crdclient.PgtaskCreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if taskcrd != nil {
		fmt.Println(taskcrd.Name + " exists ")
	}

	// make a new config for our extension's API group, using the first config as a baseline
	crdClient, crdScheme, err := crdclient.NewClient(config)
	if err != nil {
		panic(err)
	}

	// start a controller on instances of our custom resource

	pgTaskcontroller := controller.PgtaskController{
		PgtaskClient:    crdClient,
		PgtaskScheme:    crdScheme,
		PgtaskClientset: Clientset,
		Namespace:       Namespace,
	}
	pgClustercontroller := controller.PgclusterController{
		PgclusterClient:    crdClient,
		PgclusterScheme:    crdScheme,
		PgclusterClientset: Clientset,
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
	go pgClustercontroller.Run(ctx)
	go pgBackupcontroller.Run(ctx)
	go pgUpgradecontroller.Run(ctx)
	go pgPolicycontroller.Run(ctx)
	go podcontroller.Run(ctx)
	go jobcontroller.Run(ctx)

	//go backup.ProcessJobs(Clientset, crdClient, Namespace)
	go cluster.MajorUpgradeProcess(Clientset, crdClient, Namespace)

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
