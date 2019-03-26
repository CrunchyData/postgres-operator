package task

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// RemoveBackups ...
func RemoveBackups(namespace string, clientset *kubernetes.Clientset, task *crv1.Pgtask) {

	//delete any backup jobs for this cluster
	//kubectl delete job --selector=pg-cluster=clustername

	log.Debugf("deleting backup jobs with selector=%s=%s", config.LABEL_PG_CLUSTER, task.Spec.Parameters[config.LABEL_PG_CLUSTER])
	kubeapi.DeleteJobs(clientset, config.LABEL_PG_CLUSTER+"="+task.Spec.Parameters[config.LABEL_PG_CLUSTER], namespace)

}
