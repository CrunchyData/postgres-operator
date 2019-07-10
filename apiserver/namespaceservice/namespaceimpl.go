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
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
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
	for i := 0; i < len(request.Args); i++ {
		//validate the list of args (namespaces)
		errs := validation.IsDNS1035Label(request.Args[i])
		if len(errs) > 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "invalid namespace name format " + errs[0] + " namespace name " + request.Args[i]
			return resp
		}

		log.Debugf("CreateNamespace %s created by %s", request.Args[i], createdBy)
		resp.Results = append(resp.Results, "created namespace "+request.Args[i])
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
			CreatedNamespace: request.Args[i],
		}

		err := events.Publish(f)
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

	for i := 0; i < len(request.Args); i++ {
		log.Debugf("DeleteNamespace %s deleted by %s", request.Args[i], deletedBy)
		resp.Results = append(resp.Results, "deleted namespace "+request.Args[i])

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
			DeletedNamespace: request.Args[i],
		}

		err := events.Publish(f)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}

	}

	return resp

}
