package main

import (
	"flag"

	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	clientset "github.com/crunchydata/postgres-operator/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	kubeClient, err2 := kubernetes.NewForConfig(config)
	if err2 != nil {
		panic(err2.Error())
	}
	if kubeClient != nil {
		log.Println("got kube client")
	}

	restclient, _, err := clientset.NewClient(config)
	if err != nil {
		panic(err)
	}
	log.Println("got rest client")

	taskName := "goober" + "-failover"
	//get the task
	task := crv1.Pgtask{}
	err = restclient.Get().
		Resource(crv1.PgtaskResourcePlural).
		Namespace("demo").
		Name(taskName).
		Do().
		Into(&task)
	if err != nil {
		log.Error("error getting pgtask " + taskName)
		log.Error(err)
		return
	}

	log.Println("got pgtask " + task.Name)

}
