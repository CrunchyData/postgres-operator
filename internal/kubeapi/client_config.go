package kubeapi

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

import (
	clientset "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ControllerClients stores the various clients needed by a controller
type ControllerClients struct {
	Config        *rest.Config
	Kubeclientset *kubernetes.Clientset
	PGOClientset  *clientset.Clientset
	PGORestclient *rest.RESTClient
}

func loadClientConfig() (*rest.Config, error) {
	// The default loading rules try to read from the files specified in the
	// environment or from the home directory.
	loader := clientcmd.NewDefaultClientConfigLoadingRules()

	// The deferred loader tries an in-cluster config if the default loading
	// rules produce no results.
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &clientcmd.ConfigOverrides{},
	).ClientConfig()
}

// NewKubeClient returns a Clientset for interacting with Kubernetes resources, along with
// the REST config used to create the client
func NewKubeClient() (*rest.Config, *kubernetes.Clientset, error) {

	config, err := loadClientConfig()
	if err != nil {
		return nil, nil, err
	}

	clientset, err := createKubeClient(config)
	if err != nil {
		return nil, nil, err
	}
	return config, clientset, err
}

// NewPGOClient returns a Clientset and a REST client for interacting with PostgreSQL Operator
// resources, along with the REST config used to create the clients
func NewPGOClient() (*rest.Config, *rest.RESTClient, *clientset.Clientset, error) {

	config, err := loadClientConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	pgoRESTClient, pgoClientset, err := createPGOClient(config)
	if err != nil {
		return nil, nil, nil, err
	}

	return config, pgoRESTClient, pgoClientset, nil
}

// createKubeClient creates a Kube Clientset using the provided configuration
func createKubeClient(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

// createPGOClient creates a PGO RESTClient and Clientset using the provided configuration
func createPGOClient(config *rest.Config) (*rest.RESTClient, *clientset.Clientset, error) {
	// create a client for pgo resources
	pgoClientset, err := clientset.NewForConfig(config)
	pgoRESTClient := pgoClientset.CrunchydataV1().RESTClient().(*rest.RESTClient)
	if err != nil {
		return nil, nil, err
	}
	return pgoRESTClient, pgoClientset, nil
}

// NewControllerClients returns a ControllerClients struct containing the various clients needed for a controller.
// This includes a Kubernetes Clientset, along with a PGO Clientset with its associated RESTClient  and its underlying configuration.
// The Clientset is configured with a higher than normal QPS and Burst limit.
func NewControllerClients() (*ControllerClients, error) {

	config, err := loadClientConfig()
	if err != nil {
		return nil, err
	}

	// Match the settings applied by sigs.k8s.io/controller-runtime@v0.4.0;
	// see https://github.com/kubernetes-sigs/controller-runtime/issues/365.
	if config.QPS == 0.0 {
		config.QPS = 20.0
		config.Burst = 30.0
	}

	kubeClient, err := createKubeClient(config)
	if err != nil {
		return nil, err
	}

	pgoRESTClient, pgoClientset, err := createPGOClient(config)
	if err != nil {
		return nil, err
	}

	return &ControllerClients{
		Config:        config,
		Kubeclientset: kubeClient,
		PGOClientset:  pgoClientset,
		PGORestclient: pgoRESTClient,
	}, nil
}
