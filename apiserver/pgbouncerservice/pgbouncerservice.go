package pgbouncerservice

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
	log "github.com/sirupsen/logrus"
	"net/http"
)

// CreatePgbouncerHandler ...
// pgo create pgbouncer
func CreatePgbouncerHandler(w http.ResponseWriter, r *http.Request) {
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

	resp = CreatePgbouncer(&request, ns)
	json.NewEncoder(w).Encode(resp)

}

// DeletePgbouncerHandler ...
// pgo delete pgbouncer
func DeletePgbouncerHandler(w http.ResponseWriter, r *http.Request) {
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
