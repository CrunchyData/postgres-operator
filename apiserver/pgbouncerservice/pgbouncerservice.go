package pgbouncerservice

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
	"net/http"
)

// CreatePgbouncerHandler ...
// pgo create pgbouncer
func CreatePgbouncerHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("pgbouncerservice.CreatePgbouncerHandler called")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := apiserver.Authn(apiserver.CREATE_PGBOUNCER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var request msgs.CreatePgbouncerRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreatePgbouncerResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = CreatePgbouncer(&request)
	}
	json.NewEncoder(w).Encode(resp)

}

// DeletePgbouncerHandler ...
// pgo delete pgbouncer
func DeletePgbouncerHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("pgbouncerservice.DeletePgbouncerHandler called")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := apiserver.Authn(apiserver.DELETE_PGBOUNCER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var request msgs.DeletePgbouncerRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.DeletePgbouncerResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = DeletePgbouncer(&request)
	}
	json.NewEncoder(w).Encode(resp)

}
