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
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
)

// CompleteCreateClusterWorkflow ... update the pgtask for the
// create cluster workflow for a given cluster
func CompleteCreateClusterWorkflow(clusterName string, Clientset *kubernetes.Clientset, RESTClient *rest.RESTClient, ns string) {

	taskName := clusterName + "-" + crv1.PgtaskWorkflowCreateClusterType

	task := crv1.Pgtask{}
	task.Spec = crv1.PgtaskSpec{}
	task.Spec.Name = taskName

	found, err := kubeapi.Getpgtask(RESTClient, &task, taskName, ns)
	if found && err == nil {
		//mark this workflow as completed
		id := task.Spec.Parameters[crv1.PgtaskWorkflowID]

		log.Debugf("completing workflow %s  id %s", taskName, id)
		task.Spec.Parameters[crv1.PgtaskWorkflowCompletedStatus] = time.Now().Format("2006-01-02.15.04.05")

		//update pgtask
		err = kubeapi.Updatepgtask(RESTClient, &task, taskName, ns)
		if err != nil {
			log.Error(err)
		}
	} else {
		log.Errorf("Error completing  workflow %s  id %s", taskName, task.Spec.Parameters[crv1.PgtaskWorkflowID])
		log.Error(err)
	}

}

func CompleteBackupWorkflow(clusterName string, clientSet *kubernetes.Clientset, RESTClient *rest.RESTClient, ns string) {

	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackupType

	task := crv1.Pgtask{}
	task.Spec = crv1.PgtaskSpec{}
	task.Spec.Name = taskName

	found, err := kubeapi.Getpgtask(RESTClient, &task, taskName, ns)
	if found && err == nil {
		//mark this workflow as completed
		id := task.Spec.Parameters[crv1.PgtaskWorkflowID]

		log.Debugf("completing workflow %s  id %s", taskName, id)
		task.Spec.Parameters[crv1.PgtaskWorkflowCompletedStatus] = time.Now().Format("2006-01-02.15.04.05")

		//update pgtask
		err = kubeapi.Updatepgtask(RESTClient, &task, taskName, ns)
		if err != nil {
			log.Error(err)
		}
	} else {
		log.Errorf("Error completing  workflow %s  id %s", taskName, task.Spec.Parameters[crv1.PgtaskWorkflowID])
		log.Error(err)
	}

}
