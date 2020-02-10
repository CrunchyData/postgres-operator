package namespaceservice

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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

func ShowNamespace(clientset *kubernetes.Clientset, username string, request *msgs.ShowNamespaceRequest) msgs.ShowNamespaceResponse {
	log.Debug("ShowNamespace called")
	resp := msgs.ShowNamespaceResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}
	resp.Username = username
	resp.Results = make([]msgs.NamespaceResult, 0)

	//namespaceList := util.GetNamespaces()

	nsList := make([]string, 0)

	if request.AllFlag {
		namespaceList, err := kubeapi.GetNamespaces(clientset)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		for _, v := range namespaceList.Items {
			nsList = append(nsList, v.Name)
		}
	} else {
		if len(request.Args) == 0 {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = "namespace names or --all flag is required for this command"
			return resp
		}

		for i := 0; i < len(request.Args); i++ {
			_, found, _ := kubeapi.GetNamespace(clientset, request.Args[i])
			if found == false {
				resp.Status.Code = msgs.Error
				resp.Status.Msg = "namespace " + request.Args[i] + " not found"

				return resp
			} else {
				nsList = append(nsList, request.Args[i])
			}
		}
	}

	for i := 0; i < len(nsList); i++ {
		iaccess, uaccess := apiserver.UserIsPermittedInNamespace(username, nsList[i])
		r := msgs.NamespaceResult{
			Namespace:          nsList[i],
			InstallationAccess: iaccess,
			UserAccess:         uaccess,
		}
		resp.Results = append(resp.Results, r)
	}

	return resp
}

// CreateNamespace ...
func CreateNamespace(clientset *kubernetes.Clientset, createdBy string, request *msgs.CreateNamespaceRequest) msgs.CreateNamespaceResponse {

	log.Debugf("CreateNamespace %v", request)
	resp := msgs.CreateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	//iterate thru all the args (namespace names)
	for _, namespace := range request.Args {

		err := ns.CreateNamespace(clientset, apiserver.InstallationName, apiserver.PgoNamespace, createdBy, namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "created namespace "+namespace)

	}

	return resp

}

// DeleteNamespace ...
func DeleteNamespace(clientset *kubernetes.Clientset, deletedBy string, request *msgs.DeleteNamespaceRequest) msgs.DeleteNamespaceResponse {
	resp := msgs.DeleteNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	for _, namespace := range request.Args {

		err := ns.DeleteNamespace(clientset, apiserver.InstallationName, apiserver.PgoNamespace, deletedBy, namespace)

		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "deleted namespace "+namespace)

	}

	return resp

}

// UpdateNamespace ...
func UpdateNamespace(clientset *kubernetes.Clientset, updatedBy string, request *msgs.UpdateNamespaceRequest) msgs.UpdateNamespaceResponse {

	log.Debugf("UpdateNamespace %v", request)
	resp := msgs.UpdateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	//iterate thru all the args (namespace names)
	for _, namespace := range request.Args {

		err := ns.UpdateNamespace(clientset, apiserver.InstallationName, apiserver.PgoNamespace, updatedBy, namespace)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
		resp.Results = append(resp.Results, "updated namespace "+namespace)

	}

	return resp

}
