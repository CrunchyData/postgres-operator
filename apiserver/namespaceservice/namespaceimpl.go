package namespaceservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/events"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

func ShowNamespace(username string) msgs.ShowNamespaceResponse {
	log.Debug("ShowNamespace called")
	response := msgs.ShowNamespaceResponse{}
	response.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	response.Results = make([]msgs.NamespaceResult, 0)
	namespaceList := util.GetNamespaces()

	for i := 0; i < len(namespaceList); i++ {
		r := msgs.NamespaceResult{
			Namespace:  namespaceList[i],
			UserAccess: apiserver.UserIsPermittedInNamespace(username, namespaceList[i]),
		}
		response.Results = append(response.Results, r)
	}

	return response
}

// CreateNamespace ...
func CreateNamespace(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreateNamespaceRequest) msgs.CreateNamespaceResponse {

	log.Debugf("CreateNamespace %v", request)
	resp := msgs.CreateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	//iterate thru all the args (namespace names)
	for _, ns := range request.Args {
		//validate the list of args (namespaces)
		errs := validation.IsDNS1035Label(ns)
		if len(errs) > 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "invalid namespace name format " + errs[0] + " namespace name " + ns
			return resp
		}

		_, found, _ := kubeapi.GetNamespace(clientset, ns)
		if found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " already exists"
			return resp
		}

		//define the new namespace
		newns := v1.Namespace{}
		newns.Name = ns

		err := kubeapi.CreateNamespace(clientset, &newns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace create error " + ns + err.Error()
			return resp
		}

		log.Debugf("CreateNamespace %s created by %s", ns)
		//apply targeted rbac rules here

		resp.Results = append(resp.Results, "created namespace "+ns)
		//publish event
		topics := make([]string, 1)
		topics[0] = events.EventTopicPGO

		f := events.EventPGOCreateNamespaceFormat{
			EventHeader: events.EventHeader{
				Namespace: apiserver.PgoNamespace,
				Username:  createdBy,
				Topic:     topics,
				EventType: events.EventPGOCreateNamespace,
			},
			CreatedNamespace: ns,
		}

		err = events.Publish(f)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	return resp

}

// DeleteNamespace ...
func DeleteNamespace(clientset *kubernetes.Clientset, deletedBy string, request *msgs.DeleteNamespaceRequest) msgs.DeleteNamespaceResponse {
	resp := msgs.DeleteNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, ns := range request.Args {

		_, found, _ := kubeapi.GetNamespace(clientset, ns)
		if !found {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace " + ns + " not found"
			return resp
		}

		err := kubeapi.DeleteNamespace(clientset, ns)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

		log.Debugf("DeleteNamespace %s deleted by %s", ns)
		resp.Results = append(resp.Results, "deleted namespace "+ns)

		//publish the namespace delete event
		topics := make([]string, 1)
		topics[0] = events.EventTopicPGO

		f := events.EventPGODeleteNamespaceFormat{
			EventHeader: events.EventHeader{
				Namespace: apiserver.PgoNamespace,
				Username:  deletedBy,
				Topic:     topics,
				EventType: events.EventPGODeleteNamespace,
			},
			DeletedNamespace: ns,
		}

		err = events.Publish(f)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

	}

	return resp

}

func installTargetRBAC() {
	//Role pgo-role (conf/postgres-operator/pgo-role.json)
	//RoleBinding pgo-role-binding (conf/postgres-operator/pgo-role-binding.json)
	//Role pgo-backrest-role (conf/postgres-operator/pgo-backrest-role.json)
	//ServiceAccount pgo-backrest (conf/postgres-operator/pgo-backrest-sa.json)
	//RoleBinding pgo-backrest-role-binding (conf/postgres-operator/pgo-backrest-role-binding.json)
}
