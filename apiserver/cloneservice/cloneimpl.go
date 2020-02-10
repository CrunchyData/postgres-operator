package cloneservice

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"fmt"
	"io/ioutil"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//  Clone allows a user to clone a cluster into a new deployment
func Clone(request *msgs.CloneRequest, namespace, pgouser string) msgs.CloneResponse {
	log.Debugf("clone called with ")

	// set up the response here
	response := msgs.CloneResponse{
		Status: msgs.Status{
			Code: msgs.Ok,
			Msg:  "",
		},
	}

	if err := request.Validate(); err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Invalid clone request: %s", err)
		return response
	}

	log.Debug("Getting pgcluster")

	// get the information about the current pgcluster by name, to ensure it
	// exists
	sourcePgcluster := crv1.Pgcluster{}
	_, err := kubeapi.Getpgcluster(apiserver.RESTClient, &sourcePgcluster,
		request.SourceClusterName, namespace)

	// if there is an error getting the pgcluster, abort here
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Could not get cluster: %s", err)
		return response
	}

	// now, let's ensure the target pgCluster does *not* exist
	targetPgcluster := crv1.Pgcluster{}
	targetPgclusterExists, _ := kubeapi.Getpgcluster(apiserver.RESTClient,
		&targetPgcluster, request.TargetClusterName, namespace)

	if targetPgclusterExists {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Could not clone cluster: %s already exists",
			request.TargetClusterName)
		return response
	}

	// finally, let's make sure there is not already a task in progress for
	// making the clone
	selector := fmt.Sprintf("%s=true,pg-cluster=%s", config.LABEL_PGO_CLONE, request.TargetClusterName)
	taskList := crv1.PgtaskList{}

	if err := kubeapi.GetpgtasksBySelector(apiserver.RESTClient, &taskList, selector, namespace); err != nil {
		log.Error(err)
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Could not clone cluster: could not validate %s", err.Error())
		return response
	}

	// iterate through the list of tasks and see if there are any pending
	for _, task := range taskList.Items {
		if task.Spec.Status != crv1.CompletedStatus {
			response.Status.Code = msgs.Error
			response.Status.Msg = fmt.Sprintf("Could not clone cluster: there exists an ongoing clone task: [%s]. If you believe this is an error, try deleting this pgtask CRD.", task.Spec.Name)
			return response
		}
	}

	// create the workflow task to track how this is progressing
	uid := util.RandStringBytesRmndr(4)
	workflowID, err := createWorkflowTask(request.TargetClusterName, uid, namespace)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Errorf("could not create clone workflow task: %s", err).Error()
		return response
	}

	// clone is a form of restore, so validate using ValidateBackrestStorageTypeOnBackupRestore
	err = util.ValidateBackrestStorageTypeOnBackupRestore(request.BackrestStorageSource,
		sourcePgcluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE], true)
	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = err.Error()
		return response
	}

	// alright, begin the create the proper clone task!
	cloneTask := util.CloneTask{
		PGOUser:               pgouser,
		SourceClusterName:     request.SourceClusterName,
		TargetClusterName:     request.TargetClusterName,
		TaskStepLabel:         config.LABEL_PGO_CLONE_STEP_1,
		TaskType:              crv1.PgtaskCloneStep1,
		Timestamp:             time.Now(),
		WorkflowID:            workflowID,
		BackrestStorageSource: request.BackrestStorageSource,
	}

	task := cloneTask.Create()

	// create the Pgtask CRD for the clone task
	err = kubeapi.Createpgtask(apiserver.RESTClient, task, namespace)

	if err != nil {
		response.Status.Code = msgs.Error
		response.Status.Msg = fmt.Sprintf("Could not create clone task: %s", err)
		return response
	}

	response.TargetClusterName = request.TargetClusterName
	response.WorkflowID = workflowID

	return response
}

// createWorkflowTask creates the workflow task that is tracked as we attempt
// to clone the cluster
func createWorkflowTask(targetClusterName, uid, namespace string) (string, error) {
	// set a random ID for this workflow task
	u, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")

	if err != nil {
		return "", err
	}

	id := string(u[:len(u)-1])

	// set up the workflow task
	taskName := fmt.Sprintf("%s-%s-%s", targetClusterName, uid, crv1.PgtaskWorkflowCloneType)
	task := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: taskName,
			Labels: map[string]string{
				config.LABEL_PG_CLUSTER: targetClusterName,
				crv1.PgtaskWorkflowID:   id,
			},
		},
		Spec: crv1.PgtaskSpec{
			Namespace: namespace,
			Name:      taskName,
			TaskType:  crv1.PgtaskWorkflow,
			Parameters: map[string]string{
				crv1.PgtaskWorkflowSubmittedStatus: time.Now().Format(time.RFC3339),
				config.LABEL_PG_CLUSTER:            targetClusterName,
				crv1.PgtaskWorkflowID:              id,
			},
		},
	}

	// create the workflow task
	err = kubeapi.Createpgtask(apiserver.RESTClient, task, namespace)

	if err != nil {
		return "", err
	}

	// return succesfully after creating the task
	return id, nil
}
