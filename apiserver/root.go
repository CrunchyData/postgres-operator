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
	"bufio"
	"errors"
	"flag"
	log "github.com/Sirupsen/logrus"
	crdclient "github.com/crunchydata/postgres-operator/client"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// pgouserPath ...
const pgouserPath = "/config/pgouser"

// RESTClient ...
var RESTClient *rest.RESTClient

// Clientset ...
var Clientset *kubernetes.Clientset

// DebugFlag is the debug flag value
var DebugFlag bool

// BasicAuth comes from the apiserver config
var BasicAuth bool

// Namespace comes from the apiserver config in this version
var Namespace string

// TreeTrunk is for debugging only in this context
const TreeTrunk = "└── "

// TreeBranch is for debugging only in this context
const TreeBranch = "├── "

// Credentials holds the BasicAuth credentials found in the config
var Credentials map[string]string

func init() {
	BasicAuth = true

	log.Infoln("apiserver starts")

	getCredentials()

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
	RESTClient, _, err = crdclient.NewClient(config)
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

	viper.SetConfigName("pgo")     // name of config file (without extension)
	viper.AddConfigPath(".")       // adding current directory as first search path
	viper.AddConfigPath("$HOME")   // adding home directory as second search path
	viper.AddConfigPath("/config") // adding /config directory as third search path
	viper.AutomaticEnv()           // read in environment variables that match

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
	tmp := viper.GetString("BasicAuth")
	if tmp == "" {
		BasicAuth = true
	} else {
		var err error
		BasicAuth, err = strconv.ParseBool(tmp)
		if err != nil {
			log.Error("BasicAuth config value is not valid")
			os.Exit(2)
		}
	}
	log.Infof("BasicAuth is %v\n", BasicAuth)

	log.Info("namespace is " + viper.GetString("Namespace"))

}

func file2lines(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
	}

	return lines
}

func parseUserMap(dat string) (string, string) {

	fields := strings.Split(strings.TrimSpace(dat), ":")
	log.Infof("%v\n", fields)
	log.Infof("username=[%s] password=[%s]\n", fields[0], fields[1])
	return fields[0], fields[1]
}

// getCredentials ...
func getCredentials() {
	var Username, Password string

	Credentials = make(map[string]string)

	lines := file2lines(pgouserPath)
	for _, v := range lines {
		Username, Password = parseUserMap(v)
		log.Debugf("username=%s password=%s\n", Username, Password)
		Credentials[Username] = Password
	}

}

func BasicAuthCheck(username, password string) bool {

	if BasicAuth == false {
		return true
	}

	value := Credentials[username]
	if value == "" {
		return false
	}

	if value != password {
		return false
	}

	return true
}

func Authn(where string, w http.ResponseWriter, r *http.Request) error {
	var err error
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	username, password, authOK := r.BasicAuth()
	log.Debugf("Authn Attempt %s username=[%s] password=[%s]\n", where, username, password)
	if authOK == false {
		http.Error(w, "Not authorized", 401)
		return errors.New("Not Authorized")
	}

	if !BasicAuthCheck(username, password) {
		log.Errorf("Authn Failed %s username=[%s] password=[%s]\n", where, username, password)
		http.Error(w, "Not authenticated in apiserver", 401)
		return errors.New("Not Authenticated")
	}
	log.Debugf("Authn Success %s username=[%s] password=[%s]\n", where, username, password)
	return err

}
