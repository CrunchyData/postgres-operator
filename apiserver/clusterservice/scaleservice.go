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
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
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
	//  - name: "replica-count"
	//    description: "The replica count to apply to the clusters."
	//    in: "path"
	//    type: "int"
	//    required: true
	//  - name: "resources-config"
	//    description: "The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits."
	//    in: "path"
	//    type: "string"
	//    required: false
	//  - name: "storage-config"
	//    description: "The service type to use in the replica Service. If not set, the default in pgo.yaml will be used."
	//    in: "path"
	//    type: "string"
	//    required: false
	//  - name: "node-label"
	//    description: "The node label (key) to use in placing the replica database. If not set, any node is used."
	//    in: "path"
	//    type: "string"
	//    required: false
	//  - name: "service-type"
	//    description: "The service type to use in the replica Service. If not set, the default in pgo.yaml will be used."
	//    in: "path"
	//    type: "string"
	//    required: false
	//  - name: "ccp-image-tag"
	//    description: "The CCPImageTag to use for cluster creation. If specified, overrides the .pgo.yaml setting."
	//    in: "path"
	//    type: "string"
	//    required: false
	//  responses:
	//    '200':
	//      description: Output
	//      schema:
	//        "$ref": "#/definitions/ClusterScaleResponse"
	//SCALE_CLUSTER_PERM
	// This is a pain to document because it doesn't use a struct...
	var ns string
	vars := mux.Vars(r)

	clusterName := vars[config.LABEL_NAME]

	namespace := r.URL.Query().Get(config.LABEL_NAMESPACE)
	replicaCount := r.URL.Query().Get(config.LABEL_REPLICA_COUNT)
	resourcesConfig := r.URL.Query().Get(config.LABEL_RESOURCES_CONFIG)
	storageConfig := r.URL.Query().Get(config.LABEL_STORAGE_CONFIG)
	nodeLabel := r.URL.Query().Get(config.LABEL_NODE_LABEL)
	serviceType := r.URL.Query().Get(config.LABEL_SERVICE_TYPE)
	clientVersion := r.URL.Query().Get(config.LABEL_VERSION)
	ccpImageTag := r.URL.Query().Get(config.LABEL_CCP_IMAGE_TAG_KEY)

	log.Debugf("ScaleClusterHandler parameters name [%s] namespace [%s] replica-count [%s] resources-config [%s] storage-config [%s] node-label [%s] service-type [%s] version [%s] ccp-image-tag [%s]", clusterName, namespace, replicaCount, resourcesConfig, storageConfig, nodeLabel, serviceType, clientVersion, ccpImageTag)

	username, err := apiserver.Authn(apiserver.SCALE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ClusterScaleResponse{}
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

	resp = ScaleCluster(clusterName, replicaCount, resourcesConfig, storageConfig, nodeLabel, ccpImageTag, serviceType, ns, username)

	json.NewEncoder(w).Encode(resp)
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
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ScaleQuery(clusterName, ns)
	json.NewEncoder(w).Encode(resp)
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
		json.NewEncoder(w).Encode(resp)
		return
	}

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

	resp = ScaleDown(deleteData, clusterName, replicaName, ns)
	json.NewEncoder(w).Encode(resp)
}
