package kubeapi

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Google Kubernetes Engine / Google Cloud Platform authentication provider
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type KubeAPI struct {
	Client *kubernetes.Clientset
	Config *rest.Config
}

func NewConfig() (*rest.Config, error) {
	// The default loading rules try to read from the files specified in the
	// environment or from the home directory.
	loader := clientcmd.NewDefaultClientConfigLoadingRules()

	// The deferred loader tries an in-cluster config if the default loading
	// rules produce no results.
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &clientcmd.ConfigOverrides{}).ClientConfig()
}

func NewForConfig(config *rest.Config) (*KubeAPI, error) {
	var api KubeAPI
	var err error

	api.Config = config
	api.Client, err = kubernetes.NewForConfig(api.Config)

	return &api, err
}
