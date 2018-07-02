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
	"github.com/gorilla/mux"
	"net/http"
)

// ScaleClusterHandler ...
// pgo scale mycluster --replica-count=1
// parameters showsecrets
// returns a ScaleResponse
func ScaleClusterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("clusterservice.ScaleClusterHandler %v\n", vars)

	clusterName := vars["name"]
	log.Debugf(" clusterName arg is %v\n", clusterName)

	replicaCount := r.URL.Query().Get("replica-count")
	if replicaCount != "" {
		log.Debug("replica-count param was [" + replicaCount + "]")
	}
	resourcesConfig := r.URL.Query().Get("resources-config")
	if resourcesConfig != "" {
		log.Debug("resources-config param was [" + resourcesConfig + "]")
	}
	storageConfig := r.URL.Query().Get("storage-config")
	if storageConfig != "" {
		log.Debug("storage-config param was [" + storageConfig + "]")
	}
	nodeLabel := r.URL.Query().Get("node-label")
	if nodeLabel != "" {
		log.Debug("node-label param was [" + nodeLabel + "]")
	}
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
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
		resp = ScaleCluster(clusterName, replicaCount, resourcesConfig, storageConfig, nodeLabel)
	}

	json.NewEncoder(w).Encode(resp)
}
