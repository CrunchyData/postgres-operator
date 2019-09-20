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
	"time"

	crunchylog "github.com/crunchydata/postgres-operator/logging"
	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/operatorupgrade"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/client-go/kubernetes"
)

var Clientset *kubernetes.Clientset

//var log *logrus.Entry

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	//add logging configuration
	crunchylog.CrunchyLogger(crunchylog.SetParameters())
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	//give time for pgo-event to start up
	time.Sleep(time.Duration(5) * time.Second)

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

	namespaceList := ns.GetNamespaces(Clientset, operator.InstallationName)
	log.Debugf("watching the following namespaces: [%v]", namespaceList)

	//validate the NAMESPACE env var
	err = ns.ValidateNamespaces(Clientset, operator.InstallationName, operator.PgoNamespace)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	// start a controller on instances of our custom resource
	ctx, cancelFunc := context.WithCancel(context.Background())

	pgTaskcontroller := &controller.PgtaskController{
		PgtaskConfig:       config,
		PgtaskClient:       crdClient,
		PgtaskScheme:       crdScheme,
		PgtaskClientset:    Clientset,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Ctx:                ctx,
		InformerNamespaces: make(map[string]struct{}),
	}

	pgClustercontroller := &controller.PgclusterController{
		PgclusterClient:    crdClient,
		PgclusterScheme:    crdScheme,
		PgclusterClientset: Clientset,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Ctx:                ctx,
		InformerNamespaces: make(map[string]struct{}),
	}
	pgReplicacontroller := &controller.PgreplicaController{
		PgreplicaClient:    crdClient,
		PgreplicaScheme:    crdScheme,
		PgreplicaClientset: Clientset,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		Ctx:                ctx,
		InformerNamespaces: make(map[string]struct{}),
	}
	pgBackupcontroller := &controller.PgbackupController{
		PgbackupClient:     crdClient,
		PgbackupScheme:     crdScheme,
		PgbackupClientset:  Clientset,
		Ctx:                ctx,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		UpdateQueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		InformerNamespaces: make(map[string]struct{}),
	}
	pgPolicycontroller := &controller.PgpolicyController{
		PgpolicyClient:     crdClient,
		PgpolicyScheme:     crdScheme,
		PgpolicyClientset:  Clientset,
		Ctx:                ctx,
		InformerNamespaces: make(map[string]struct{}),
	}
	podcontroller := &controller.PodController{
		PodClientset:       Clientset,
		PodClient:          crdClient,
		Ctx:                ctx,
		InformerNamespaces: make(map[string]struct{}),
	}
	jobcontroller := &controller.JobController{
		JobClientset:       Clientset,
		JobClient:          crdClient,
		Ctx:                ctx,
		InformerNamespaces: make(map[string]struct{}),
	}
	nscontroller := &controller.NamespaceController{
		NamespaceClientset:     Clientset,
		NamespaceClient:        crdClient,
		Ctx:                    ctx,
		ThePodController:       podcontroller,
		TheJobController:       jobcontroller,
		ThePgpolicyController:  pgPolicycontroller,
		ThePgbackupController:  pgBackupcontroller,
		ThePgreplicaController: pgReplicacontroller,
		ThePgclusterController: pgClustercontroller,
		ThePgtaskController:    pgTaskcontroller,
	}

	defer cancelFunc()
	go pgTaskcontroller.Run()
	go pgTaskcontroller.RunWorker()
	go pgClustercontroller.Run()
	go pgClustercontroller.RunWorker()
	go pgReplicacontroller.Run()
	go pgReplicacontroller.RunWorker()
	go pgBackupcontroller.Run()
	go pgBackupcontroller.RunWorker()
	go pgBackupcontroller.RunUpdateWorker()
	go pgPolicycontroller.Run()
	go podcontroller.Run()
	go nscontroller.Run()
	go jobcontroller.Run()

	cluster.InitializeAutoFailover(Clientset, crdClient, namespaceList)

	operatorupgrade.OperatorUpdateCRPgoVersion(Clientset, crdClient, namespaceList)

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
