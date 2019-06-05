package main

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"syscall"
	"time"

	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	schedulerLabel  = "crunchy-scheduler=true"
	namespaceEnv    = "NAMESPACE"
	pgoNamespaceEnv = "PGO_OPERATOR_NAMESPACE"
	timeoutEnv      = "TIMEOUT"
	inCluster       = true
)

var namespace string
var pgoNamespace string
var namespaceList []string
var timeout time.Duration
var seconds int
var kubeClient *kubernetes.Clientset

func init() {
	var err error
	log.SetLevel(log.InfoLevel)

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	namespace = os.Getenv(namespaceEnv)
	if namespace == "" {
		log.WithFields(log.Fields{}).Fatalf("Failed to get NAMESPACE environment: %s", namespaceEnv)
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

	namespaceList = util.GetNamespaces()
	log.Debugf("watching the following namespaces: [%v]", namespaceList)

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

}

func main() {
	log.Info("Starting Crunchy Scheduler")

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
