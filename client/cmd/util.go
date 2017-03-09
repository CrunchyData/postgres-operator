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
	//"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	//for TPR client
	"github.com/crunchydata/operator/tpr"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	//"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"
)

var Config *rest.Config
var Clientset *kubernetes.Clientset
var Tprclient *rest.RESTClient

func ConnectToKube() {

	//fmt.Println("ConnectToKube called")
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

	//verify that the TPRs exist in the Kube
	//var tpr *v1beta1.ThirdPartyResource
	_, err = Clientset.Extensions().ThirdPartyResources().Get("crunchy-cluster.crunchydata.com")
	if err != nil {
		panic(err.Error())
	}
	_, err = Clientset.Extensions().ThirdPartyResources().Get("crunchy-database.crunchydata.com")
	if err != nil {
		panic(err.Error())
	}

	var tprconfig *rest.Config
	tprconfig = Config
	configureTPRClient(tprconfig)

	Tprclient, err = rest.RESTClientFor(tprconfig)
	if err != nil {
		panic(err.Error())
	}

	//fmt.Println("connected to kube. at " + KubeconfigPath)

}

func configureTPRClient(config *rest.Config) {
	groupversion := unversioned.GroupVersion{
		Group:   "crunchydata.com",
		Version: "v1",
	}

	config.GroupVersion = &groupversion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				groupversion,
				&tpr.CrunchyDatabase{},
				&tpr.CrunchyDatabaseList{},
				&tpr.CrunchyCluster{},
				&tpr.CrunchyClusterList{},
				&api.ListOptions{},
				&api.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(api.Scheme)
}
