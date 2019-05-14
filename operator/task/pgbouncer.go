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
	clusteroperator "github.com/crunchydata/postgres-operator/operator/cluster"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Update authorizations for pgbouncer user in postgres database
func UpdatePgBouncerAuthorizations(clusterName string, clientset *kubernetes.Clientset, RESTClient *rest.RESTClient, ns, podIP string) {

	taskName := clusterName + crv1.PgtaskUpdatePgbouncerAuths
	task := crv1.Pgtask{}
	task.Spec = crv1.PgtaskSpec{}
	task.Spec.Name = taskName
	var username, password string

	found, err := kubeapi.Getpgtask(RESTClient, &task, taskName, ns)
	if found && err == nil {

		// get secret name from task and retrieve username and password from secret.

		secretName := task.ObjectMeta.Labels[config.LABEL_PGBOUNCER_SECRET]
		log.Debugf("Got secret for pgbouncer: %s", secretName)

		username, password, err = util.GetPasswordFromSecret(clientset, ns, secretName)

		if err != nil {
			log.Debug("Error retrieving username and password from pgbouncer secret")
			log.Debug(err.Error())
		} else {

			log.Debugf("Updating pgbouncer auth at ip %s", podIP)
			err = clusteroperator.UpdatePgBouncerAuthorizations(clientset, ns, username, password, secretName, clusterName, podIP)

			if err != nil {
				log.Debug("Failed to update existing pgbouncer credentials")
				log.Debug(err.Error())
			} else {
				log.Debug("Pgbouncer authorization update complete")
			}
		}

		//delete the pgtask to not redo this again
		kubeapi.Deletepgtask(RESTClient, taskName, ns)
	}
}
