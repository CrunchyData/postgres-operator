// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

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
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/operator"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

// CreateService ...
func CreateService(clientset *kubernetes.Clientset, fields *ServiceTemplateFields, namespace string) error {
	var serviceDoc bytes.Buffer

	//create the service if it doesn't exist
	_, found, err := kubeapi.GetService(clientset, fields.Name, namespace)
	if !found || err != nil {

		err = config.ServiceTemplate.Execute(&serviceDoc, fields)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if operator.CRUNCHY_DEBUG {
			config.ServiceTemplate.Execute(os.Stdout, fields)
		}

		service := v1.Service{}
		err = json.Unmarshal(serviceDoc.Bytes(), &service)
		if err != nil {
			log.Error("error unmarshalling json into Service " + err.Error())
			return err
		}

		_, err = kubeapi.CreateService(clientset, &service, namespace)
	}

	return err

}
