// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"github.com/crunchydata/postgres-operator/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api/v1"
	"k8s.io/api/core/v1"
	"text/template"
)

// ServiceTemplate1 ...
var ServiceTemplate1 *template.Template

func init() {
	ServiceTemplate1 = util.LoadTemplate("/operator-conf/cluster-service-1.json")

}

// CreateService ...
func CreateService(clientset *kubernetes.Clientset, fields *ServiceTemplateFields, namespace string) error {
	var err error
	var replicaServiceDoc bytes.Buffer
	var replicaServiceResult *v1.Service

	//create the replica service if it doesn't exist
	_, err = clientset.CoreV1().Services(namespace).Get(fields.Name, meta_v1.GetOptions{})
	if kerrors.IsNotFound(err) {

		err = ServiceTemplate1.Execute(&replicaServiceDoc, fields)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		replicaServiceDocString := replicaServiceDoc.String()
		log.Debug(replicaServiceDocString)

		replicaService := v1.Service{}
		err = json.Unmarshal(replicaServiceDoc.Bytes(), &replicaService)
		if err != nil {
			log.Error("error unmarshalling json into replica Service " + err.Error())
			return err
		}

		replicaServiceResult, err = clientset.Core().Services(namespace).Create(&replicaService)
		if err != nil {
			log.Error("error creating replica Service " + err.Error())
			return err
		}
		log.Info("created replica service " + replicaServiceResult.Name + " in namespace " + namespace)
	}

	return err

}
