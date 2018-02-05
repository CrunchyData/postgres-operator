package main

import "fmt"
import "flag"
import "encoding/json"
import log "github.com/Sirupsen/logrus"

//import "k8s.io/client-go/pkg/api/v1"

//import "github.com/crunchydata/postgres-operator/operator/util"
import "k8s.io/client-go/tools/clientcmd"
import "k8s.io/client-go/kubernetes"

//import v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
import api "k8s.io/client-go/pkg/api"

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

	deploymentName := "friday-replica"

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = "/spec/replicas"
	things[0].Value = "1"

	var patchBytes []byte
	patchBytes, err = json.Marshal(things)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
	}
	log.Debug(string(patchBytes))

	_, err = clientset.Deployments("default").Patch(deploymentName, api.JSONPatchType, patchBytes)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		panic(err.Error())
	}
	log.Info("patch succeeded")

}
