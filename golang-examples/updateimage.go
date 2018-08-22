package main

import "fmt"
import "flag"

import log "github.com/Sirupsen/logrus"

import "github.com/crunchydata/postgres-operator/kubeapi"
import "k8s.io/client-go/tools/clientcmd"
import "k8s.io/client-go/kubernetes"

type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func main() {

	fmt.Println("secrets...")
	kubeconfig := flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")

	flag.Parse()
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	deploymentName := "fango"

	jsonvalue := "crunchydata/crunchy-postgres:centos7-10.4-1.8.5"
	jsonpath := "/spec/template/spec/containers/0/image"
	err = kubeapi.PatchDeployment(clientset, deploymentName, "demo", jsonpath, jsonvalue)
	log.Info("patch succeeded")

}
