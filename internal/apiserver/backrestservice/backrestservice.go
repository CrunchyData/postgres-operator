package backrestservice

/*
Copyright 2018 - 2023 Crunchy Data Solutions, Inc.
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

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// CreateBackupHandler ...
// pgo backup all
// pgo backup --selector=name=mycluster
// pgo backup mycluster
func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /backrestbackup backrestservice backrestbackup
	/*```
	Performs a backup using pgBackrest
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Create Backrest Backup Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/CreateBackrestBackupRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/CreateBackrestBackupResponse"
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
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = CreateBackup(&request, ns, username)
	_ = json.NewEncoder(w).Encode(resp)
}

// DeleteBackrestHandler deletes a targeted backup from a pgBackRest repository
// pgo delete backup hippo --target=pgbackrest-backup-id
func DeleteBackrestHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /backrest backrestservice
	/*```
	  Delete a pgBackRest backup
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "PostgreSQL Cluster Disk Utilization"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/DeleteBackrestBackupRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/DeleteBackrestBackupResponse"
	log.Debug("backrestservice.DeleteBackrestHandler called")

	// first, check that the requesting user is authorized to make this request
	username, err := apiserver.Authn(apiserver.DELETE_BACKUP_PERM, w, r)
	if err != nil {
		return
	}

	// decode the request paramaeters
	var request msgs.DeleteBackrestBackupRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := msgs.DeleteBackrestBackupResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	log.Debugf("DeleteBackrestHandler parameters [%+v]", request)

	// set some of the header...though we really should not be setting the HTTP
	// Status upfront, but whatever
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// check that the client versions match. If they don't, error out
	if request.ClientVersion != msgs.PGO_VERSION {
		response := msgs.DeleteBackrestBackupResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  apiserver.VERSION_MISMATCH_ERROR,
			},
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	// ensure that the user has access to this namespace. if not, error out
	if _, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace); err != nil {
		response := msgs.DeleteBackrestBackupResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	// process the request
	response := DeleteBackup(request)

	// turn the response into JSON
	_ = json.NewEncoder(w).Encode(response)
}

// ShowBackrestHandler ...
// returns a ShowBackrestResponse
func ShowBackrestHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /backrest/{name} backrestservice backrest-name
	/*```
	Returns a ShowBackrestResponse that provides information about a given backup
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
	//  - name: "selector"
	//    description: "Selector"
	//    in: "path"
	//    type: "string"
	//    required: true
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ShowBackrestResponse"
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
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowBackrest(backupname, selector, ns)
	_ = json.NewEncoder(w).Encode(resp)
}

// RestoreHandler ...
// pgo restore mycluster --to-cluster=restored
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /restore backrestservice restore
	/*```
	Restore a cluster with backrest
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Restore Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/RestoreRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/RestoreResponse"
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
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp = Restore(&request, ns, username)
	_ = json.NewEncoder(w).Encode(resp)
}
