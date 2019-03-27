package userservice

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
	"encoding/json"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// UserHandler ...
// pgo user XXXX
func UserHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug("userservice.UserHandler called")

	var request msgs.UserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.UserResponse{}
	username, err := apiserver.Authn(apiserver.USER_PERM, w, r)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	var ns string
	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = User(&request, ns)

	json.NewEncoder(w).Encode(resp)
}

// CreateUserHandler ...
// pgo create user
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
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

	var ns string
	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateUser(&request, ns)
	json.NewEncoder(w).Encode(resp)

}

// DeleteUserHandler ...
// pgo delete user someuser
// parameters name
// parameters selector
// returns a DeleteUserResponse
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	name := vars["name"]
	selector := r.URL.Query().Get("selector")
	namespace := r.URL.Query().Get("namespace")
	clientVersion := r.URL.Query().Get("version")

	log.Debugf("DeleteUserHandler parameters selector [%s] namespace [%s] version [%s] name [%s]", selector, namespace, clientVersion, name)

	username, err := apiserver.Authn(apiserver.DELETE_USER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.DeleteUserResponse{}

	var ns string
	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeleteUser(name, selector, ns)
	json.NewEncoder(w).Encode(resp)

}

// ShowUserHandler ...
// pgo show user
// parameters selector
// returns a ShowUserResponse
func ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	clustername := vars["name"]
	selector := r.URL.Query().Get("selector")
	namespace := r.URL.Query().Get("namespace")
	expired := r.URL.Query().Get("expired")
	clientVersion := r.URL.Query().Get("version")
	log.Debugf("ShowUserHandler parameters expired [%s] selector [%s] namespace [%s] expired [%s] version [%s]", expired, selector, namespace, clientVersion, clustername)

	username, err := apiserver.Authn(apiserver.SHOW_SECRETS_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ShowUserResponse{}
	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]msgs.ShowUserDetail, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	var ns string
	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		resp.Results = make([]msgs.ShowUserDetail, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowUser(clustername, selector, expired, ns)
	json.NewEncoder(w).Encode(resp)

}
