package pgdumpservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/util"
	"github.com/gorilla/mux"
	"net/http"
)

// BackupHandler ...
// pgo backup --backup-type=pgdump mycluster
func BackupHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("pgdumpservice.CreatepgDumpHandlerBackupHandler called")

	var request msgs.CreatepgDumpBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err := apiserver.Authn(apiserver.CREATE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := CreatepgDump(&request)

	json.NewEncoder(w).Encode(resp)
}

// ShowpgDumpHandler ...
// returns a ShowpgDumpResponse
func ShowDumpHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("pgdumpservice.ShowDumpHandler %v\n", vars)

	clustername := vars[util.LABEL_NAME]

	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debugf("version parameter is [%s]", clientVersion)
	}
	selector := r.URL.Query().Get(util.LABEL_SELECTOR)
	if selector != "" {
		log.Debugf("selector parameter is [%s]", selector)
	}

	err := apiserver.Authn(apiserver.SHOW_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")

	log.Debug("pgdumpservice.pgdumpHandler GET called")
	var resp msgs.ShowBackupResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowBackupResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}

	} else {
		resp = ShowpgDump(clustername, selector)
	}
	json.NewEncoder(w).Encode(resp)

}

// RestoreHandler ...
// pgo restore mycluster --to-cluster=restored
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("pgdumpservice.RestoreHandler called")

	var request msgs.RestoreRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err = apiserver.Authn(apiserver.RESTORE_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := Restore(&request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
