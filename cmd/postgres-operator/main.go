package main

/*
Copyright 2017 - 2022 Crunchy Data
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
	"fmt"
	"os"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller"
	"github.com/crunchydata/postgres-operator/internal/controller/manager"
	nscontroller "github.com/crunchydata/postgres-operator/internal/controller/namespace"
	crunchylog "github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/ns"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/operator"
)

func main() {
	if flush, err := initOpenTelemetry(); err != nil {
		log.Fatal(err)
	} else {
		defer flush()
	}

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	// add logging configuration
	crunchylog.CrunchyLogger(crunchylog.SetParameters())
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	// give time for pgo-event to start up
	time.Sleep(time.Duration(5) * time.Second)

	newKubernetesClient := func() (*kubeapi.Client, error) {
		config, err := kubeapi.LoadClientConfig()
		if err != nil {
			return nil, err
		}

		config.Wrap(otelTransportWrapper())

		return kubeapi.NewClientForConfig(config)
	}

	client, err := newKubernetesClient()
	if err != nil {
		log.Fatal(err)
	}

	operator.Initialize(client)

	// Configure namespaces for the Operator.  This includes determining the namespace
	// operating mode, creating/updating namespaces (if permitted), and obtaining a valid
	// list of target namespaces for the operator install
	namespaceList, err := operator.SetupNamespaces(client)
	if err != nil {
		log.Fatalf("Error configuring operator namespaces: %v", err)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	// create a new controller manager with controllers for all current namespaces and then run
	// all of those controllers
	controllerManager, err := manager.NewControllerManager(namespaceList, operator.Pgo,
		operator.PgoNamespace, operator.InstallationName, operator.NamespaceOperatingMode())
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("controller manager created")

	controllerManager.NewKubernetesClient = newKubernetesClient

	// If not using the "disabled" namespace operating mode, start a real namespace controller
	// that is able to resond to namespace events in the Kube cluster.  If using the "disabled"
	// operating mode, then create a fake client containing all namespaces defined for the install
	// (i.e. via the NAMESPACE environment variable) and use that to create the namespace
	// controller.  This allows for namespace and RBAC reconciliation logic to be run in a
	// consistent manner regardless of the namespace operating mode being utilized.
	if operator.NamespaceOperatingMode() != ns.NamespaceOperatingModeDisabled {
		if err := createAndStartNamespaceController(client, controllerManager,
			stopCh); err != nil {
			log.Fatal(err)
		}
	} else {
		fakeClient, err := ns.CreateFakeNamespaceClient(operator.InstallationName)
		if err != nil {
			log.Fatal(err)
		}
		if err := createAndStartNamespaceController(fakeClient, controllerManager,
			stopCh); err != nil {
			log.Fatal(err)
		}
	}

	defer controllerManager.RemoveAll()

	log.Info("PostgreSQL Operator initialized and running, waiting for signal to exit")
	<-stopCh
	log.Infof("Signal received, now exiting")
}

// createAndStartNamespaceController creates a namespace controller and then starts it
func createAndStartNamespaceController(kubeClientset kubernetes.Interface,
	controllerManager controller.Manager, stopCh <-chan struct{}) error {
	nsKubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset,
		time.Duration(*operator.Pgo.Pgo.NamespaceRefreshInterval)*time.Second,
		kubeinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = fmt.Sprintf("%s=%s,%s=%s",
				config.LABEL_VENDOR, config.LABEL_CRUNCHY,
				config.LABEL_PGO_INSTALLATION_NAME, operator.InstallationName)
		}))
	nsController, err := nscontroller.NewNamespaceController(controllerManager,
		nsKubeInformerFactory.Core().V1().Namespaces(),
		*operator.Pgo.Pgo.NamespaceWorkerCount)
	if err != nil {
		return err
	}

	// start the namespace controller
	nsKubeInformerFactory.Start(stopCh)

	if ok := cache.WaitForNamedCacheSync("namespace", stopCh,
		nsKubeInformerFactory.Core().V1().Namespaces().Informer().HasSynced); !ok {
		return fmt.Errorf("failed waiting for namespace cache to sync")
	}

	for i := 0; i < nsController.WorkerCount(); i++ {
		go nsController.RunWorker(stopCh)
	}

	log.Debug("namespace controller is now running")

	return nil
}
