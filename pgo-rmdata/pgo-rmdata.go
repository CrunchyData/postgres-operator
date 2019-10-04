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
	"flag"
	crunchylog "github.com/crunchydata/postgres-operator/logging"
	"github.com/crunchydata/postgres-operator/pgo-rmdata/rmdata"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"os"
)

var request rmdata.Request

var Clientset *kubernetes.Clientset

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	request = rmdata.Request{
		RemoveData:   false,
		IsReplica:    false,
		IsBackup:     false,
		RemoveBackup: false,
		ClusterName:  "",
		ReplicaName:  "",
		Namespace:    "",
	}
	flag.BoolVar(&request.RemoveData, "remove-data", false, "")
	flag.BoolVar(&request.IsReplica, "is-replica", false, "")
	flag.BoolVar(&request.IsBackup, "is-backup", false, "")
	flag.BoolVar(&request.RemoveBackup, "remove-backup", false, "")
	flag.StringVar(&request.ClusterName, "pg-cluster", "", "")
	flag.StringVar(&request.ReplicaName, "replica-name", "", "")
	flag.StringVar(&request.Namespace, "namespace", "", "")
	flag.Parse()

	crunchylog.CrunchyLogger(crunchylog.SetParameters())
	if os.Getenv("CRUNCHY_DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	var err error
	request.RESTConfig, err = buildConfig(*kubeconfig)

	if err != nil {
		log.Fatalln(err.Error())
	}

	request.Clientset, err = kubernetes.NewForConfig(request.RESTConfig)
	if err != nil {
		log.Fatalln(err.Error())
	}

	request.RESTClient, _, err = util.NewClient(request.RESTConfig)
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Infoln("pgo-rmdata starts")
	log.Infof("request is %s", request.String())

	rmdata.Delete(request)

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
