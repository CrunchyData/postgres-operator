package pgadminservice

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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
	"net/http"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"

	log "github.com/sirupsen/logrus"
)

// CreatePgAdminHandler ...
// pgo create pgadmin
func CreatePgAdminHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgadmin pgadminservice pgadmin-post
	/*```
	  Create a pgAdmin instance
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create PgAdmin Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreatePgAdminRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreatePgAdminResponse"
	var ns string
	log.Debug("pgadminservice.CreatePgAdminHandler called")
	username, err := apiserver.Authn(apiserver.CREATE_PGADMIN_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var request msgs.CreatePgAdminRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreatePgAdminResponse{}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.SetError(apiserver.VERSION_MISMATCH_ERROR)
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.SetError(err.Error())
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreatePgAdmin(&request, ns, username)
	json.NewEncoder(w).Encode(resp)

}

// DeletePgAdminHandler ...
// pgo delete pgadmin
func DeletePgAdminHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /pgadmin pgadminservice pgadmin-delete
	/*```
	  Delete pgadmin from a cluster
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete PgAdmin Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeletePgAdminRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeletePgAdminResponse"
	var ns string
	log.Debug("pgadminservice.DeletePgAdminHandler called")
	username, err := apiserver.Authn(apiserver.DELETE_PGADMIN_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var request msgs.DeletePgAdminRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.DeletePgAdminResponse{}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.SetError(apiserver.VERSION_MISMATCH_ERROR)
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.SetError(err.Error())
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeletePgAdmin(&request, ns)
	json.NewEncoder(w).Encode(resp)

}

// ShowPgAdminHandler is the HTTP handler to get information about a pgBouncer
// deployment, aka `pgo show pgadmin`
func ShowPgAdminHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgadmin/show pgadminservice pgadmin-post
	/*```
	  Show information about a pgBouncer deployment
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Show PGBouncer Information"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ShowPgAdminRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowPgAdminResponse"
	log.Debug("pgadminservice.ShowPgAdminHandler called")

	// first, determine if the user is authorized to access this resource
	username, err := apiserver.Authn(apiserver.SHOW_PGADMIN_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// get the information that is in the request
	var request msgs.ShowPgAdminRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.ShowPgAdminResponse{}

	// ensure the versions align...
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.SetError(apiserver.VERSION_MISMATCH_ERROR)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// ensure the namespace being used exists
	namespace, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)

	if err != nil {
		resp.SetError(err.Error())
		json.NewEncoder(w).Encode(resp)
		return
	}

	// get the information about a pgAdmin deployment(s)
	resp = ShowPgAdmin(&request, namespace)
	json.NewEncoder(w).Encode(resp)

}
