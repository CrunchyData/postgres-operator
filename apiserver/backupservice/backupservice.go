package backupservice

/*
Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	// swagger:operation GET /backups/{name} backupservice backups-name
	/*```
	Show a pgbasebackup using the backup name
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "name"
	//    description: "Backup Name"
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
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowBackupResponse"
	log.Warn("DEPRECRATED: Please use pgbackrest")
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
	// swagger:operation GET /backupsdelete/{name} backupservice backupsdelete-name
	/*```
	  Delete a backup taken with pgbasebackup
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "name"
	//    description: "Backup name"
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
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeleteBackupResponse"
	log.Warn("DEPRECRATED: Please use pgbackrest")
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
	// swagger:operation POST /backups backupservice backups
	/*```
	  Create a backup with pgbasebackup
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create Backup Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreateBackupRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreateBackupResponse"
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

	resp = CreateBackup(&request, ns, username)

	json.NewEncoder(w).Encode(resp)
}

// RestoreHandler takes a GET request for URL path '/pgbasebackuprestore', calls the required
// business logic to perform a pg_basebackup restore, and then returns the appropriate response
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pgbasebackuprestore backupservice pgbasebackuprestore
	/*```
	  Restore a cluster using pgbasebackup
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "pgbasebackup restore Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/PgbasebackupRestoreRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/PgbasebackupRestoreResponse"
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
