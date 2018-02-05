/*
Copyright 2016 The Kubernetes Authors.

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

	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api"
	//"k8s.io/client-go/pkg/api/errors"

	//"k8s.io/client-go/pkg/runtime"
	//"k8s.io/client-go/pkg/runtime/serializer"

	//"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	//"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	var namespace = "default"
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	secret := v1.Secret{}
	secret.Name = "pgroot-secret"
	//secret.ObjectMeta.Name = "pgroot-secret"
	secret.Data = make(map[string][]byte)
	secret.Data["username"] = []byte("testuser")
	secret.Data["password"] = []byte("mypassword")

	_, err = clientset.Secrets(namespace).Create(&secret)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("Created secret")
	}

}
