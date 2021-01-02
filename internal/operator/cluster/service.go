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
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// serviceInfo is a structured way of compiling all of the info required to
// update a service
type serviceInfo struct {
	serviceName      string
	serviceNamespace string
	serviceType      v1.ServiceType
}

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

// UpdateClusterService updates parameters (really just one) on a Service that
// represents a PostgreSQL cluster
func UpdateClusterService(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {
	return updateService(clientset, serviceInfo{
		serviceName:      cluster.Name,
		serviceNamespace: cluster.Namespace,
		serviceType:      cluster.Spec.ServiceType,
	})
}

// UpdateClusterService updates parameters (really just one) on a Service that
// represents a PostgreSQL replca instance
func UpdateReplicaService(clientset kubernetes.Interface, cluster *crv1.Pgcluster, replica *crv1.Pgreplica) error {
	serviceType := cluster.Spec.ServiceType

	// if the replica has a specific service type, override with that
	if replica.Spec.ServiceType != "" {
		serviceType = replica.Spec.ServiceType
	}

	return updateService(clientset, serviceInfo{
		serviceName:      replica.Spec.ClusterName + ReplicaSuffix,
		serviceNamespace: replica.Namespace,
		serviceType:      serviceType,
	})
}

// updateService does the legwork for updating a service
func updateService(clientset kubernetes.Interface, info serviceInfo) error {
	ctx := context.TODO()

	// first, attempt to get the Service. If we cannot do that, then we can't
	// update the service
	svc, err := clientset.CoreV1().Services(info.serviceNamespace).Get(ctx, info.serviceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// update the desired attributes, which is really just the ServiceType
	svc.Spec.Type = info.serviceType

	// ...so, while the documentation says that any "NodePort" settings are wiped
	// if the type is not "NodePort", this is actually not the case, so we need to
	// overcompensate for that
	// Ref: https://godoc.org/k8s.io/api/core/v1#ServicePort
	if svc.Spec.Type != v1.ServiceTypeNodePort {
		for i := range svc.Spec.Ports {
			svc.Spec.Ports[i].NodePort = 0
		}
	}

	_, err = clientset.CoreV1().Services(info.serviceNamespace).Update(ctx, svc, metav1.UpdateOptions{})

	return err
}
