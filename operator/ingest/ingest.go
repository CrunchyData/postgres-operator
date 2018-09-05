package ingest

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
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

type ingestTemplateFields struct {
	Name            string
	PvcName         string
	SecurityContext string
	Namespace       string
	WatchDir        string
	DBHost          string
	DBPort          string
	DBName          string
	DBSecret        string
	DBTable         string
	DBColumn        string
	COImageTag      string
	COImagePrefix   string
	MaxJobs         int
}

// CreateIngest ...
func CreateIngest(namespace string, clientset *kubernetes.Clientset, client *rest.RESTClient, i *crv1.Pgingest) {

	//create the ingest deployment

	jobFields := ingestTemplateFields{
		Name:            i.Spec.Name,
		PvcName:         i.Spec.PVCName,
		SecurityContext: "",
		Namespace:       namespace,
		WatchDir:        i.Spec.WatchDir,
		DBHost:          i.Spec.DBHost,
		DBPort:          i.Spec.DBPort,
		DBName:          i.Spec.DBName,
		DBSecret:        i.Spec.DBSecret,
		DBTable:         i.Spec.DBTable,
		DBColumn:        i.Spec.DBColumn,
		MaxJobs:         i.Spec.MaxJobs,
		COImageTag:      operator.Pgo.Pgo.COImageTag,
		COImagePrefix:   operator.Pgo.Pgo.COImagePrefix,
	}

	var doc2 bytes.Buffer
	err := operator.IngestjobTemplate.Execute(&doc2, jobFields)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if operator.CRUNCHY_DEBUG {
		operator.IngestjobTemplate.Execute(os.Stdout, jobFields)
	}

	deployment := v1beta1.Deployment{}
	err = json.Unmarshal(doc2.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling ingest json into Deployment " + err.Error())
		return
	}

	err = kubeapi.CreateDeployment(clientset, &deployment, namespace)
	if err != nil {
		log.Error("error creating ingest Deployment " + err.Error())
		return
	}

}

// Delete ingest
func Delete(clientset *kubernetes.Clientset, name string, namespace string) error {
	log.Debug("in ingest.Delete")
	var err error

	err = kubeapi.DeleteDeployment(clientset, name, namespace)
	if err != nil {
		log.Error("error deleting replica Deployment " + err.Error())
		return err
	}

	return err

}
