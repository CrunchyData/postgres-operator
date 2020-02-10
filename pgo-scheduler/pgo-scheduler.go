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
	crunchylog "github.com/crunchydata/postgres-operator/logging"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
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

	// set up the data structures for preventing double informers from being
	// created
	informerNsMutex = sync.Mutex{}
	informerNamespaces = map[string]struct{}{}

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

	kubeClient, err = newKubeClient()
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

	for _, namespace := range namespaceList {
		SetupWatch(namespace, scheduler, stop)
	}

	SetupNamespaceWatch(installationName, scheduler, stop)

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

func SetupWatch(namespace string, scheduler *scheduler.Scheduler, stop chan struct{}) {
	// don't create informer for namespace if one has already been created
	informerNsMutex.Lock()
	defer informerNsMutex.Unlock()

	// if the namespace is already in the informer namespaces map, then exit
	if _, ok := informerNamespaces[namespace]; ok {
		return
	}

	informerNamespaces[namespace] = struct{}{}

	watchlist := cache.NewListWatchFromClient(kubeClient.Core().RESTClient(),
		"configmaps", namespace, fields.Everything())

	_, controller := cache.NewInformer(watchlist, &v1.ConfigMap{}, 0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cm, ok := obj.(*v1.ConfigMap)
				if !ok {
					log.WithFields(log.Fields{}).Error("Could not convert runtime object to configmap..")
				}

				if _, ok := cm.Labels["crunchy-scheduler"]; !ok {
					return
				}

				if err := scheduler.AddSchedule(cm); err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("Failed to add schedules")
				}
			},
			DeleteFunc: func(obj interface{}) {
				cm, ok := obj.(*v1.ConfigMap)
				if !ok {
					log.WithFields(log.Fields{}).Error("Could not convert runtime object to configmap..")
				}

				if _, ok := cm.Labels["crunchy-scheduler"]; !ok {
					return
				}
				scheduler.DeleteSchedule(cm)
			},
		},
	)
	go controller.Run(stop)
}

func SetupNamespaceWatch(installationName string, scheduler *scheduler.Scheduler, stop chan struct{}) {
	watchlist := cache.NewListWatchFromClient(kubeClient.Core().RESTClient(),
		"namespaces", "", fields.Everything())

	_, controller := cache.NewInformer(watchlist, &v1.Namespace{}, 0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ns, ok := obj.(*v1.Namespace)
				if !ok {
					log.WithFields(log.Fields{}).Error("Could not convert runtime object to Namespace..")
				} else {
					labels := ns.GetObjectMeta().GetLabels()
					if labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY && labels[config.LABEL_PGO_INSTALLATION_NAME] == installationName {
						log.WithFields(log.Fields{}).Infof("Added namespace: %s", ns.Name)
						SetupWatch(ns.Name, scheduler, stop)
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

			},
		},
	)
	go controller.Run(stop)
}
