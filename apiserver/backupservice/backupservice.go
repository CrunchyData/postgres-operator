package backupservice

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

// ShowBackupHandler ...
// returns a ShowBackupResponse
func ShowBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("backupservice.ShowBackupHandler %v\n", vars)

	backupname := vars["name"]

	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err := apiserver.Authn(apiserver.SHOW_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")

	log.Debug("backupservice.ShowBackupHandler GET called")
	var resp msgs.ShowBackupResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowBackupResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}

	} else {
		resp = ShowBackup(backupname)
	}
	json.NewEncoder(w).Encode(resp)

}

// DeleteBackupHandler ...
// returns a ShowBackupResponse
func DeleteBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("backupservice.DeleteBackupHandler %v\n", vars)

	backupname := vars["name"]
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err := apiserver.Authn(apiserver.DELETE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	log.Debug("backupservice.DeleteBackupHandler called")
	var resp msgs.DeleteBackupResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.DeleteBackupResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = DeleteBackup(backupname)
	}
	json.NewEncoder(w).Encode(resp)

}

// CreateBackupHandler ...
// pgo backup all
// pgo backup --selector=name=mycluster
// pgo backup mycluster
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Debug("backupservice.CreateBackupHandler called")

	var request msgs.CreateBackupRequest
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
