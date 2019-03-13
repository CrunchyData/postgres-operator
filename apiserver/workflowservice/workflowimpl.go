package workflowservice

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
	"errors"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
)

// ShowWorkflow ...
func ShowWorkflow(id, ns string) (msgs.ShowWorkflowDetail, error) {

	log.Debugf("ShowWorkflow called with id %s", id)
	detail := msgs.ShowWorkflowDetail{}

	//get the pgtask for this workflow

	selector := crv1.PgtaskWorkflowID + "=" + id

	taskList := crv1.PgtaskList{}

	err := kubeapi.GetpgtasksBySelector(apiserver.RESTClient, &taskList, selector, ns)
	if err != nil {
		return detail, err
	}
	if len(taskList.Items) > 1 {
		return detail, errors.New("more than 1 workflow id found for id " + id)
	}
	if len(taskList.Items) == 0 {
		return detail, errors.New("workflow id NOT found for id " + id)
	}
	t := taskList.Items[0]
	detail.ClusterName = t.Spec.Parameters[config.LABEL_PG_CLUSTER]
	detail.Parameters = t.Spec.Parameters

	return detail, err

}
