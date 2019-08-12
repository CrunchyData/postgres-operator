package main

import (
	"flag"
	//	"github.com/crunchydata/postgres-operator/kubeapi"
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

func init() {
	request = rmdata.Request{}
	flag.BoolVar(&request.RemoveData, "remove-data", false, "")
	flag.BoolVar(&request.RemoveBackup, "remove-backup", false, "")
	flag.StringVar(&request.RemoveCluster, "pg-cluster", "", "")
	flag.StringVar(&request.RemoveNamespace, "namespace", "", "")
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")

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
	log.Infof("%s", request.String())

	rmdata.Delete(request)

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
