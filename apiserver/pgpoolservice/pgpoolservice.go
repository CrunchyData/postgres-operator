package pgpoolservice

/*
Copyright 2017-2019 Crunchy Data Solutions, Inc.
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

// CreatePgpoolHandler ...
// pgo create pgpool
func CreatePgpoolHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("pgpoolservice.CreatePgpoolHandler called")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := apiserver.Authn(apiserver.CREATE_PGPOOL_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var request msgs.CreatePgpoolRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreatePgpoolResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = CreatePgpool(&request)
	}
	json.NewEncoder(w).Encode(resp)

}

// DeletePgpoolHandler ...
// pgo delete pgpool
func DeletePgpoolHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("pgpoolservice.DeletePgpoolHandler called")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := apiserver.Authn(apiserver.DELETE_PGPOOL_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var request msgs.DeletePgpoolRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.DeletePgpoolResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = DeletePgpool(&request)
	}
	json.NewEncoder(w).Encode(resp)

}
