package apiserver

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	crdclient "github.com/crunchydata/kraken/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var RestClient *rest.RESTClient
var Clientset *kubernetes.Clientset

func init() {
	log.Infoln("apiserver starts")

	ConnectToKube()

}

func ConnectToKube() {

	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	//Clientset, err = apiextensionsclient.NewForConfig(config)
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// make a new config for our extension's API group, using the first config as a baseline
	RestClient, _, err = crdclient.NewClient(config)
	if err != nil {
		panic(err)
	}

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
