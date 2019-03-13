package backrestservice

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
	"github.com/crunchydata/postgres-operator/config"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// CreateBackupHandler ...
// pgo backup all
// pgo backup --selector=name=mycluster
// pgo backup mycluster
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	var ns string
	log.Debug("backrestservice.CreateBackupHandler called")

	var request msgs.CreateBackrestBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.CreateBackrestBackupResponse{}
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

// ShowBackrestHandler ...
// returns a ShowBackrestResponse
func ShowBackrestHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	vars := mux.Vars(r)

	backupname := vars[config.LABEL_NAME]

	clientVersion := r.URL.Query().Get(config.LABEL_VERSION)
	selector := r.URL.Query().Get(config.LABEL_SELECTOR)
	namespace := r.URL.Query().Get(config.LABEL_NAMESPACE)

	log.Debugf("ShowBackrestHandler parameters name [%s] version [%s] selector [%s] namespace [%s]", backupname, clientVersion, selector, namespace)

	username, err := apiserver.Authn(apiserver.SHOW_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("backrestservice.ShowBackrestHandler GET called")
	resp := msgs.ShowBackrestResponse{}

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

	resp = ShowBackrest(backupname, selector, ns)
	json.NewEncoder(w).Encode(resp)

}

// RestoreHandler ...
// pgo restore mycluster --to-cluster=restored
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	log.Debug("backrestservice.RestoreHandler called")

	var request msgs.RestoreRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.RESTORE_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.RestoreResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = Restore(&request, ns)
	json.NewEncoder(w).Encode(resp)
}
