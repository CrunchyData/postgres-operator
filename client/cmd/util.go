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

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	//for TPR client
	"github.com/crunchydata/postgres-operator/tpr"
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

	//verify that the TPRs exist in the Kube
	//var tpr *v1beta1.ThirdPartyResource
	_, err = Clientset.Extensions().ThirdPartyResources().Get("pg-cluster.crunchydata.com")
	if err != nil {
		log.Error(err.Error())
		log.Error("required pg-cluster.crunchydata.com TPR was not found on your kube cluster")
		os.Exit(2)
	}
	_, err = Clientset.Extensions().ThirdPartyResources().Get("pg-backup.crunchydata.com")
	if err != nil {
		log.Error(err.Error())
		log.Error("required pg-backup.crunchydata.com TPR was not found on your kube cluster")
		os.Exit(2)
	}
	_, err = Clientset.Extensions().ThirdPartyResources().Get("pg-upgrade.crunchydata.com")
	if err != nil {
		log.Error(err.Error())
		log.Error("required pg-upgrade.crunchydata.com TPR was not found on your kube cluster")
		os.Exit(2)
	}
	_, err = Clientset.Extensions().ThirdPartyResources().Get("pg-clone.crunchydata.com")
	if err != nil {
		log.Error(err.Error())
		log.Error("required pg-clone.crunchydata.com TPR was not found on your kube cluster")
		os.Exit(2)
	}

	var tprconfig *rest.Config
	tprconfig = Config
	configureTPRClient(tprconfig)

	Tprclient, err = rest.RESTClientFor(tprconfig)
	if err != nil {
		log.Error(err.Error())
		log.Error("can not get client to TPR")
		os.Exit(2)
	}

	log.Debug("connected to kube. at " + KubeconfigPath)

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
				&tpr.PgClone{},
				&tpr.PgCloneList{},
				&tpr.PgCluster{},
				&tpr.PgClusterList{},
				&tpr.PgBackup{},
				&tpr.PgBackupList{},
				&tpr.PgPolicy{},
				&tpr.PgPolicyList{},
				&tpr.PgUpgrade{},
				&tpr.PgUpgradeList{},
				&api.ListOptions{},
				&api.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(api.Scheme)
}

func PrintSecrets(db string) {

	lo := v1.ListOptions{LabelSelector: "pg-database=" + db}
	secrets, err := Clientset.Secrets(Namespace).List(lo)
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

func GetMasterSecretPassword(db string) string {

	lo := v1.ListOptions{LabelSelector: "pg-database=" + db}
	secrets, err := Clientset.Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return "error"
	}

	log.Debug("secrets for " + db)
	secretName := db + tpr.PGMASTER_SECRET_SUFFIX
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
