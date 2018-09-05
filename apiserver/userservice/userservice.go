package userservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// UserHandler ...
// pgo user XXXX
func UserHandler(w http.ResponseWriter, r *http.Request) {

	log.Debug("userservice.UserHandler called")

	var request msgs.UserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err := apiserver.Authn(apiserver.USER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var resp msgs.UserResponse
	if request.ClientVersion != msgs.PGO_VERSION {
		resp = msgs.UserResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = User(&request)
	}

	json.NewEncoder(w).Encode(resp)
}

// CreateUserHandler ...
// pgo create user
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("userservice.CreateUserHandler called")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := apiserver.Authn(apiserver.CREATE_USER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var request msgs.CreateUserRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreateUserResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = CreateUser(&request)
	}
	json.NewEncoder(w).Encode(resp)

}

// DeleteUserHandler ...
// pgo delete user someuser
// parameters name
// parameters selector
// returns a DeleteUserResponse
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("userservice.DeleteUserHandler %v\n", vars)

	username := vars["name"]

	selector := r.URL.Query().Get("selector")
	if selector != "" {
		log.Debug("selector param was [" + selector + "]")
	}
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err := apiserver.Authn(apiserver.DELETE_USER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	log.Debug("userservice.DeleteUserHandler DELETE called")
	var resp msgs.DeleteUserResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.DeleteUserResponse{}
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = DeleteUser(username, selector)
	}
	json.NewEncoder(w).Encode(resp)

}

// ShowUserHandler ...
// pgo show user
// parameters selector
// returns a ShowUserResponse
func ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("userservice.ShowUserHandler %v\n", vars)

	clustername := vars["name"]

	selector := r.URL.Query().Get("selector")
	if selector != "" {
		log.Debug("selector param was [" + selector + "]")
	}
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err := apiserver.Authn(apiserver.SHOW_SECRETS_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	log.Debug("userservice.ShowUserHandler GET called")

	var resp msgs.ShowUserResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowUserResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]msgs.ShowUserDetail, 0)
	} else {
		resp = ShowUser(clustername, selector)
	}
	json.NewEncoder(w).Encode(resp)

}
