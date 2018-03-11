package util

/*
 Copyright2017-2018 Data Solutions, Inc.
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
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	jsonpatch "github.com/evanphx/json-patch"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const GLOBAL_CUSTOM_CONFIGMAP = "pgo-custom-pg-config"

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

// LoadTemplate will load a JSON template from a path
func LoadTemplate(path string) *template.Template {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("error loading template" + err.Error())
		panic(err.Error())
	}
	return template.Must(template.New(path).Parse(string(buf)))

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
func GetLabels(name, clustername string, clone, replica bool) string {
	var output string
	if clone {
		output += fmt.Sprintf("\"clone\": \"%s\",\n", "true")
	}
	if replica {
		output += fmt.Sprintf("\"replica\": \"%s\",\n", "true")
	}
	output += fmt.Sprintf("\"name\": \"%s\",\n", name)
	output += fmt.Sprintf("\"pg-cluster\": \"%s\"\n", clustername)
	return output
}

// GetLabelsFromMap ...
func GetLabelsFromMap(labels map[string]string) string {
	var output string

	mapLen := len(labels)
	i := 1
	for key, value := range labels {
		//fmt.Println("Key:", key, "Value:", value)
		if i < mapLen {
			output += fmt.Sprintf("\"" + key + "\": \"" + value + "\",")
		} else {
			output += fmt.Sprintf("\"" + key + "\": \"" + value + "\"")
		}
		i++
	}
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
func RunPsql(password string, hostip string, sqlstring string) {

	log.Debug("RunPsql hostip=[" + hostip + "] sql=[" + sqlstring + "]")
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
		return
	}

	log.Debugf("runpsql output [%s]\n", out.String()[0:20])
}

// GetSecretPassword ...
func GetSecretPassword(clientset *kubernetes.Clientset, db, suffix, Namespace string) (string, error) {

	var err error

	lo := meta_v1.ListOptions{LabelSelector: "pg-database=" + db}
	secrets, err := clientset.Core().Secrets(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of secrets" + err.Error())
		return "", err
	}

	log.Debug("secrets for " + db)
	secretName := db + suffix
	for _, s := range secrets.Items {
		log.Debug("secret : " + s.ObjectMeta.Name)
		if s.ObjectMeta.Name == secretName {
			log.Debug("pgprimary password found")
			return string(s.Data["password"][:]), err
		}
	}

	log.Error("primary secret not found for " + db)
	return "", errors.New("primary secret not found for " + db)

}

// GetStorageSpec ...
func GetStorageSpec(cfg *viper.Viper) crv1.PgStorageSpec {
	storage := crv1.PgStorageSpec{}
	storage.StorageClass = cfg.GetString("StorageClass")
	storage.AccessMode = cfg.GetString("AccessMode")
	storage.Size = cfg.GetString("Size")
	storage.StorageType = cfg.GetString("StorageType")
	storage.Fsgroup = cfg.GetString("Fsgroup")
	storage.SupplementalGroups = cfg.GetString("SupplementalGroups")
	return storage

}
