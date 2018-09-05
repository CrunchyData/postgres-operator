/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"flag"
	"fmt"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	lo := meta_v1.ListOptions{LabelSelector: "pg-cluster=dinner"}
	pods, err := clientset.CoreV1().Pods("demo").List(lo)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	for _, pod := range pods.Items {
		readyCount := 0
		containerCount := 0
		for _, stat := range pod.Status.ContainerStatuses {
			containerCount++
			if stat.Ready {
				readyCount++
			}
			if stat.State.Waiting != nil {
				fmt.Printf("container state is waiting")
			} else if stat.State.Running != nil {
				fmt.Printf("container state is running")
			} else if stat.State.Terminated != nil {
				fmt.Printf("container state is terminated")
			}
		}
		fmt.Printf("Ready %d/%d\n", readyCount, containerCount)
		fmt.Printf("NodeName is %s\n", pod.Spec.NodeName)
		for k, v := range pod.Spec.Volumes {

			if v.Name == "pgdata" || v.Name == "pgwal-volume" {
				if v.VolumeSource.PersistentVolumeClaim != nil {
					fmt.Printf("key %d volname %s pvc %s\n", k, pod.Name, v.VolumeSource.PersistentVolumeClaim.ClaimName)
				}
			}
		}
	}
}
