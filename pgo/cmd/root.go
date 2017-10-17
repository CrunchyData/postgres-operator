package cmd

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
	log "github.com/Sirupsen/logrus"
	"os"
	"strconv"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/client"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/api/core/v1"
	//apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RED for color highlights
var RED func(a ...interface{}) string

// GREEN for color highlights..
var GREEN func(a ...interface{}) string

var cfgFile string

// KubeconfigPath is the kubeconfig flag value
var KubeconfigPath string

// Labelselector is the selector flag value
var Labelselector string

// DebugFlag is the debug flag value
var DebugFlag bool

// Namespace is the namespace flag value
var Namespace string

// Selector is the selector flag value
var Selector string

// DryRun is the dry-run flag value
var DryRun bool

// RestClient is the global connection to the Kube REST api
var RestClient *rest.RESTClient

// Clientset is the global Kube API connection
var Clientset *kubernetes.Clientset

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pgo",
	Short: "The pgo command line interface.",
	Long: `The pgo command line interface lets you
create and manage PostgreSQL clusters.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Debug(err.Error())
		os.Exit(-1)
	}

}

func init() {

	GREEN = color.New(color.FgGreen).SprintFunc()
	RED = color.New(color.FgRed).SprintFunc()

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pgo.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	RootCmd.PersistentFlags().StringVar(&KubeconfigPath, "kubeconfig", "", "kube config file")
	RootCmd.PersistentFlags().StringVar(&Namespace, "namespace", "", "kube namespace to work in (default is default)")
	RootCmd.PersistentFlags().StringVar(&Labelselector, "selector", "", "label selector string")
	RootCmd.PersistentFlags().BoolVar(&DebugFlag, "debug", false, "enable debug with true")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

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

	if KubeconfigPath == "" {
		KubeconfigPath = viper.GetString("Kubeconfig")
	}
	if KubeconfigPath == "" {
		log.Error("--kubeconfig flag is not set and required")
		os.Exit(2)
	}

	log.Debug("kubeconfig path is " + viper.GetString("Kubeconfig"))

	if Namespace == "" {
		Namespace = viper.GetString("Namespace")
	}
	if Namespace == "" {
		log.Error("--namespace flag is not set and required")
		os.Exit(2)
	}

	log.Debug("namespace is " + viper.GetString("Namespace"))

	validateConfig()

	ConnectToKube()

	/**
	file, err2 := os.Create("/tmp/pgo-bash-completion.out")
	if err2 != nil {
		log.Error(err2.Error())
	}
	defer file.Close()
	RootCmd.GenBashCompletion(file)
	*/

}

// validateConfig ...
func validateConfig() {
	switch viper.GetString("PrimaryStorage.AccessMode") {
	case string(v1.ReadWriteOnce), string(v1.ReadWriteMany), string(v1.ReadOnlyMany):
	default:
		log.Error("invalid PrimaryStorage.AccessMode specified")
		os.Exit(2)
	}
	switch viper.GetString("ReplicaStorage.AccessMode") {
	case string(v1.ReadWriteOnce), string(v1.ReadWriteMany), string(v1.ReadOnlyMany):
	default:
		log.Error("invalid ReplicaStorage.AccessMode specified")
		os.Exit(2)
	}
	switch viper.GetString("PrimaryStorage.StorageType") {
	case crv1.StorageExisting, crv1.StorageCreate, crv1.StorageEmptydir, crv1.StorageDynamic:
	default:
		log.Error("invalid PrimaryStorage.StorageType specified")
		os.Exit(2)
	}
	switch viper.GetString("ReplicaStorage.StorageType") {
	case crv1.StorageExisting, crv1.StorageCreate, crv1.StorageEmptydir, crv1.StorageDynamic:
	default:
		log.Error("invalid ReplicaStorage.StorageType specified")
		os.Exit(2)
	}

	rep := viper.GetString("Cluster.Replicas")
	if rep != "" {
		_, err := strconv.Atoi(rep)
		if err != nil {
			log.Error("Cluster.Replicas not a valid integer")
			os.Exit(2)
		}
	}
	port := viper.GetString("Cluster.Port")
	if port != "" {
		_, err := strconv.Atoi(port)
		if err != nil {
			log.Error("Cluster.Port not a valid integer")
			os.Exit(2)
		}
	}
	strategy := viper.GetString("Cluster.Strategy")
	if strategy != "" {
		_, err := strconv.Atoi(strategy)
		if err != nil {
			log.Error("Cluster.Strategy not a valid integer")
			os.Exit(2)
		}
	}

	pvcsize := viper.GetString("PrimaryStorage.Size")
	if pvcsize != "" {
		_, err := resource.ParseQuantity(pvcsize)
		if err != nil {
			log.Error("PrimaryStorage.Size not a valid quantity")
			os.Exit(2)
		}
	}
	pvcsize = viper.GetString("ReplicaStorage.Size")
	if pvcsize != "" {
		_, err := resource.ParseQuantity(pvcsize)
		if err != nil {
			log.Error("ReplicaStorage.Size not a valid quantity")
			os.Exit(2)
		}
	}
	passwordAge := viper.GetString("Cluster.PasswordAgeDays")
	if passwordAge != "" {
		_, err := resource.ParseQuantity(passwordAge)
		if err != nil {
			log.Error("Cluster.PasswordAge not a valid quantity")
			os.Exit(2)
		}
	}
	passwordLen := viper.GetString("Cluster.PasswordLength")
	if passwordLen != "" {
		_, err := resource.ParseQuantity(passwordLen)
		if err != nil {
			log.Error("Cluster.PasswordLength not a valid quantity")
			os.Exit(2)
		}
	}

}

// ConnectToKube establish a connection to the Kube
func ConnectToKube() {

	config, err := buildConfig(KubeconfigPath)
	if err != nil {
		panic(err)
	}

	//Clientset, err = apiextensionsclient.NewForConfig(config)
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// make a new config for our extension's API group, using the first config as a baseline
	RestClient, _, err = client.NewClient(config)
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
