package clusterservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	vars := mux.Vars(r)
	log.Debugf("clusterservice.ScaleClusterHandler %v\n", vars)

	clusterName := vars[util.LABEL_NAME]
	log.Debugf(" clusterName arg is %v\n", clusterName)

	replicaCount := r.URL.Query().Get(util.LABEL_REPLICA_COUNT)
	if replicaCount != "" {
		log.Debug("replica-count param was [" + replicaCount + "]")
	}
	resourcesConfig := r.URL.Query().Get(util.LABEL_RESOURCES_CONFIG)
	if resourcesConfig != "" {
		log.Debug("resources-config param was [" + resourcesConfig + "]")
	}
	storageConfig := r.URL.Query().Get(util.LABEL_STORAGE_CONFIG)
	if storageConfig != "" {
		log.Debug("storage-config param was [" + storageConfig + "]")
	}
	nodeLabel := r.URL.Query().Get(util.LABEL_NODE_LABEL)
	if nodeLabel != "" {
		log.Debug("node-label param was [" + nodeLabel + "]")
	}
	serviceType := r.URL.Query().Get(util.LABEL_SERVICE_TYPE)
	if serviceType != "" {
		log.Debug("service-type param was [" + serviceType + "]")
	}
	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}
	ccpImageTag := r.URL.Query().Get(util.LABEL_CCP_IMAGE_TAG_KEY)
	if ccpImageTag != "" {
		log.Debug("ccp-image-tag param was [" + ccpImageTag + "]")
	}

	switch r.Method {
	case "GET":
		log.Debug("clusterservice.ScaleClusterHandler GET called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var resp msgs.ClusterScaleResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ClusterScaleResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = ScaleCluster(clusterName, replicaCount, resourcesConfig, storageConfig, nodeLabel, ccpImageTag, serviceType)
	}

	json.NewEncoder(w).Encode(resp)
}

// ScaleQueryHandler ...
// pgo scale mycluster --query
// returns a ScaleQueryResponse
func ScaleQueryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("clusterservice.ScaleQueryHandler %v\n", vars)

	clusterName := vars[util.LABEL_NAME]
	log.Debugf(" clusterName arg is %v\n", clusterName)
	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	switch r.Method {
	case "GET":
		log.Debug("clusterservice.ScaleQueryHandler GET called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var resp msgs.ScaleQueryResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ScaleQueryResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = ScaleQuery(clusterName)
	}

	json.NewEncoder(w).Encode(resp)
}

// ScaleDownHandler ...
// pgo scale mycluster --scale-down-target=somereplicaname
// returns a ScaleDownResponse
func ScaleDownHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("clusterservice.ScaleDownHandler %v\n", vars)

	clusterName := vars[util.LABEL_NAME]
	log.Debugf(" clusterName arg is %v\n", clusterName)
	clientVersion := r.URL.Query().Get(util.LABEL_VERSION)
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}
	replicaName := r.URL.Query().Get(util.LABEL_REPLICA_NAME)
	if replicaName != "" {
		log.Debug("replicaName param was [" + replicaName + "]")
	}
	tmp := r.URL.Query().Get(util.LABEL_DELETE_DATA)
	if tmp != "" {
		log.Debug("delete-data param was [" + tmp + "]")
	}

	switch r.Method {
	case "GET":
		log.Debug("clusterservice.ScaleDownHandler GET called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var resp msgs.ScaleDownResponse
	deleteData, err := strconv.ParseBool(tmp)
	if err != nil {
		resp = msgs.ScaleDownResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
	} else if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ScaleDownResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = ScaleDown(deleteData, clusterName, replicaName)
	}

	json.NewEncoder(w).Encode(resp)
}
