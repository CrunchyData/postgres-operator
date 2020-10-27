package pgdumpservice

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// BackupHandler ...
// pgo backup --backup-type=pgdump mycluster
func BackupHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgdumpbackup pgdumpservice pgdumpbackup
	/*```
	Backup a cluster using pgdump
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create pgDump Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreatepgDumpBackupRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreatepgDumpBackupResponse"
	var ns string
	log.Debug("pgdumpservice.CreatepgDumpHandlerBackupHandler called")

	var request msgs.CreatepgDumpBackupRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.CREATE_DUMP_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.CreatepgDumpBackupResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreatepgDump(&request, ns)
	json.NewEncoder(w).Encode(resp)
}

// ShowpgDumpHandler ...
// returns a ShowpgDumpResponse
func ShowDumpHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pgdump/{name} pgdumpservice pgdump-name
	/*```
	Show backups taken with pgdump
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "name"
	//    description: "Cluster Name"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  - name: "version"
	//    description: "Client Version"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  - name: "namespace"
	//    description: "Namespace"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  - name: "selector"
	//    description: "Selector"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowBackupResponse"
	var ns string
	vars := mux.Vars(r)

	clustername := vars[config.LABEL_NAME]

	clientVersion := r.URL.Query().Get(config.LABEL_VERSION)
	namespace := r.URL.Query().Get(config.LABEL_NAMESPACE)
	selector := r.URL.Query().Get(config.LABEL_SELECTOR)

	log.Debugf("ShowDumpHandler parameters version [%s] namespace [%s] selector [%s] name [%s]", clientVersion, namespace, selector, clustername)

	username, err := apiserver.Authn(apiserver.SHOW_BACKUP_PERM, w, r)
	if err != nil {
		return
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("pgdumpservice.pgdumpHandler GET called")
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

	resp = ShowpgDump(clustername, selector, ns)
	json.NewEncoder(w).Encode(resp)

}

// RestoreHandler ...
// pgo restore mycluster --restore-type=pgdump --to-cluster=restored
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /restore pgdumpservice restore
	/*```
	Restore a cluster with pgrestore. This endpoint is used to restore backups taken with pgdump
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Restore Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/PgRestoreRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/PgRestoreResponse"
	var ns string

	log.Debug("pgdumpservice.RestoreHandler called")

	var request msgs.PgRestoreRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	username, err := apiserver.Authn(apiserver.RESTORE_DUMP_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.PgRestoreResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = Restore(&request, ns)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
