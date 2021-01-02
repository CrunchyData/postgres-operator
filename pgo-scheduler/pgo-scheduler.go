package main

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller"
	nscontroller "github.com/crunchydata/postgres-operator/internal/controller/namespace"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	crunchylog "github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/ns"
	"github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	sched "github.com/crunchydata/postgres-operator/pgo-scheduler/scheduler"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	schedulerLabel       = "crunchy-scheduler=true"
	pgoNamespaceEnv      = "PGO_OPERATOR_NAMESPACE"
	timeoutEnv           = "TIMEOUT"
	inCluster            = true
	namespaceWorkerCount = 1
)

var nsRefreshInterval = 10 * time.Minute
var installationName string
var namespace string
var pgoNamespace string
var timeout time.Duration
var seconds int
var kubeClient kubernetes.Interface

// this is used to prevent a race condition where an informer is being created
// twice when a new scheduler-enabled ConfigMap is added.
var informerNsMutex sync.Mutex
var informerNamespaces map[string]struct{}

// NamespaceOperatingMode defines the namespace operating mode for the cluster,
// e.g. "dynamic", "readonly" or "disabled".  See type NamespaceOperatingMode
// for detailed explanations of each mode available.
var namespaceOperatingMode ns.NamespaceOperatingMode

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

	// Configure namespaces for the Scheduler.  This includes determining the namespace
	// operating mode and obtaining a valid list of target namespaces for the operator install.
	if err := setNamespaceOperatingMode(kubeClient); err != nil {
		log.Errorf("Error configuring operator namespaces: %w", err)
		os.Exit(2)
	}
}

func main() {
	log.Info("Starting Crunchy Scheduler")
	//give time for pgo-event to start up
	time.Sleep(time.Duration(5) * time.Second)

	scheduler := scheduler.New(schedulerLabel, pgoNamespace, kubeClient)
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

	nsList, err := ns.GetInitialNamespaceList(kubeClient, namespaceOperatingMode,
		installationName, pgoNamespace)
	if err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to obtain initial namespace list: %s", err)
		os.Exit(2)
	}

	log.WithFields(log.Fields{}).Infof("Watching namespaces: %s", nsList)

	controllerManager, err := sched.NewControllerManager(nsList, scheduler, installationName, namespaceOperatingMode)
	if err != nil {
		log.WithFields(log.Fields{}).Fatalf("Failed to create controller manager: %s", err)
		os.Exit(2)
	}
	controllerManager.RunAll()

	// if the namespace operating mode is not disabled, then create and start a namespace
	// controller
	if namespaceOperatingMode != ns.NamespaceOperatingModeDisabled {
		if err := createAndStartNamespaceController(kubeClient, controllerManager,
			scheduler, stop); err != nil {
			log.WithFields(log.Fields{}).Fatalf("Failed to create namespace informer factory: %s",
				err)
			os.Exit(2)
		}
	}

	// If not using the "disabled" namespace operating mode, start a real namespace controller
	// that is able to resond to namespace events in the Kube cluster.  If using the "disabled"
	// operating mode, then create a fake client containing all namespaces defined for the install
	// (i.e. via the NAMESPACE environment variable) and use that to create the namespace
	// controller.  This allows for namespace and RBAC reconciliation logic to be run in a
	// consistent manner regardless of the namespace operating mode being utilized.
	if namespaceOperatingMode != ns.NamespaceOperatingModeDisabled {
		if err := createAndStartNamespaceController(kubeClient, controllerManager, scheduler,
			stop); err != nil {
			log.Fatal(err)
		}
	} else {
		fakeClient, err := ns.CreateFakeNamespaceClient(installationName)
		if err != nil {
			log.Fatal(err)
		}
		if err := createAndStartNamespaceController(fakeClient, controllerManager, scheduler,
			stop); err != nil {
			log.Fatal(err)
		}
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

// setNamespaceOperatingMode set the namespace operating mode for the Operator by calling the
// proper utility function to determine which mode is applicable based on the current
// permissions assigned to the Operator Service Account.
func setNamespaceOperatingMode(clientset kubernetes.Interface) error {
	nsOpMode, err := ns.GetNamespaceOperatingMode(clientset)
	if err != nil {
		return err
	}
	namespaceOperatingMode = nsOpMode

	return nil
}

// createAndStartNamespaceController creates a namespace controller and then starts it
func createAndStartNamespaceController(kubeClientset kubernetes.Interface,
	controllerManager controller.Manager, schedular *sched.Scheduler,
	stopCh <-chan struct{}) error {

	nsKubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset,
		nsRefreshInterval,
		kubeinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = fmt.Sprintf("%s=%s,%s=%s",
				config.LABEL_VENDOR, config.LABEL_CRUNCHY,
				config.LABEL_PGO_INSTALLATION_NAME, installationName)
		}))

	nsController, err := nscontroller.NewNamespaceController(controllerManager,
		nsKubeInformerFactory.Core().V1().Namespaces(), namespaceWorkerCount)
	if err != nil {
		return err
	}

	// start the namespace controller
	nsKubeInformerFactory.Start(stopCh)

	if ok := cache.WaitForNamedCacheSync("scheduler namespace", stopCh,
		nsKubeInformerFactory.Core().V1().Namespaces().Informer().HasSynced); !ok {
		return fmt.Errorf("failed waiting for scheduler namespace cache to sync")
	}

	for i := 0; i < nsController.WorkerCount(); i++ {
		go nsController.RunWorker(stopCh)
	}

	log.Debug("scheduler namespace controller is now running")

	return nil
}
