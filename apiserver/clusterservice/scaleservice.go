package clusterservice

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
	"strconv"
)

// ScaleClusterHandler ...
// pgo scale mycluster --replica-count=1
// parameters showsecrets
// returns a ScaleResponse
func ScaleClusterHandler(w http.ResponseWriter, r *http.Request) {
	//SCALE_CLUSTER_PERM
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

	resp = ScaleCluster(clusterName, replicaCount, resourcesConfig, storageConfig, nodeLabel, ccpImageTag, serviceType, ns)

	json.NewEncoder(w).Encode(resp)
}

// ScaleQueryHandler ...
// pgo scale mycluster --query
// returns a ScaleQueryResponse
func ScaleQueryHandler(w http.ResponseWriter, r *http.Request) {
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
