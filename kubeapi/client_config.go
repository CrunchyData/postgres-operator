package kubeapi

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

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

// NewClient returns a Clientset and its underlying configuration.
func NewClient() (*rest.Config, *kubernetes.Clientset, error) {
	config, err := loadClientConfig()
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	return config, clientset, err
}

// NewControllerClient returns a Clientset and its underlying configuration.
// The Clientset is configured with a higher than normal QPS and Burst limit.
func NewControllerClient() (*rest.Config, *kubernetes.Clientset, error) {
	config, err := loadClientConfig()
	if err != nil {
		return nil, nil, err
	}

	// Match the settings applied by sigs.k8s.io/controller-runtime@v0.4.0;
	// see https://github.com/kubernetes-sigs/controller-runtime/issues/365.
	if config.QPS == 0.0 {
		config.QPS = 20.0
		config.Burst = 30.0
	}

	clientset, err := kubernetes.NewForConfig(config)
	return config, clientset, err
}
