package userservice

/*
Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

// UserHandler provides a means to update a PostgreSQL user
// pgo update user
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /userupdate userservice userupdate
	/*```
	Update a postgres user
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Update User Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/UpdateUserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/UpdateUserResponse"
	log.Debug("userservice.UpdateUserHandler called")

	var request msgs.UpdateUserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.UpdateUserResponse{}
	username, err := apiserver.Authn(apiserver.UPDATE_USER_PERM, w, r)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	_, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = UpdateUser(&request, username)

	_ = json.NewEncoder(w).Encode(resp)
}

// CreateUserHandler ...
// pgo create user
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /usercreate userservice usercreate
	/*```
	Create PostgreSQL user
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create User Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreateUserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreateUserResponse"
	log.Debug("userservice.CreateUserHandler called")
	username, err := apiserver.Authn(apiserver.CREATE_USER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var request msgs.CreateUserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreateUserResponse{}

	_, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateUser(&request, username)
	_ = json.NewEncoder(w).Encode(resp)
}

// DeleteUserHandler ...
// pgo delete user someuser
// parameters name
// parameters selector
// returns a DeleteUserResponse
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /userdelete userservice userdelete
	/*```
	Delete PostgreSQL user
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Delete User Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeleteUserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeleteUserResponse"
	var request msgs.DeleteUserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.DeleteUserResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	log.Debugf("DeleteUserHandler parameters %v", request)

	pgouser, err := apiserver.Authn(apiserver.DELETE_USER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = apiserver.GetNamespace(apiserver.Clientset, pgouser, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeleteUser(&request, pgouser)
	_ = json.NewEncoder(w).Encode(resp)
}

// ShowUserHandler allows one to display information about PostgreSQL uesrs that
// are in a PostgreSQL cluster
func ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /usershow userservice usershow
	/*```
	Show PostgreSQL user(s)
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Show User Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ShowUserRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowUserResponse"
	var request msgs.ShowUserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("ShowUserHandler parameters [%v]", request)

	username, err := apiserver.Authn(apiserver.SHOW_SECRETS_PERM, w, r)
	if err != nil {
		return
	}

	// a special authz check here: if the ShowSystemAccounts flag is set, ensure
	// the user is authorized to show system accounts
	if request.ShowSystemAccounts &&
		!apiserver.BasicAuthzCheck(username, apiserver.SHOW_SYSTEM_ACCOUNTS_PERM) {
		log.Errorf("Authorization Failed %s username=[%s]", apiserver.SHOW_SYSTEM_ACCOUNTS_PERM, username)
		http.Error(w, "Not authorized for this apiserver action", 403)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ShowUserResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	_, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowUser(&request)
	_ = json.NewEncoder(w).Encode(resp)
}
