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
	"github.com/crunchydata/postgres-operator/kubeapi"
	//	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	//	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//	"strings"
)

// RemoveBackups ...
func UpdatePgBouncerAuthorizations(clusterName string, Clientset *kubernetes.Clientset, RESTClient *rest.RESTClient, ns string) {

	taskName := clusterName + crv1.PgtaskUpdatePgbouncerAuths
	task := crv1.Pgtask{}
	task.Spec = crv1.PgtaskSpec{}
	task.Spec.Name = taskName

	found, err := kubeapi.Getpgtask(RESTClient, &task, taskName, ns)
	if found && err == nil {

		log.Debug("***** Updating pgbouncer authorization in postgres ****")

		//delete the pgtask to not redo this again
		kubeapi.Deletepgtask(RESTClient, taskName, ns)
	}
}
