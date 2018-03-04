package apiserver

/*
Copyright 2018 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/util"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// MetricsFlag if set to true will cause crunchy-collect to be added into new clusters
var MetricsFlag bool

// AuditFlag if set to true will cause auditing to occur in the logs
var AuditFlag bool

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

type CredentialDetail struct {
	Username string
	Password string
	Role     string
}

// Credentials holds the BasicAuth credentials found in the config
var Credentials map[string]CredentialDetail

var StorageMap map[string]interface{}
var ContainerResourcesMap map[string]interface{}

func init() {
	BasicAuth = true
	MetricsFlag = false
	AuditFlag = false

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

	AuditFlag = viper.GetBool("Pgo.Audit")
	if AuditFlag {
		log.Info("audit flag is set to true")
	}

	MetricsFlag = viper.GetBool("Pgo.Metrics")
	if MetricsFlag {
		log.Info("metrics flag is set to true")
	}

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

	StorageMap = viper.GetStringMap("Storage")

	if !validStorageSettings() {
		log.Error("Storage Settings are not defined correctly, can't continue")
		os.Exit(2)
	}

	ContainerResourcesMap = viper.GetStringMap("ContainerResources")

	if !validContainerResourcesSettings() {
		log.Error("Container Resources settings are not defined correctly, can't continue")
		os.Exit(2)
	}

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

func parseUserMap(dat string) CredentialDetail {

	creds := CredentialDetail{}

	fields := strings.Split(strings.TrimSpace(dat), ":")
	//log.Infof("%v\n", fields)
	//log.Infof("username=[%s] password=[%s] role=[%s]\n", fields[0], fields[1], fields[2])
	creds.Username = fields[0]
	creds.Password = fields[1]
	creds.Role = fields[2]
	return creds
}

// getCredentials ...
func getCredentials() {

	Credentials = make(map[string]CredentialDetail)

	lines := file2lines(pgouserPath)
	for _, v := range lines {
		creds := parseUserMap(v)
		Credentials[creds.Username] = creds
	}

}

func BasicAuthCheck(username, password string) bool {

	if BasicAuth == false {
		return true
	}

	value := Credentials[username]
	if (CredentialDetail{}) == value {
		return false
	}

	if value.Password != password {
		return false
	}

	return true
}

func BasicAuthzCheck(username, perm string) bool {

	return true
}

func Authn(perm string, w http.ResponseWriter, r *http.Request) error {
	var err error
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	username, password, authOK := r.BasicAuth()
	if AuditFlag {
		log.Infof("[audit] %s username=[%s] method=[%s]\n", perm, username, r.Method)
	}

	log.Debugf("Authn Attempt %s username=[%s]\n", perm, username)
	if authOK == false {
		http.Error(w, "Not authorized", 401)
		return errors.New("Not Authorized")
	}

	if !BasicAuthCheck(username, password) {
		log.Errorf("Authn Failed %s username=[%s]\n", perm, username)
		http.Error(w, "Not authenticated in apiserver", 401)
		return errors.New("Not Authenticated")
	}

	if !BasicAuthzCheck(username, perm) {
		log.Errorf("Authn Failed %s username=[%s]\n", perm, username)
		http.Error(w, "Not authorized for this apiserver action", 401)
		return errors.New("Not Authorized for this apiserver action")
	}

	log.Debug("Authn Success")
	return err

}

func validContainerResourcesSettings() bool {
	log.Infof("ContainerResources has %d definitions \n", len(ContainerResourcesMap))

	//validate any Container Resources in pgo.yaml for correct formats
	//log.Infof("%v is the ContainerResourcesMap\n", ContainerResourcesMap)
	if !IsValidContainerResourceValues() {
		return false
	}

	drs := viper.GetString("DefaultContainerResource")
	if drs == "" {
		log.Info("DefaultContainerResources was not specified in pgo.yaml, so no container resources will be specified")
		return true
	}

	//validate the DefaultContainerResource value
	if IsValidContainerResource(drs) {
		log.Info(drs + " is valid")
	} else {
		log.Error(drs + " is NOT valid")
		return false
	}

	return true

}

func validStorageSettings() bool {
	log.Infof("Storage has %d definitions\n", len(StorageMap))

	ps := viper.GetString("PrimaryStorage")
	if IsValidStorageName(ps) {
		log.Info(ps + " is valid")
	} else {
		log.Error(ps + " is NOT valid")
		return false
	}
	rs := viper.GetString("ReplicaStorage")
	if IsValidStorageName(rs) {
		log.Info(rs + " is valid")
	} else {
		log.Error(rs + " is NOT valid")
		return false
	}
	bs := viper.GetString("BackupStorage")
	if IsValidStorageName(bs) {
		log.Info(bs + " is valid")
	} else {
		log.Error(bs + " is NOT valid")
		return false
	}

	return true

}

func IsValidContainerResource(name string) bool {
	_, ok := ContainerResourcesMap[name]
	return ok
}

func IsValidStorageName(name string) bool {
	_, ok := StorageMap[name]
	return ok
}

// IsValidNodeName returns true or false if
// a node is valid, returns a string that
// describes the not valid condition, and
// lastly a string of all valid nodes found
func IsValidNodeName(nodeName string) (bool, string, string) {

	var err error
	found := false
	allNodes := ""

	lo := meta_v1.ListOptions{}
	nodes, err := Clientset.CoreV1().Nodes().List(lo)
	if err != nil {
		log.Error(err)
		return false, err.Error(), allNodes
	}

	for _, node := range nodes.Items {
		log.Infof("%v\n", node)
		if node.Name == nodeName {
			found = true
		}
		allNodes += node.Name + " "
	}

	if found == false {
		return false, "not found", allNodes
	}

	return true, "", allNodes
}

func IsValidContainerResourceValues() bool {

	var err error

	for k, v := range ContainerResourcesMap {
		log.Infof("Container Resources %s [%v]\n", k, v)
		resources := util.GetContainerResources(viper.Sub("ContainerResources." + k))
		_, err = resource.ParseQuantity(resources.RequestsMemory)
		if err != nil {
			log.Errorf("%s.RequestsMemory value invalid format\n", k)
			return false
		}
		_, err = resource.ParseQuantity(resources.RequestsCPU)
		if err != nil {
			log.Errorf("%s.RequestsCPU value invalid format\n", k)
			return false
		}
		_, err = resource.ParseQuantity(resources.LimitsMemory)
		if err != nil {
			log.Errorf("%s.LimitsMemory value invalid format\n", k)
			return false
		}
		_, err = resource.ParseQuantity(resources.LimitsCPU)
		if err != nil {
			log.Errorf("%s.LimitsCPU value invalid format\n", k)
			return false
		}
	}
	return true
}
