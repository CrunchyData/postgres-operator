package backrestservice

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
	"github.com/crunchydata/postgres-operator/util"
	"github.com/gorilla/mux"
	"net/http"
)

// CreateBackupHandler ...
// pgo backup all
// pgo backup --selector=name=mycluster
// pgo backup mycluster
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("backrestservice.CreateBackupHandler called")

	var request msgs.CreateBackrestBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	err = apiserver.Authn(apiserver.CREATE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := CreateBackup(&request)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}

// ShowBackrestHandler ...
// returns a ShowBackrestResponse
func ShowBackrestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("backrestservice.ShowBackrestHandler %v\n", vars)

	backupname := vars[util.LABEL_NAME]

	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}
	selector := r.URL.Query().Get(util.LABEL_SELECTOR)
	if selector != "" {
		log.Debug("selector param was [" + selector + "]")
	}

	err := apiserver.Authn(apiserver.SHOW_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")

	log.Debug("backrestservice.ShowBackrestHandler GET called")
	var resp msgs.ShowBackrestResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowBackrestResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}

	} else {
		resp = ShowBackrest(backupname, selector)
	}
	json.NewEncoder(w).Encode(resp)

}

// RestoreHandler ...
// pgo restore mycluster --to-cluster=restored
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("backrestservice.RestoreHandler called")

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
