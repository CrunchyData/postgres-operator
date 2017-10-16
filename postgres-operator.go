/*
Copyright 2017 The Kubernetes Authors.

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

// Note: the example only works with the code within the same release/branch.
package main

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
	"github.com/crunchydata/postgres-operator/operator/backup"
	"github.com/crunchydata/postgres-operator/operator/upgrade"

	"github.com/crunchydata/postgres-operator/controller"
	"k8s.io/client-go/kubernetes"
)

var Clientset *kubernetes.Clientset

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

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

	// initialize custom resource using a CustomResourceDefinition if it does not exist
	crd, err := crdclient.CreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if crd != nil {
		fmt.Println(crd.Name + " exists ")
	}
	//defer apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, nil)

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
	policylogcrd, err := crdclient.PgpolicylogCreateCustomResourceDefinition(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}
	if policylogcrd != nil {
		fmt.Println(policylogcrd.Name + " exists ")
	}

	// make a new config for our extension's API group, using the first config as a baseline
	crdClient, crdScheme, err := crdclient.NewClient(config)
	if err != nil {
		panic(err)
	}

	// start a controller on instances of our custom resource

	pgClustercontroller := controller.PgclusterController{
		PgclusterClient:    crdClient,
		PgclusterScheme:    crdScheme,
		PgclusterClientset: Clientset,
	}
	pgUpgradecontroller := controller.PgupgradeController{
		PgupgradeClientset: Clientset,
		PgupgradeClient:    crdClient,
		PgupgradeScheme:    crdScheme,
	}
	pgBackupcontroller := controller.PgbackupController{
		PgbackupClient:    crdClient,
		PgbackupScheme:    crdScheme,
		PgbackupClientset: Clientset,
	}
	pgPolicycontroller := controller.PgpolicyController{
		PgpolicyClient:    crdClient,
		PgpolicyScheme:    crdScheme,
		PgpolicyClientset: Clientset,
	}
	pgPolicylogcontroller := controller.PgpolicylogController{
		PgpolicylogClientset: Clientset,
		PgpolicylogClient:    crdClient,
		PgpolicylogScheme:    crdScheme,
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go pgClustercontroller.Run(ctx)
	go pgBackupcontroller.Run(ctx)
	go pgUpgradecontroller.Run(ctx)
	go pgPolicycontroller.Run(ctx)
	go pgPolicylogcontroller.Run(ctx)

	Namespace := "default"
	go backup.ProcessJobs(Clientset, crdClient, Namespace)
	go upgrade.MajorUpgradeProcess(Clientset, crdClient, Namespace)

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
