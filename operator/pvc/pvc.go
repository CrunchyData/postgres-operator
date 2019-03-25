package pvc

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
	"encoding/json"
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"
	"time"
)

type matchLabelsTemplateFields struct {
	Key   string
	Value string
}

// TemplateFields ...
type TemplateFields struct {
	Name         string
	AccessMode   string
	ClusterName  string
	Size         string
	StorageClass string
	MatchLabels  string
}

// CreatePVC create a pvc
func CreatePVC(clientset *kubernetes.Clientset, storageSpec *crv1.PgStorageSpec, pvcName, clusterName, namespace string) (string, error) {
	var err error

	switch storageSpec.StorageType {
	case "":
		log.Debug("StorageType is empty")
	case "emptydir":
		log.Debug("StorageType is emptydir")
	case "existing":
		log.Debug("StorageType is existing")
		pvcName = storageSpec.Name
	case "create", "dynamic":
		log.Debug("StorageType is create")
		log.Debugf("pvcname=%s storagespec=%v", pvcName, storageSpec)
		err = Create(clientset, pvcName, clusterName, storageSpec, namespace)
		if err != nil {
			log.Error("error in pvc create " + err.Error())
			return pvcName, err
		}
		log.Info("created PVC =" + pvcName + " in namespace " + namespace)
	}

	return pvcName, err
}

// Create a pvc
func Create(clientset *kubernetes.Clientset, name, clusterName string, storageSpec *crv1.PgStorageSpec, namespace string) error {
	log.Debug("in createPVC")
	var doc2 bytes.Buffer
	var err error

	pvcFields := TemplateFields{
		Name:         name,
		AccessMode:   storageSpec.AccessMode,
		StorageClass: storageSpec.StorageClass,
		ClusterName:  clusterName,
		Size:         storageSpec.Size,
		MatchLabels:  storageSpec.MatchLabels,
	}

	if storageSpec.StorageType == "dynamic" {
		log.Debug("using dynamic PVC template")
		err = config.PVCStorageClassTemplate.Execute(&doc2, pvcFields)
		if operator.CRUNCHY_DEBUG {
			config.PVCStorageClassTemplate.Execute(os.Stdout, pvcFields)
		}
	} else {
		log.Debugf("matchlabels from spec is [%s]", storageSpec.MatchLabels)
		if storageSpec.MatchLabels != "" {
			arr := strings.Split(storageSpec.MatchLabels, "=")
			if len(arr) != 2 {
				log.Error("%s MatchLabels is not formatted correctly", storageSpec.MatchLabels)
				return errors.New("match labels is not formatted correctly")
			}
			pvcFields.MatchLabels = getMatchLabels(arr[0], arr[1])
			log.Debugf("matchlabels constructed is %s", pvcFields.MatchLabels)
		}

		err = config.PVCTemplate.Execute(&doc2, pvcFields)
		if operator.CRUNCHY_DEBUG {
			config.PVCTemplate.Execute(os.Stdout, pvcFields)
		}
	}
	if err != nil {
		log.Error("error in pvc create exec" + err.Error())
		return err
	}

	//template name is lspvc-pod.json
	//create lspvc pod
	newpvc := v1.PersistentVolumeClaim{}
	err = json.Unmarshal(doc2.Bytes(), &newpvc)
	if err != nil {
		log.Error("error unmarshalling json into PVC " + err.Error())
		return err
	}

	err = kubeapi.CreatePVC(clientset, &newpvc, namespace)
	if err != nil {
		return err
	}

	//TODO replace sleep with proper wait
	time.Sleep(3000 * time.Millisecond)
	return nil

}

// Delete a pvc
func Delete(clientset *kubernetes.Clientset, name string, namespace string) error {
	var err error
	var found bool
	var pvc *v1.PersistentVolumeClaim

	//see if the PVC exists
	pvc, found, err = kubeapi.GetPVC(clientset, name, namespace)
	if err != nil || !found {
		log.Infof("\nPVC %s\n", name+" is not found, will not attempt delete")
		return nil
	}

	log.Debugf("PVC %s is found", pvc.Name)

	if pvc.ObjectMeta.Labels[config.LABEL_PGREMOVE] == "true" {
		log.Debugf("delete PVC %s in namespace %s", name, namespace)
		err = kubeapi.DeletePVC(clientset, name, namespace)
		if err != nil {
			return err
		}
	}

	return nil

}

// Exists test to see if pvc exists
func Exists(clientset *kubernetes.Clientset, name string, namespace string) bool {
	_, found, _ := kubeapi.GetPVC(clientset, name, namespace)

	return found
}

func getMatchLabels(key, value string) string {

	matchLabelsTemplateFields := matchLabelsTemplateFields{}
	matchLabelsTemplateFields.Key = key
	matchLabelsTemplateFields.Value = value

	var doc bytes.Buffer
	err := config.PVCMatchLabelsTemplate.Execute(&doc, matchLabelsTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	return doc.String()

}
