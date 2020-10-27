package restartservice

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// RestartHandler handles requests to the "restart" endpoint.
// pgo restart mycluster
// pgo restart mycluster --target=mycluster-abcd
func RestartHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /restart restartservice restart
	/*```
	RESTART performs a PostgreSQL restart on a PostgreSQL cluster.  If no targets are specified,
	then all instances (the primary and all replicas) within the cluster will be restarted.
	Otherwise, only those targets specified will be restarted.
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "Restart Request"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/RestartRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/RestartResponse"

	log.Debug("restartservice.RestartHandler called")

	resp := msgs.RestartResponse{}

	var request msgs.RestartRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	username, err := apiserver.Authn(apiserver.RESTART_PERM, w, r)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
	}

	if _, err := apiserver.GetNamespace(apiserver.Clientset, username,
		request.Namespace); err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	json.NewEncoder(w).Encode(Restart(&request, username))
}

// QueryRestartHandler handles requests to query a cluster for instances available to use as
// as targets for a PostgreSQL restart.
// pgo restart mycluster --query
func QueryRestartHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /restart/{name} restartservice restart-service
	/*```
	  Prints the list of restart candidates.
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
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/QueryRestartResponse"

	resp := msgs.QueryRestartResponse{}

	clusterName := mux.Vars(r)["name"]
	clientVersion := r.URL.Query().Get("version")
	namespace := r.URL.Query().Get("namespace")

	log.Debugf("QueryRestartHandler parameters version[%s] namespace [%s] name [%s]", clientVersion,
		namespace, clusterName)

	username, err := apiserver.Authn(apiserver.RESTART_PERM, w, r)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		json.NewEncoder(w).Encode(resp)
	}

	if _, err := apiserver.GetNamespace(apiserver.Clientset, username, namespace); err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	json.NewEncoder(w).Encode(QueryRestart(clusterName, namespace))
}
