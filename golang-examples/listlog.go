package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/crunchydata/postgres-operator/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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

	timeout := time.Duration(2 * time.Second)
	lo := metav1.ListOptions{LabelSelector: "name=" + "lspvc"}
	podPhase := corev1.PodSucceeded
	err = util.WaitUntilPod(clientset, lo, podPhase, timeout, "default")
	if err != nil {
		fmt.Println("error waiting on lspvc pod to complete" + err.Error())
	}
	logOptions := corev1.PodLogOptions{}
	podName := "lspvc-donut"
	req := clientset.Core().Pods("default").GetLogs(podName, &logOptions)
	if req == nil {
		fmt.Println("error in get logs for " + podName)
	} else {
		fmt.Println("got the logs for " + podName)
	}

	readCloser, err := req.Stream()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer func() {
		if readCloser != nil {
			readCloser.Close()
		}
	}()
	var buf2 bytes.Buffer
	_, err = io.Copy(&buf2, readCloser)
	fmt.Printf("backups are... \n%s", buf2.String())

}
