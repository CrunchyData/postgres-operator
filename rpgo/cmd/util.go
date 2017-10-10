/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

package cmd

import (
	"fmt"
	log "github.com/Sirupsen/logrus"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	//"os"
	//crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	//"k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	//"k8s.io/apimachinery/pkg/runtime/serializer"
)

//var Config *rest.Config
//var Clientset *kubernetes.Clientset

/**
func ConnectToKube() {

	log.Debug("ConnectToKube called")
	//setup connection to kube
	// uses the current context in kubeconfig
	var err error
	Config, err = clientcmd.BuildConfigFromFlags("", KubeconfigPath)
	if err != nil {
		log.Error(err.Error())
		log.Error("can not connect to Kube using kubeconfig")
		os.Exit(2)
	}

	// creates the clientset
	Clientset, err = kubernetes.NewForConfig(Config)
	if err != nil {
		log.Error(err.Error())
		log.Error("can not create client to Kube ")
		os.Exit(2)
	}

	log.Debug("connected to kube. at " + KubeconfigPath)

}
*/

func PrintSecrets(db string) {

	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + db}
	secrets, err := Clientset.Core().Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return
	}

	log.Debug("secrets for " + db)
	for _, s := range secrets.Items {
		fmt.Println("")
		fmt.Println("secret : " + s.ObjectMeta.Name)
		fmt.Println(TREE_BRANCH + "username: " + string(s.Data["username"][:]))
		fmt.Println(TREE_TRUNK + "password: " + string(s.Data["password"][:]))
	}

}

func GetSecretPassword(db, suffix string) string {

	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + db}
	secrets, err := Clientset.Core().Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return "error"
	}

	log.Debug("secrets for " + db)
	secretName := db + suffix
	for _, s := range secrets.Items {
		log.Debug("secret : " + s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretName {
			log.Debug("pgmaster password found")
			return string(s.Data["password"][:])
		}
	}

	log.Error("master secret not found for " + db)
	return "error"

}
