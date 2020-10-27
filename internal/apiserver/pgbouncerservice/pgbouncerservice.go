package pgbouncerservice

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/apiserver"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// CreatePgbouncerHandler ...
// pgo create pgbouncer
func CreatePgbouncerHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgbouncer pgbouncerservice pgbouncer-post
	/*```
	  Create a pgbouncer
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create Pgbouncer Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreatePgbouncerRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreatePgbouncerResponse"
	var ns string
	log.Debug("pgbouncerservice.CreatePgbouncerHandler called")
	username, err := apiserver.Authn(apiserver.CREATE_PGBOUNCER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var request msgs.CreatePgbouncerRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreatePgbouncerResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreatePgbouncer(&request, ns, username)
	json.NewEncoder(w).Encode(resp)

}

/* The delete pgboucner handler is setup to be used by two different routes. To keep
the documentation consistent with the API this endpoint is documented along with the
/pgbouncer (DELETE) enpoint. This endpoint should be deprecated in future API versions.
*/
// swagger:operation DELETE /pgbouncer pgbouncerservice pgbouncer-delete
/*```
Delete a pgbouncer from a cluster
*/
// ---
//  produces:
//  - application/json
//  parameters:
//  - name: "Delete PgBouncer Request"
//    in: "body"
//    schema:
//      "$ref": "#/definitions/DeletePgbouncerRequest"
//  responses:
//    '200':
//      description: Output
//      schema:
//        "$ref": "#/definitions/DeletePgbouncerResponse"
// DeletePgbouncerHandler ...
// pgo delete pgbouncer
func DeletePgbouncerHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /pgbouncer pgbouncerservice pgbouncer-delete
	/*```
	  Delete a pgbouncer from a cluster
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete PgBouncer Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeletePgbouncerRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeletePgbouncerResponse"
	var ns string
	log.Debug("pgbouncerservice.DeletePgbouncerHandler called")
	username, err := apiserver.Authn(apiserver.DELETE_PGBOUNCER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var request msgs.DeletePgbouncerRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.DeletePgbouncerResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeletePgbouncer(&request, ns)
	json.NewEncoder(w).Encode(resp)

}

// ShowPgBouncerHandler is the HTTP handler to get information about a pgBouncer
// deployment, aka `pgo show pgbouncer`
func ShowPgBouncerHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgbouncer/show pgbouncerservice pgbouncer-post
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
	//      "$ref": "#/definitions/ShowPgBouncerRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowPgBouncerResponse"
	log.Debug("pgbouncerservice.ShowPgbouncerHandler called")

	// first, determine if the user is authorized to access this resource
	username, err := apiserver.Authn(apiserver.SHOW_PGBOUNCER_PERM, w, r)

	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	// get the information that is in the request
	var request msgs.ShowPgBouncerRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	// ensure the versions align...
	if request.ClientVersion != msgs.PGO_VERSION {
		response := msgs.ShowPgBouncerResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  apiserver.VERSION_MISMATCH_ERROR,
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// ensure the namespace being used exists
	namespace, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)

	if err != nil {
		response := msgs.ShowPgBouncerResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// get the information about a pgbouncer deployment(s)
	response := ShowPgBouncer(&request, namespace)
	json.NewEncoder(w).Encode(response)

}

// UpdatePgBouncerHandler is the HTTP handler to perform update tasks on a
// pgbouncer instance, such as rotating the password
func UpdatePgBouncerHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation PUT /pgbouncer pgbouncerservice pgbouncer-put
	/*```
	  Update a pgBouncer cluster, e.g. rotate the password
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Update PGBouncer"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/UpdatePgBouncerRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/UpdatePgBouncerResponse"
	log.Debug("pgbouncerservice.UpdatePgbouncerHandler called")

	// first, determine if the user is authorized to access this resource
	username, err := apiserver.Authn(apiserver.UPDATE_PGBOUNCER_PERM, w, r)

	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	// get the information that is in the request
	var request msgs.UpdatePgBouncerRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	// ensure the versions align...
	if request.ClientVersion != msgs.PGO_VERSION {
		response := msgs.UpdatePgBouncerResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  apiserver.VERSION_MISMATCH_ERROR,
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// ensure the namespace being used exists
	namespace, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)

	if err != nil {
		response := msgs.UpdatePgBouncerResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// get the information about a pgbouncer deployment(s)
	response := UpdatePgBouncer(&request, namespace, username)
	json.NewEncoder(w).Encode(response)
}
