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

package pvc

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	//v1 "k8s.io/api/core/v1"
	"text/template"
	"time"
)

const PVC_PATH = "/operator-conf/pvc.json"
const PVC_SC_PATH = "/operator-conf/pvc-storageclass.json"

var PVCTemplate, PVCStorageClassTemplate *template.Template

type PVCTemplateFields struct {
	PVC_NAME        string
	PVC_ACCESS_MODE string
	PVC_SIZE        string
	STORAGE_CLASS   string
}

func init() {
	var err error
	var buf, buf2 []byte

	buf, err = ioutil.ReadFile(PVC_PATH)
	if err != nil {
		log.Error("error in pvc init" + err.Error())
		panic(err.Error())
	}
	PVCTemplate = template.Must(template.New("pvc template").Parse(string(buf)))

	buf2, err = ioutil.ReadFile(PVC_SC_PATH)
	if err != nil {
		log.Error("error in pvc storage class init" + err.Error())
		panic(err.Error())
	}
	PVCStorageClassTemplate = template.Must(template.New("pvc sc template").Parse(string(buf2)))
}

func CreatePVC(clientset *kubernetes.Clientset, name string, storageSpec *crv1.PgStorageSpec, namespace string) (string, error) {
	var pvcName string
	var err error

	switch storageSpec.StorageType {
	case "":
		log.Debug("StorageType is empty")
	case "emptydir":
		log.Debug("StorageType is emptydir")
	case "existing":
		log.Debug("StorageType is existing")
		pvcName = storageSpec.PvcName
	case "create", "dynamic":
		log.Debug("StorageType is create")
		pvcName = name + "-pvc"
		log.Debug("PVC_NAME=%s PVC_SIZE=%s PVC_ACCESS_MODE=%s\n",
			pvcName, storageSpec.PvcAccessMode, storageSpec.PvcSize)
		err = Create(clientset, pvcName, storageSpec.PvcAccessMode, storageSpec.PvcSize, storageSpec.StorageType, storageSpec.StorageClass, namespace)
		if err != nil {
			log.Error("error in pvc create " + err.Error())
			return pvcName, err
		}
		log.Info("created PVC =" + pvcName + " in namespace " + namespace)
	}

	return pvcName, err
}

func Create(clientset *kubernetes.Clientset, name string, accessMode string, pvcSize string, storageType string, storageClass string, namespace string) error {
	log.Debug("in createPVC")
	var doc2 bytes.Buffer
	var err error

	pvcFields := PVCTemplateFields{
		PVC_NAME:        name,
		PVC_ACCESS_MODE: accessMode,
		STORAGE_CLASS:   storageClass,
		PVC_SIZE:        pvcSize,
	}

	if storageType == "dynamic" {
		log.Debug("using dynamic PVC template")
		err = PVCStorageClassTemplate.Execute(&doc2, pvcFields)
	} else {
		err = PVCTemplate.Execute(&doc2, pvcFields)
	}
	if err != nil {
		log.Error("error in pvc create exec" + err.Error())
		return err
	}
	pvcDocString := doc2.String()
	log.Debug(pvcDocString)

	//template name is lspvc-pod.json
	//create lspvc pod
	newpvc := v1.PersistentVolumeClaim{}
	err = json.Unmarshal(doc2.Bytes(), &newpvc)
	if err != nil {
		log.Error("error unmarshalling json into PVC " + err.Error())
		return err
	}
	var result *v1.PersistentVolumeClaim
	result, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Create(&newpvc)
	if err != nil {
		log.Error("error creating pvc " + err.Error() + " in namespace " + namespace)
		return err
	}
	log.Debug("created PVC " + result.Name + " in namespace " + namespace)
	//TODO replace sleep with proper wait
	time.Sleep(3000 * time.Millisecond)
	return nil

}

func Delete(clientset *kubernetes.Clientset, name string, namespace string) error {
	log.Debug("in pvc.Delete")
	var err error

	var pvc *v1.PersistentVolumeClaim

	//see if the PVC exists
	options := meta_v1.GetOptions{}
	pvc, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(name, options)
	if err != nil {
		log.Info("\nPVC %s\n", name+" is not found, will not attempt delete")
		return nil
	} else {
		log.Info("\nPVC %s\n", pvc.Name+" is found")
		log.Info("%v\n", pvc)
		//if pgremove = true remove it
		if pvc.ObjectMeta.Labels["pgremove"] == "true" {
			log.Info("pgremove is true on this pvc")
			log.Debug("delete PVC " + name + " in namespace " + namespace)
			err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(name, &meta_v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting PVC " + name + err.Error() + " in namespace " + namespace)
				return err
			}
		}
	}

	return nil

}

func Exists(clientset *kubernetes.Clientset, name string, namespace string) bool {
	options := meta_v1.GetOptions{}
	_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(name, options)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		} else {
			log.Error("error getting PVC " + name + " " + err.Error() + " in namespace " + namespace)
		}
	}

	return true
}
