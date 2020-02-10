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
	"encoding/json"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// ShowNamespaceHandler ...
// pgo show namespace
func ShowNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /namespace namespaceservice namespace
	/*```
	  Show namespace information
	*/
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: "Show Namespace Request"
	//   in: "body"
	//   schema:
	//     "$ref": "#/definitions/ShowNamespaceRequest"
	//	responses:
	//	  '200':
	//	    description: Output
	//	    schema:
	//	      "$ref": "#/definitions/ShowNamespaceResponse"

	resp := msgs.ShowNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("namespaceservice.ShowNamespaceHandler called")

	var request msgs.ShowNamespaceRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("ShowNamespaceHandler called [%v]", request)

	username, err := apiserver.Authn(apiserver.SHOW_NAMESPACE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowNamespace(apiserver.Clientset, username, &request)
	json.NewEncoder(w).Encode(resp)
}

func CreateNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /namespacecreate namespaceservice namespacecreate
	/*```
	  Create a namespace
	*/
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: "Create Namespace"
	//   in: "body"
	//   schema:
	//     "$ref": "#/definitions/CreateNamespaceRequest"
	// responses:
	//   '200':
	//     description: Output
	//     schema:
	//       "$ref": "#/definitions/CreateNamespaceResponse"
	resp := msgs.CreateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("namespaceservice.CreateNamespaceHandler called")

	var request msgs.CreateNamespaceRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_NAMESPACE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debugf("namespaceservice.CreateNamespaceHandler got request %v", request)
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateNamespace(apiserver.Clientset, username, &request)
	json.NewEncoder(w).Encode(resp)
}

func DeleteNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /namespacedelete namespaceservice namespacedelete
	/*```
	  Delete a namespaces
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete Namespace"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeleteNamespaceRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeleteNamespaceResponse"
	var request msgs.DeleteNamespaceRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("DeleteNamespaceHandler parameters [%v]", request)

	username, err := apiserver.Authn(apiserver.DELETE_NAMESPACE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.DeleteNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeleteNamespace(apiserver.Clientset, username, &request)
	json.NewEncoder(w).Encode(resp)

}
func UpdateNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /namespaceupdate namespaceservice namespaceupdate
	/*```
	  Update a namespace, applying Operator RBAC
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Update Namespace"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/UpdateNamespaceRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/UpdateNamespaceResponse"
	resp := msgs.UpdateNamespaceResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	log.Debug("namespaceservice.UpdateNamespaceHandler called")

	var request msgs.UpdateNamespaceRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.UPDATE_NAMESPACE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debugf("namespaceservice.UpdateNamespaceHandler got request %v", request)
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = UpdateNamespace(apiserver.Clientset, username, &request)
	json.NewEncoder(w).Encode(resp)
}
