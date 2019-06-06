package backupservice

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
	"net/http"

	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ShowBackupHandler ...
// returns a ShowBackupResponse
func ShowBackupHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	vars := mux.Vars(r)

	backupname := vars["name"]

	clientVersion := r.URL.Query().Get("version")
	namespace := r.URL.Query().Get("namespace")

	log.Debugf("ShowBackupHandler parameters name [%s] version [%s] namespace [%s]", backupname, clientVersion, namespace)

	username, err := apiserver.Authn(apiserver.SHOW_BACKUP_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("backupservice.ShowBackupHandler GET called")
	resp := msgs.ShowBackupResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return

	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowBackup(backupname, ns)
	json.NewEncoder(w).Encode(resp)

}

// DeleteBackupHandler ...
// returns a ShowBackupResponse
func DeleteBackupHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	vars := mux.Vars(r)

	backupname := vars["name"]
	clientVersion := r.URL.Query().Get("version")
	namespace := r.URL.Query().Get("namespace")

	log.Debugf("DeleteBackupHandler parameters name [%s] version [%s] namespace [%s]", backupname, clientVersion, namespace)

	username, err := apiserver.Authn(apiserver.DELETE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("backupservice.DeleteBackupHandler called")

	resp := msgs.DeleteBackupResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = DeleteBackup(backupname, ns)
	json.NewEncoder(w).Encode(resp)

}

// CreateBackupHandler ...
// pgo backup all
// pgo backup --selector=name=mycluster
// pgo backup mycluster
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	var ns string
	log.Debug("backupservice.CreateBackupHandler called")

	var request msgs.CreateBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.CreateBackupResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateBackup(&request, ns)

	json.NewEncoder(w).Encode(resp)
}

// RestoreHandler takes a GET request for URL path '/pgbasebackuprestore', calls the required
// business logic to perform a pg_basebackup restore, and then returns the appropriate response
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	log.Debug("backup.RestoreHandler called")

	var request msgs.PgbasebackupRestoreRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.RESTORE_PGBASEBACKUP_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp := msgs.PgbasebackupRestoreResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := Restore(&request, ns)
	json.NewEncoder(w).Encode(resp)
}
