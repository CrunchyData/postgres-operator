// Package cluster holds the cluster CRD logic and definitions
// A cluster is comprised of a primary service, replica service,
// primary deployment, and replica deployment
package cluster

/*
 Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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
	"context"
	"encoding/json"
	"os"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/operator"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateService ...
func CreateService(clientset kubernetes.Interface, fields *ServiceTemplateFields, namespace string) error {
	ctx := context.TODO()
	var serviceDoc bytes.Buffer

	// create the service if it doesn't exist
	_, err := clientset.CoreV1().Services(namespace).Get(ctx, fields.Name, metav1.GetOptions{})
	if err != nil {

		err = config.ServiceTemplate.Execute(&serviceDoc, fields)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if operator.CRUNCHY_DEBUG {
			_ = config.ServiceTemplate.Execute(os.Stdout, fields)
		}

		service := corev1.Service{}
		err = json.Unmarshal(serviceDoc.Bytes(), &service)
		if err != nil {
			log.Error("error unmarshalling json into Service " + err.Error())
			return err
		}

		_, err = clientset.CoreV1().Services(namespace).Create(ctx, &service, metav1.CreateOptions{})
	}

	return err
}
