package main

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/controller"
	"github.com/crunchydata/postgres-operator/kubeapi"
	crunchylog "github.com/crunchydata/postgres-operator/logging"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	sched "github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	schedulerLabel  = "crunchy-scheduler=true"
	pgoNamespaceEnv = "PGO_OPERATOR_NAMESPACE"
	timeoutEnv      = "TIMEOUT"
	inCluster       = true
)

var installationName string
var namespace string
var pgoNamespace string
var namespaceList []string
var timeout time.Duration
var seconds int
var kubeClient *kubernetes.Clientset

// this is used to prevent a race condition where an informer is being created
// twice when a new scheduler-enabled ConfigMap is added.
var informerNsMutex sync.Mutex
var informerNamespaces map[string]struct{}

func init() {
	var err error
	log.SetLevel(log.InfoLevel)

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	//add logging configuration
	crunchylog.CrunchyLogger(crunchylog.SetParameters())
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	installationName = os.Getenv("PGO_INSTALLATION_NAME")
	if installationName == "" {
		log.Fatal("PGO_INSTALLATION_NAME env var is not set")
	} else {
		log.Info("PGO_INSTALLATION_NAME set to " + installationName)
	}

	pgoNamespace = os.Getenv(pgoNamespaceEnv)
	if pgoNamespace == "" {
		log.WithFields(log.Fields{}).Fatalf("Failed to get PGO_OPERATOR_NAMESPACE environment: %s", pgoNamespaceEnv)
	}

	secondsEnv := os.Getenv(timeoutEnv)
	seconds = 300
	if secondsEnv == "" {
		log.WithFields(log.Fields{}).Info("No timeout set, defaulting to 300 seconds")
	} else {
		seconds, err = strconv.Atoi(secondsEnv)
		if err != nil {
			log.WithFields(log.Fields{}).Fatalf("Failed to convert timeout env to seconds: %s", err)
		}
	}

	log.WithFields(log.Fields{}).Infof("Setting timeout to: %d", seconds)
	timeout = time.Second * time.Duration(seconds)

	_, kubeClient, err = kubeapi.NewKubeClient()
	if err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to connect to kubernetes: %s", err)
	}

	var Pgo config.PgoConfig
	if err := Pgo.GetConfig(kubeClient, pgoNamespace); err != nil {
		log.WithFields(log.Fields{}).Fatalf("error in Pgo configuration: %s", err)
	}
	namespaceList = ns.GetNamespaces(kubeClient, installationName)
	log.Debugf("watching the following namespaces: [%v]", namespaceList)

}

func main() {
	log.Info("Starting Crunchy Scheduler")
	//give time for pgo-event to start up
	time.Sleep(time.Duration(5) * time.Second)

	scheduler := scheduler.New(schedulerLabel, pgoNamespace, namespaceList, kubeClient)
	scheduler.CronClient.Start()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.WithFields(log.Fields{
			"signal": sig,
		}).Warning("Received signal")
		done <- true
	}()

	stop := make(chan struct{})

	log.WithFields(log.Fields{}).Infof("Watching namespaces: %s", namespaceList)

	controllerManager, err := sched.NewControllerManager(namespaceList, scheduler)
	if err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to create controller manager: %s", err)
	}
	controllerManager.RunAll()

	nsKubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, 0)
	if err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to create namespace informer factory: %s", err)
	}
	SetupNamespaceController(installationName, scheduler,
		nsKubeInformerFactory.Core().V1().Namespaces(), controllerManager)
	nsKubeInformerFactory.Start(stop)

	for {
		select {
		case <-done:
			log.Warning("Shutting down scheduler")
			scheduler.CronClient.Stop()
			close(stop)
			os.Exit(0)
		default:
			time.Sleep(time.Second * 1)
		}
	}
}

func newKubeClient() (*kubernetes.Clientset, error) {
	var client *kubernetes.Clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		return client, err
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return client, err
	}

	return client, nil
}

// SetupNamespaceController sets up a namespace controller that monitors for namespace add and
// delete events, and then either creates or removes controllers for those namespaces
func SetupNamespaceController(installationName string, scheduler *scheduler.Scheduler,
	informer coreinformers.NamespaceInformer, controllerManager controller.ManagerInterface) {

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ns, ok := obj.(*v1.Namespace)
			if !ok {
				log.WithFields(log.Fields{}).Error("Could not convert runtime object to Namespace..")
			} else {
				labels := ns.GetObjectMeta().GetLabels()
				if labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY && labels[config.LABEL_PGO_INSTALLATION_NAME] == installationName {
					log.WithFields(log.Fields{}).Infof("Added namespace: %s", ns.Name)
					controllerManager.AddAndRunControllerGroup(ns.Name)
				} else {
					log.WithFields(log.Fields{}).Infof("Not adding namespace since it is not owned by this Operator installation: %s", ns.Name)
				}
			}

		},
		DeleteFunc: func(obj interface{}) {
			ns, ok := obj.(*v1.Namespace)
			if !ok {
				log.WithFields(log.Fields{}).Error("Could not convert runtime object to Namespace..")
			} else {
				log.WithFields(log.Fields{}).Infof("Deleted namespace: %s", ns.Name)
			}
			controllerManager.RemoveGroup(ns.Name)
		},
	})
}
