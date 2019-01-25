package clusterservice

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	"github.com/crunchydata/postgres-operator/util"
	"github.com/gorilla/mux"
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
	log.Debugf("clusterservice.ScaleClusterHandler %v\n", vars)

	clusterName := vars[util.LABEL_NAME]
	log.Debugf("clusterName argument is %v\n", clusterName)

	replicaCount := r.URL.Query().Get(util.LABEL_REPLICA_COUNT)
	if replicaCount != "" {
		log.Debugf("replica-count parameter is [%s]", replicaCount)
	}
	resourcesConfig := r.URL.Query().Get(util.LABEL_RESOURCES_CONFIG)
	if resourcesConfig != "" {
		log.Debugf("resources-config parameter is [%s]", resourcesConfig)
	}
	storageConfig := r.URL.Query().Get(util.LABEL_STORAGE_CONFIG)
	if storageConfig != "" {
		log.Debugf("storage-config parameter is [%s]", storageConfig)
	}
	nodeLabel := r.URL.Query().Get(util.LABEL_NODE_LABEL)
	if nodeLabel != "" {
		log.Debugf("node-label parameter is [%s]", nodeLabel)
	}
	serviceType := r.URL.Query().Get(util.LABEL_SERVICE_TYPE)
	if serviceType != "" {
		log.Debugf("service-type parameter is [%s]", serviceType)
	}
	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debugf("version parameter is [%s]", clientVersion)
	}
	ccpImageTag := r.URL.Query().Get(util.LABEL_CCP_IMAGE_TAG_KEY)
	if ccpImageTag != "" {
		log.Debugf("ccp-image-tag parameter is [%s]", ccpImageTag)
	}

	switch r.Method {
	case "GET":
		log.Debug("clusterservice.ScaleClusterHandler GET called")
	}

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

	ns, err = apiserver.GetNamespace(username, "")
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
	log.Debugf("clusterservice.ScaleQueryHandler %v\n", vars)

	clusterName := vars[util.LABEL_NAME]
	log.Debugf(" clusterName argument is %v\n", clusterName)
	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debugf("version parameter is [%s]", clientVersion)
	}

	switch r.Method {
	case "GET":
		log.Debug("clusterservice.ScaleQueryHandler GET called")
	}

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

	ns, err = apiserver.GetNamespace(username, "")
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
	log.Debugf("clusterservice.ScaleDownHandler %v\n", vars)

	clusterName := vars[util.LABEL_NAME]
	log.Debugf("clusterName argument is %v\n", clusterName)
	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debugf("version parameter is [%s]", clientVersion)
	}
	replicaName := r.URL.Query().Get(util.LABEL_REPLICA_NAME)
	if replicaName != "" {
		log.Debugf("replicaName parameter is [%s]", replicaName)
	}
	tmp := r.URL.Query().Get(util.LABEL_DELETE_DATA)
	if tmp != "" {
		log.Debugf("delete-data parameter is [%s]", tmp)
	}

	switch r.Method {
	case "GET":
		log.Debug("clusterservice.ScaleDownHandler GET called")
	}

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

	ns, err = apiserver.GetNamespace(username, "")
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ScaleDown(deleteData, clusterName, replicaName, ns)
	json.NewEncoder(w).Encode(resp)
}
