package clusterservice

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
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
	"github.com/crunchydata/postgres-operator/internal/config"
	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ScaleClusterHandler ...
// pgo scale mycluster --replica-count=1
// parameters showsecrets
// returns a ScaleResponse
func ScaleClusterHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /clusters/scale/{name} clusterservice clusters-scale-name
	/*```
	The scale command allows you to adjust a Cluster's replica configuration
	*/
	// ---
	//  produces:
	//  - application/json
	//  parameters:
	//  - name: "PostgreSQL Scale Cluster"
	//    in: "body"
	//    schema:
	//      "$ref": "#/definitions/ClusterScaleRequest"
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ClusterScaleResponse"
	log.Debug("clusterservice.ScaleClusterHandler called")

	// first, check that the requesting user is authorized to make this request
	username, err := apiserver.Authn(apiserver.SCALE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	// decode the request parameters
	request := msgs.ClusterScaleRequest{}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		_ = json.NewEncoder(w).Encode(msgs.ClusterScaleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		})
		return
	}

	// set some of the header...though we really should not be setting the HTTP
	// Status upfront, but whatever
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// determine if this is the correct client version
	if request.ClientVersion != msgs.PGO_VERSION {
		_ = json.NewEncoder(w).Encode(msgs.ClusterScaleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  apiserver.VERSION_MISMATCH_ERROR,
			},
		})
		return
	}

	// ensure that the user has access to this namespace. if not, error out
	if _, err := apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace); err != nil {
		_ = json.NewEncoder(w).Encode(msgs.ClusterScaleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  err.Error(),
			},
		})
		return
	}

	// ensure that the cluster name is set in the URL, as the request parameters
	// will use that as precedence
	vars := mux.Vars(r)
	clusterName, ok := vars[config.LABEL_NAME]

	if !ok {
		_ = json.NewEncoder(w).Encode(msgs.ClusterScaleResponse{
			Status: msgs.Status{
				Code: msgs.Error,
				Msg:  "cluster name required in URL",
			},
		})
		return
	}

	request.Name = clusterName

	response := ScaleCluster(request, username)

	_ = json.NewEncoder(w).Encode(response)
}

// ScaleQueryHandler ...
// pgo scale mycluster --query
// returns a ScaleQueryResponse
func ScaleQueryHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /scale/{name} clusterservice scale-name
	/*```
	Provides the list of targetable replica candidates for scaledown.
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
	//        "$ref": "#/definitions/ScaleQueryResponse"
	//SCALE_CLUSTER_PERM
	var ns string
	vars := mux.Vars(r)

	clusterName := vars[config.LABEL_NAME]
	clientVersion := r.URL.Query().Get(config.LABEL_VERSION)
	namespace := r.URL.Query().Get(config.LABEL_NAMESPACE)

	log.Debugf("ScaleQueryHandler parameters clusterName [%v] version [%s] namespace [%s]", clusterName, clientVersion, namespace)

	username, err := apiserver.Authn(apiserver.SCALE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ScaleQueryResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

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

	resp = ScaleQuery(clusterName, ns)
	_ = json.NewEncoder(w).Encode(resp)
}

// ScaleDownHandler ...
// pgo scale mycluster --scale-down-target=somereplicaname
// returns a ScaleDownResponse
func ScaleDownHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /scaledown/{name} clusterservice scaledown-name
	/*```
	Scale down a cluster by removing the given replica
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
	//  - name: "replica-name"
	//    description: "The replica to target for scaling down."
	//    in: "path"
	//    type: "string"
	//    required: true
	//  - name: "delete-data"
	//    description: "Causes the data for the scaled down replica to be removed permanently."
	//    in: "path"
	//    type: "string"
	//    required: true
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ScaleDownResponse"
	//SCALE_CLUSTER_PERM
	var ns string
	vars := mux.Vars(r)

	clusterName := vars[config.LABEL_NAME]
	clientVersion := r.URL.Query().Get(config.LABEL_VERSION)
	namespace := r.URL.Query().Get(config.LABEL_NAMESPACE)
	replicaName := r.URL.Query().Get(config.LABEL_REPLICA_NAME)
	tmp := r.URL.Query().Get(config.LABEL_DELETE_DATA)

	log.Debugf("ScaleDownHandler parameters clusterName [%s] version [%s] namespace [%s] replica-name [%s] delete-data [%s]", clusterName, clientVersion, namespace, replicaName, tmp)

	username, err := apiserver.Authn(apiserver.SCALE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ScaleDownResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	deleteData, err := strconv.ParseBool(tmp)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

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

	resp = ScaleDown(deleteData, clusterName, replicaName, ns)
	_ = json.NewEncoder(w).Encode(resp)
}
