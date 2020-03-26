package main

/*
Copyright 2017 - 2020 Crunchy Data
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
	"time"

	"github.com/kubernetes/sample-controller/pkg/signals"

	"github.com/crunchydata/postgres-operator/controller/manager"
	"github.com/crunchydata/postgres-operator/controller/namespace"
	crunchylog "github.com/crunchydata/postgres-operator/logging"
	"github.com/crunchydata/postgres-operator/operator/operatorupgrade"
	log "github.com/sirupsen/logrus"

	kubeinformers "k8s.io/client-go/informers"

	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/operator"
)

func main() {

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

	clients, err := kubeapi.NewControllerClients()
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	kubeClientset := clients.Kubeclientset
	pgoRESTclient := clients.PGORestclient

	operator.Initialize(kubeClientset)

	namespaceList := ns.GetNamespaces(kubeClientset, operator.InstallationName)
	log.Debugf("watching the following namespaces: [%v]", namespaceList)

	//validate the NAMESPACE env var
	err = ns.ValidateNamespaces(kubeClientset, operator.InstallationName, operator.PgoNamespace)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	// create a new controller manager with controllers for all current namespaces and then run
	// all of those controllers
	controllerManager, err := manager.NewControllerManager(namespaceList)
	controllerManager.RunAll()

	nsKubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientset, 0)
	nsController, err := namespace.NewNamespaceController(clients, controllerManager,
		nsKubeInformerFactory.Core().V1().Namespaces())
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}
	nsController.AddNamespaceEventHandler()

	// start the namespace controller
	nsKubeInformerFactory.Start(stopCh)

	defer controllerManager.StopAll()

	operatorupgrade.OperatorUpdateCRPgoVersion(kubeClientset, pgoRESTclient, namespaceList)

	log.Info("PostgreSQL Operator initialized and running, waiting for signal to exit")
	<-stopCh
	log.Infof("Signal received, now exiting")
}
