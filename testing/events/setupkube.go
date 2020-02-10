package eventtest

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
	"flag"
	//"fmt"
	//kerrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	//"os/exec"
	//"time"
	//meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig      = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	namespace       = flag.String("namespace", "pgouser1", "namespace to test within ")
	username        = flag.String("username", "pgouser1", "username to test within ")
	rolename        = flag.String("rolename", "pgoadmin", "rolename to test with")
	testclustername = flag.String("clustername", "", "cluster name to test with")
	eventTcpAddress = flag.String("event-tcp-address", "localhost:14150", "tcp port to the event pgo-event port")

	EventTCPAddress = "localhost:14150"
	Namespace       = "pgouser1"
	TestUsername    = "pgouser1"
	TestClusterName = "foo"
	TestRolename    = "pgoadmin"
	SLEEP_SECS      = 10
)

func SetupKube() (*kubernetes.Clientset, *rest.RESTClient) {

	log.SetLevel(log.DebugLevel)
	log.Debug("debug flag set to true")

	var RESTClient *rest.RESTClient

	flag.Parse()

	if *eventTcpAddress != "" {
		EventTCPAddress = *eventTcpAddress
		log.Infof("connecting to event router at %s\n", EventTCPAddress)
	}

	if *namespace == "" {
		val := os.Getenv("PGO_NAMESPACE")
		if val == "" {
			log.Info("PGO_NAMESPACE env var is required for smoketest")
			os.Exit(2)
		}
	} else {
		Namespace = *namespace
	}
	if *testclustername != "" {
		TestClusterName = *testclustername
	}
	if *username != "" {
		TestUsername = *username
	}
	if *rolename != "" {
		TestRolename = *rolename
	}

	log.Infof("running test in namespace %s\n", Namespace)
	log.Infof("running test as user %s\n", TestUsername)
	log.Infof("running test with pgorole %s\n", TestRolename)
	log.Infof("running test on cluster %s\n", TestClusterName)
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	RESTClient, _, err = util.NewClient(config)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}

	//createTestCluster(clientset)
	verifyExists(RESTClient)

	return clientset, RESTClient
}

// buildConfig ...
func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func verifyExists(RESTClient *rest.RESTClient) {
	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(RESTClient, &cluster, TestClusterName, Namespace)
	if !found || err != nil {
		log.Infof("test cluster %s deployment not found can not continue", TestClusterName)
		os.Exit(2)
	}

	log.Infof("pgcluster %s is found\n", TestClusterName)

}
