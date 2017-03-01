package cmd

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Config *rest.Config
var Clientset *kubernetes.Clientset

func ConnectToKube() {

	//setup connection to kube
	// uses the current context in kubeconfig
	var err error
	Config, err = clientcmd.BuildConfigFromFlags("", KubeconfigPath)
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	Clientset, err = kubernetes.NewForConfig(Config)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("connected to kube. at " + KubeconfigPath)

}
