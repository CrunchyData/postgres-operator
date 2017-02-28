package main

import (
	"flag"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
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
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		lo := v1.ListOptions{LabelSelector: "k8s-app=kube-dns"}
		fmt.Println("label selector is " + lo.LabelSelector)
		pods, err := clientset.Core().Pods("").List(lo)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		for _, pod := range pods.Items {
			fmt.Println("pod Name " + pod.ObjectMeta.Name)
			fmt.Println("pod phase is " + pod.Status.Phase)
		}
		time.Sleep(10 * time.Second)
	}
}
