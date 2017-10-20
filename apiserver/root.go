package apiserver

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

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	crdclient "github.com/crunchydata/postgres-operator/client"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// RestClient ...
var RestClient *rest.RESTClient

// Clientset ...
var Clientset *kubernetes.Clientset

// DebugFlag is the debug flag value
var DebugFlag bool

// Namespace is the namespace flag value
var Namespace string

// TreeTrunk is for debugging only in this context
const TreeTrunk = "└── "

// TreeBranch is for debugging only in this context
const TreeBranch = "├── "

func init() {
	log.Infoln("apiserver starts")

	initConfig()

	ConnectToKube()

}

// ConnectToKube ...
func ConnectToKube() {

	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	//Clientset, err = apiextensionsclient.NewForConfig(config)
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// make a new config for our extension's API group, using the first config as a baseline
	RestClient, _, err = crdclient.NewClient(config)
	if err != nil {
		panic(err)
	}

}

// buildConfig ...
func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func initConfig() {
	//	if cfgFile != "" { // enable ability to specify config file via flag
	//		viper.SetConfigFile(cfgFile)
	//	}

	viper.SetConfigName(".pgo")     // name of config file (without extension)
	viper.AddConfigPath(".")        // adding current directory as first search path
	viper.AddConfigPath("$HOME")    // adding home directory as second search path
	viper.AddConfigPath("/etc/pgo") // adding /etc/pgo directory as third search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		log.Debugf("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Debug("config file not found")
	}

	if DebugFlag || viper.GetBool("Pgo.Debug") {
		log.Debug("debug flag is set to true")
		log.SetLevel(log.DebugLevel)
	}

	//	if KubeconfigPath == "" {
	//		KubeconfigPath = viper.GetString("Kubeconfig")
	//	}
	//	if KubeconfigPath == "" {
	//		log.Error("--kubeconfig flag is not set and required")
	//		os.Exit(2)
	//	}

	//	log.Debug("kubeconfig path is " + viper.GetString("Kubeconfig"))

	if Namespace == "" {
		Namespace = viper.GetString("Namespace")
	}
	if Namespace == "" {
		log.Error("--namespace flag is not set and required")
		os.Exit(2)
	}

	log.Debug("namespace is " + viper.GetString("Namespace"))

}
