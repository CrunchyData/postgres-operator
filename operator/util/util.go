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

package util

import (
	"bytes"
	"encoding/json"
	//jsonpatch "github.com/evanphx/json-patch"
	//kjson "k8s.io/client-go/pkg/util/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"strconv"
	"text/template"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	//"k8s.io/client-go/pkg/api/meta"
	//"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

func CreateSecContext(FS_GROUP string, SUPP string) string {

	var sc bytes.Buffer
	var fsgroup = false
	var supp = false

	if FS_GROUP != "" {
		fsgroup = true
	}
	if SUPP != "" {
		supp = true
	}
	if fsgroup || supp {
		sc.WriteString("\"securityContext\": {\n")
	}
	if fsgroup {
		sc.WriteString("\t \"fsGroup\": " + FS_GROUP)
		if fsgroup && supp {
			sc.WriteString(",")
		}
		sc.WriteString("\n")
	}

	if supp {
		sc.WriteString("\t \"supplementalGroups\": [" + SUPP + "]\n")
	}

	//closing of securityContext
	if fsgroup || supp {
		sc.WriteString("},")
	}

	return sc.String()
}

func LoadTemplate(path string) *template.Template {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("error loading template" + err.Error())
		panic(err.Error())
	}
	return template.Must(template.New(path).Parse(string(buf)))

}

type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func Patch(tprclient *rest.RESTClient, path string, value string, resource string, name string, namespace string) error {

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = path
	things[0].Value = value

	patchBytes, err4 := json.Marshal(things)
	if err4 != nil {
		log.Error("error in converting patch " + err4.Error())
	}
	log.Debug(string(patchBytes))

	_, err6 := tprclient.Patch(api.JSONPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}

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

	_, err = clientset.Deployments(namespace).Patch(name, api.JSONPatchType, patchBytes, "")
	if err != nil {
		log.Error("error patching deployment " + err.Error())
	}

	return err

}

func CreatePVCSnippet(storageType string, PVC_NAME string) string {

	var sc bytes.Buffer

	if storageType != "emptydir" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + PVC_NAME + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
}

func CreateBackupPVCSnippet(BACKUP_PVC_NAME string) string {

	var sc bytes.Buffer

	if BACKUP_PVC_NAME != "" {
		sc.WriteString("\"persistentVolumeClaim\": {\n")
		sc.WriteString("\t \"claimName\": \"" + BACKUP_PVC_NAME + "\"")
		sc.WriteString("\n")
	} else {
		sc.WriteString("\"emptyDir\": {")
		sc.WriteString("\n")
	}

	sc.WriteString("}")

	return sc.String()
}

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

	_, err = clientset.Deployments(namespace).Patch(deploymentName, api.JSONPatchType, patchBytes)
	if err != nil {
		log.Error("error creating master Deployment " + err.Error())
		return err
	}
	log.Debug("replica count patch succeeded")
	return err
}

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

/**
func UpdateDeploymentLabels(clientset *kubernetes.Clientset, depName, namespace string, newLabels map[string]string) error {

	var err error
	var deployment *v1beta1.Deployment

	//get the deployment
	deployment, err = clientset.Deployments(namespace).Get(depName)
	if err != nil {
		return err
	}
	log.Debug("got the deployment" + deployment.Name)

	var patchBytes, newData, origData []byte
	origData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	accessor, err2 := meta.Accessor(deployment)
	if err2 != nil {
		return err2
	}

	objLabels := accessor.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	//update the deployment labels
	for key, value := range newLabels {
		objLabels[key] = value
	}
	log.Debugf("updated labels are %v\n", objLabels)

	accessor.SetLabels(objLabels)

	newData, err = json.Marshal(deployment)
	if err != nil {
		return err
	}

	patchBytes, err = jsonpatch.CreateMergePatch(origData, newData)
	createdPatch := err == nil
	if err != nil {
		return err
	}
	if createdPatch {
		log.Debug("created merge patch")
	}

	_, err = clientset.Deployments(namespace).Patch(depName, api.MergePatchType, patchBytes, "")
	if err != nil {
		log.Debug("error patching deployment " + err.Error())
	}
	return err

}
*/
