package util

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())

}

// CreateSecContext will generate the JSON security context fragment
// for a storage type
func CreateSecContext(fsGroup string, suppGroup string) string {

	var sc bytes.Buffer
	var fsgroup = false
	var supp = false

	if fsGroup != "" {
		fsgroup = true
	}
	if suppGroup != "" {
		supp = true
	}
	if fsgroup || supp {
		sc.WriteString("\"securityContext\": {\n")
	}
	if fsgroup {
		sc.WriteString("\t \"fsGroup\": " + fsGroup)
		if fsgroup && supp {
			sc.WriteString(",")
		}
		sc.WriteString("\n")
	}

	if supp {
		sc.WriteString("\t \"supplementalGroups\": [" + suppGroup + "]\n")
	}

	//closing of securityContext
	if fsgroup || supp {
		sc.WriteString("},")
	}

	return sc.String()
}

// ThingSpec is a json patch structure
type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// Patch will patch a particular resource
func Patch(restclient *rest.RESTClient, path string, value string, resource string, name string, namespace string) error {

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = path
	things[0].Value = value

	patchBytes, err4 := json.Marshal(things)
	if err4 != nil {
		log.Error("error in converting patch " + err4.Error())
	}
	log.Debug(string(patchBytes))

	_, err6 := restclient.Patch(types.JSONPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

// DrainDeployment will drain a deployment to 0 pods
func DrainDeployment(clientset *kubernetes.Clientset, name string, namespace string) error {

	var err error
	var patchBytes []byte

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = "/spec/replicas"
	things[0].Value = "0"

	patchBytes, err = json.Marshal(things)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
	}
	log.Debug(string(patchBytes))

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(name, types.JSONPatchType, patchBytes, "")
	if err != nil {
		log.Error("error patching deployment " + err.Error())
	}

	return err

}

// CreatePVCSnippet generates the PVC json snippet
func CreatePVCSnippet(storageType string, PVCName string) string {

	var sc bytes.Buffer

	if storageType != "emptydir" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + PVCName + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
}

// CreateBackupPVCSnippet generates the PVC definition fragment
func CreateBackupPVCSnippet(backupPVCName string) string {

	var sc bytes.Buffer

	if backupPVCName != "" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + backupPVCName + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
}

// ScaleDeployment will increase the number of pods in a deployment
func ScaleDeployment(clientset *kubernetes.Clientset, deploymentName, namespace string, replicaCount int) error {
	var err error

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = "/spec/replicas"
	things[0].Value = strconv.Itoa(replicaCount)

	var patchBytes []byte
	patchBytes, err = json.Marshal(things)
	if err != nil {
		log.Error("error in converting patch " + err.Error())
		return err
	}
	log.Debug(string(patchBytes))

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Patch(deploymentName, types.JSONPatchType, patchBytes)
	if err != nil {
		log.Error("error creating primary Deployment " + err.Error())
		return err
	}
	log.Debug("replica count patch succeeded")
	return err
}

// GetLabels ...
func GetLabels(name, clustername string, replica bool) string {
	var output string
	if replica {
		output += fmt.Sprintf("\"primary\": \"%s\",\n", "false")
	}
	output += fmt.Sprintf("\"name\": \"%s\",\n", name)
	output += fmt.Sprintf("\"pg-cluster\": \"%s\"\n", clustername)
	return output
}

// PatchClusterCRD patches the pgcluster CRD
func PatchClusterCRD(restclient *rest.RESTClient, labelMap map[string]string, oldCrd *crv1.Pgcluster, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	if oldCrd.ObjectMeta.Labels == nil {
		oldCrd.ObjectMeta.Labels = make(map[string]string)
	}
	for k, v := range labelMap {
		oldCrd.ObjectMeta.Labels[k] = v
	}

	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCrd)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgclusterResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

// RunPsql runs a psql statement
func RunPsql(password string, hostip string, sqlstring string) error {

	log.Debugf("RunPsql hostip=[%s] sql=[%s]", hostip, sqlstring)
	cmd := exec.Command("runpsql.sh", password, hostip)

	cmd.Stdin = strings.NewReader(sqlstring)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Error("error in run cmd " + err.Error())
		log.Error(out.String())
		log.Error(stderr.String())
		return err
	}

	if stderr.String() != "" {
		log.Errorf("stderror in run cmd %s", stderr.String())
		log.Error(out.String())
		return errors.New("SQL error: " + stderr.String())
	}

	log.Debugf("runpsql output [%s]", out.String()[0:20])
	return err
}

// GetSecretPassword ...
func GetSecretPassword(clientset *kubernetes.Clientset, db, suffix, Namespace string) (string, error) {

	var err error

	selector := "pg-cluster=" + db
	secrets, err := kubeapi.GetSecrets(clientset, selector, Namespace)
	if err != nil {
		return "", err
	}

	log.Debugf("secrets for %s", db)
	secretName := db + suffix
	for _, s := range secrets.Items {
		log.Debugf("secret : %s", s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretName {
			log.Debug("pgprimary password found")
			return string(s.Data["password"][:]), err
		}
	}

	log.Error("primary secret not found for " + db)
	return "", errors.New("primary secret not found for " + db)

}

// RandStringBytesRmndr ...
func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// CreateBackrestPVCSnippet
func CreateBackrestPVCSnippet(backRestPVCName string) string {

	var sc bytes.Buffer

	if backRestPVCName != "" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + backRestPVCName + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\"medium\": \"Memory\"")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
}

// Generates an Md5Hash
func GetMD5HashForAuthFile(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// NewClient gets a REST connection to Kube
func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := crv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &crv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, scheme, nil
}

func ValidateNamespaces(clientset *kubernetes.Clientset) error {
	raw := os.Getenv("NAMESPACE")

	//the case of 'all' namespaces
	if raw == "" {
		return nil
	}

	allFound := false

	nsList := strings.Split(raw, ",")

	//check for the invalid case where a user has NAMESPACE=demo1,,demo2
	if len(nsList) > 1 {
		for i := 0; i < len(nsList); i++ {
			if nsList[i] == "" {
				allFound = true
			}
		}
	}

	if allFound && len(nsList) > 1 {
		return errors.New("'' (empty string), found within the NAMESPACE environment variable along with other namespaces, this is not an accepted format")
	}

	//check for the case of a non-existing namespace being used
	for i := 0; i < len(nsList); i++ {
		_, found, _ := kubeapi.GetNamespace(clientset, nsList[i])
		if !found {
			return errors.New("NAMESPACE environment variable contains a namespace of " + nsList[i] + " but that is not found on this kube system")
		}
	}

	return nil

}

func GetNamespaces() []string {
	raw := os.Getenv("NAMESPACE")

	//the case of 'all' namespaces
	if raw == "" {
		return []string{""}
	}

	return strings.Split(raw, ",")

}

func WatchingNamespace(clientset *kubernetes.Clientset, requestedNS string) bool {

	log.Debugf("WatchingNamespace [%s]", requestedNS)

	nsList := GetNamespaces()

	//handle the case where we are watching all namespaces but
	//the user might enter an invalid namespace not on the kube
	if nsList[0] == "" {
		_, found, _ := kubeapi.GetNamespace(clientset, requestedNS)
		if !found {
			return false
		} else {
			return true
		}
	}

	for i := 0; i < len(nsList); i++ {
		if nsList[i] == requestedNS {
			return true
		}
	}

	return false
}
