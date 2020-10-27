package task

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"context"

	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RemoveBackups ...
func RemoveBackups(namespace string, clientset kubernetes.Interface, task *crv1.Pgtask) {
	ctx := context.TODO()

	//delete any backup jobs for this cluster
	//kubectl delete job --selector=pg-cluster=clustername

	log.Debugf("deleting backup jobs with selector=%s=%s", config.LABEL_PG_CLUSTER, task.Spec.Parameters[config.LABEL_PG_CLUSTER])
	deletePropagation := metav1.DeletePropagationForeground
	clientset.
		BatchV1().Jobs(namespace).
		DeleteCollection(ctx,
			metav1.DeleteOptions{PropagationPolicy: &deletePropagation},
			metav1.ListOptions{LabelSelector: config.LABEL_PG_CLUSTER + "=" + task.Spec.Parameters[config.LABEL_PG_CLUSTER]})
}
