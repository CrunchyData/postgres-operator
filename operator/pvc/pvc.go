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
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"text/template"
	"time"
)

const PVC_PATH = "/pgconf/postgres-operator/pvc.json"

var PVCTemplate *template.Template

type PVCTemplateFields struct {
	PVC_NAME        string
	PVC_ACCESS_MODE string
	PVC_SIZE        string
}

func init() {
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(PVC_PATH)
	if err != nil {
		log.Error(err.Error())
		panic(err.Error())
	}
	PVCTemplate = template.Must(template.New("pvc template").Parse(string(buf)))
}

func Create(clientset *kubernetes.Clientset, name string, accessMode string, pvcSize string) error {
	log.Debug("in createPVC")
	var doc2 bytes.Buffer
	var err error

	pvcFields := PVCTemplateFields{
		PVC_NAME:        name,
		PVC_ACCESS_MODE: accessMode,
		PVC_SIZE:        pvcSize,
	}

	err = PVCTemplate.Execute(&doc2, pvcFields)
	if err != nil {
		log.Error(err.Error())
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
	result, err = clientset.Core().PersistentVolumeClaims(v1.NamespaceDefault).Create(&newpvc)
	if err != nil {
		log.Error("error creating pvc " + err.Error())
		return err
	}
	log.Debug("created PVC " + result.Name)
	time.Sleep(3000 * time.Millisecond)
	return nil

}

func Delete(clientset *kubernetes.Clientset, name string) error {
	log.Debug("in pvc.Delete")
	var err error

	var pvc *v1.PersistentVolumeClaim

	//see if the PVC exists
	pvc, err = clientset.Core().PersistentVolumeClaims(v1.NamespaceDefault).Get(name)
	if err != nil {
		log.Info("\nPVC %s\n", name+" is not found, will not attempt delete")
		return nil
	} else {
		log.Info("\nPVC %s\n", pvc.Name+" is found")
		log.Info("%v\n", pvc)
		//if pgremove = true remove it
		if pvc.ObjectMeta.Labels["pgremove"] == "true" {
			log.Info("pgremove is true on this pvc")
			log.Debug("delete PVC " + name)
			err = clientset.Core().PersistentVolumeClaims(v1.NamespaceDefault).Delete(name, &v1.DeleteOptions{})
			if err != nil {
				log.Error("error deleting PVC " + name + err.Error())
				return err
			}
		}
	}

	return nil

}
