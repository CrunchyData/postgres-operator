package task

/*
 Copyright 2018 - 2022 Crunchy Data Solutions, Inc.
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
	"encoding/json"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	pgo "github.com/crunchydata/postgres-operator/pkg/generated/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CompleteCreateClusterWorkflow ... update the pgtask for the
// create cluster workflow for a given cluster
func CompleteCreateClusterWorkflow(clusterName string, clientset pgo.Interface, ns string) {

	taskName := clusterName + "-" + crv1.PgtaskWorkflowCreateClusterType

	completeWorkflow(clientset, ns, taskName)

}

func CompleteBackupWorkflow(clusterName string, clientset pgo.Interface, ns string) {

	taskName := clusterName + "-" + crv1.PgtaskWorkflowBackupType

	completeWorkflow(clientset, ns, taskName)

}

func completeWorkflow(clientset pgo.Interface, taskNamespace, taskName string) {

	task, err := clientset.CrunchydataV1().Pgtasks(taskNamespace).Get(taskName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Error completing  workflow %s", taskName)
		log.Error(err)
		return
	}

	//mark this workflow as completed
	id := task.Spec.Parameters[crv1.PgtaskWorkflowID]
	log.Debugf("completing workflow %s  id %s", taskName, id)

	task.Spec.Parameters[crv1.PgtaskWorkflowCompletedStatus] = time.Now().Format(time.RFC3339)

	patch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"parameters": task.Spec.Parameters,
		},
	})
	if err == nil {
		_, err = clientset.CrunchydataV1().Pgtasks(task.Namespace).Patch(task.Name, types.MergePatchType, patch)
	}
	if err != nil {
		log.Error(err)
	}

}
