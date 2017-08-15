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

// Package main is the main function for the crunchy operator
package server

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"os"
	"time"

	"github.com/crunchydata/postgres-operator/operator/backup"
	"github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/operator/upgrade"
	"github.com/crunchydata/postgres-operator/tpr"

	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const PG_VERSION = "v1"
const PG_GROUP = "crunchydata.com"
const PG_CLONE = "pg-clone.crunchydata.com"
const PG_POLICY_LOG = "pg-policylog.crunchydata.com"
const PG_POLICY = "pg-policy.crunchydata.com"
const PG_CLUSTER = "pg-cluster.crunchydata.com"
const PG_BACKUP = "pg-backup.crunchydata.com"
const PG_UPGRADE = "pg-upgrade.crunchydata.com"

var (
	config     *rest.Config
	Stopchan   chan struct{}
	Clientset  *kubernetes.Clientset
	Restclient *rest.RESTClient
	Namespace  string
)

func Execute() {
	kubeconfig := flag.String("kubeconfig", "", "the path to a kubeconfig, specifies this tool runs outside the cluster")
	var debug = flag.Bool("debug", false, "defaults to false")
	var err error
	flag.Parse()

	var debugEnv = os.Getenv("DEBUG")
	Namespace = os.Getenv("NAMESPACE")

	if Namespace == "" {
		Namespace = "default"
	}

	if *debug || debugEnv != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	Restclient, err = buildClientFromFlags(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Info("error creating cluster config ")
		panic(err.Error())
	}

	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("error creating kube client ")
		panic(err.Error())
	}

	initializeResources(Clientset)

	//wait a bit to let the resources be created
	time.Sleep(2000 * time.Millisecond)

	log.Info("---------------------------------------------------------")

	Stopchan = make(chan struct{}, 1)

	go cluster.Process(Clientset, Restclient, Stopchan, Namespace)
	go backup.Process(Clientset, Restclient, Stopchan, Namespace)
	go backup.ProcessJobs(Clientset, Restclient, Stopchan, Namespace)
	go upgrade.Process(Clientset, Restclient, Stopchan, Namespace)
	go upgrade.MajorUpgradeProcess(Clientset, Restclient, Stopchan, Namespace)
	go cluster.ProcessClone(Clientset, Restclient, Stopchan, Namespace)
	go cluster.CompleteClone(config, Clientset, Restclient, Stopchan, Namespace)
	go cluster.ProcessPolicies(Clientset, Restclient, Stopchan, Namespace)
	go cluster.ProcessPolicylog(Clientset, Restclient, Stopchan, Namespace)

}

func buildClientFromFlags(kubeconfig string) (*rest.RESTClient, error) {
	config, err := configFromFlags(kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &schema.GroupVersion{
		Group:   PG_GROUP,
		Version: PG_VERSION,
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	schemeBuilder.AddToScheme(api.Scheme)

	return rest.RESTClientFor(config)
}

func configFromFlags(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		schema.GroupVersion{Group: PG_GROUP, Version: PG_VERSION},
		&tpr.PgCluster{},
		&tpr.PgClusterList{},
		&tpr.PgClone{},
		&tpr.PgCloneList{},
		&tpr.PgBackup{},
		&tpr.PgBackupList{},
		&tpr.PgPolicy{},
		&tpr.PgPolicyList{},
		&tpr.PgPolicylog{},
		&tpr.PgPolicylogList{},
		&tpr.PgUpgrade{},
		&tpr.PgUpgradeList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
		&v1batch.Job{},
		&v1batch.JobList{},
	)

	return nil
}

func initializeResources(clientset *kubernetes.Clientset) {
	// initialize third party resources if they do not exist

	tpr, err := clientset.Extensions().ThirdPartyResources().Get(PG_CLUSTER, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: PG_CLUSTER,
				},
				Versions: []v1beta1.APIVersion{
					{Name: PG_VERSION},
				},
				Description: "A postgres cluster ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get(PG_BACKUP, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: PG_BACKUP,
				},
				Versions: []v1beta1.APIVersion{
					{Name: PG_VERSION},
				},
				Description: "A postgres backup ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get(PG_UPGRADE, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: PG_UPGRADE,
				},
				Versions: []v1beta1.APIVersion{
					{Name: PG_VERSION},
				},
				Description: "A postgres upgrade ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get(PG_POLICY, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: PG_POLICY,
				},
				Versions: []v1beta1.APIVersion{
					{Name: PG_VERSION},
				},
				Description: "A postgres policy ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get(PG_CLONE, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: PG_CLONE,
				},
				Versions: []v1beta1.APIVersion{
					{Name: PG_VERSION},
				},
				Description: "A postgres clone ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	tpr, err = clientset.Extensions().ThirdPartyResources().Get(PG_POLICY_LOG, meta_v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tpr := &v1beta1.ThirdPartyResource{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: PG_POLICY_LOG,
				},
				Versions: []v1beta1.APIVersion{
					{Name: PG_VERSION},
				},
				Description: "A postgres policy log ThirdPartyResource",
			}

			result, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				panic(err)
			}
			log.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			panic(err)
		}
	} else {
		log.Infof("SKIPPING: already exists %#v\n", tpr)
	}

}
