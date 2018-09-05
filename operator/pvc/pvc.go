package pvc

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"github.com/crunchydata/postgres-operator/util"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"time"
)

type matchLabelsTemplateFields struct {
	Name string
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
		log.Debugf("pvcname=%s storagespec=%v\n", pvcName, storageSpec)
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
		MatchLabels:  "",
	}

	if storageSpec.StorageType == "dynamic" {
		log.Debug("using dynamic PVC template")
		err = operator.PVCStorageClassTemplate.Execute(&doc2, pvcFields)
		if operator.CRUNCHY_DEBUG {
			operator.PVCStorageClassTemplate.Execute(os.Stdout, pvcFields)
		}
	} else {
		log.Debug("matchlabels from spec is [" + storageSpec.MatchLabels + "]")
		if storageSpec.MatchLabels != "" {
			pvcFields.MatchLabels = getMatchLabels(clusterName)
			log.Debug("matchlabels constructed is " + pvcFields.MatchLabels)
		}

		err = operator.PVCTemplate.Execute(&doc2, pvcFields)
		if operator.CRUNCHY_DEBUG {
			operator.PVCTemplate.Execute(os.Stdout, pvcFields)
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

	log.Debugf("\nPVC %s\n", pvc.Name+" is found")

	if pvc.ObjectMeta.Labels[util.LABEL_PGREMOVE] == "true" {
		log.Debug("delete PVC " + name + " in namespace " + namespace)
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

func getMatchLabels(name string) string {

	matchLabelsTemplateFields := matchLabelsTemplateFields{}
	matchLabelsTemplateFields.Name = name

	var doc bytes.Buffer
	err := operator.PVCMatchLabelsTemplate.Execute(&doc, matchLabelsTemplateFields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	return doc.String()

}
