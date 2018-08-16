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
	"strconv"
)

// TestResults ...
type TestResults struct {
	Results []string
}

// ClusterDetail ...
type ClusterDetail struct {
	Name string
	//deployments
	//replicasets
	//pods
	//services
	//secrets
}

// CreateClusterHandler ...
// pgo create cluster
// parameters secretfrom
func CreateClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("clusterservice.CreateClusterHandler called")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := apiserver.Authn(apiserver.CREATE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	var request msgs.CreateClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreateClusterResponse{}
	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
	} else {
		resp = CreateCluster(&request)
	}
	json.NewEncoder(w).Encode(resp)

}

// ShowClusterHandler ...
// pgo show cluster
// pgo delete mycluster
// parameters showsecrets
// parameters selector
// parameters postgresversion
// returns a ShowClusterResponse
func ShowClusterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("clusterservice.ShowClusterHandler %v\n", vars)

	clustername := vars["name"]

	selector := r.URL.Query().Get("selector")
	if selector != "" {
		log.Debug("selector param was [" + selector + "]")
	}
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err := apiserver.Authn(apiserver.SHOW_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	log.Debug("clusterservice.ShowClusterHandler GET called")

	var resp msgs.ShowClusterResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ShowClusterResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]msgs.ShowClusterDetail, 0)
	} else {
		resp = ShowCluster(clustername, selector)
	}
	json.NewEncoder(w).Encode(resp)

}

// DeleteClusterHandler ...
// pgo delete mycluster
// parameters showsecrets
// parameters selector
// parameters postgresversion
// returns a ShowClusterResponse
func DeleteClusterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debugf("clusterservice.DeleteClusterHandler %v\n", vars)

	clustername := vars["name"]

	selector := r.URL.Query().Get("selector")
	if selector != "" {
		log.Debug("selector param was [" + selector + "]")
	}
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	deleteData := false
	deleteDataStr := r.URL.Query().Get("delete-data")
	if deleteDataStr != "" {
		log.Debug("delete-data param was [" + deleteDataStr + "]")
		deleteData, _ = strconv.ParseBool(deleteDataStr)
	}
	deleteBackups := false
	deleteBackupsStr := r.URL.Query().Get("delete-backups")
	if deleteDataStr != "" {
		log.Debug("delete-backups param was [" + deleteBackupsStr + "]")
		deleteBackups, _ = strconv.ParseBool(deleteBackupsStr)
	}

	err := apiserver.Authn(apiserver.DELETE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	log.Debug("clusterservice.DeleteClusterHandler called")

	var resp msgs.DeleteClusterResponse
	if clientVersion != msgs.PGO_VERSION {
		resp := msgs.DeleteClusterResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]string, 0)
	} else {
		resp = DeleteCluster(clustername, selector, deleteData, deleteBackups)
	}
	json.NewEncoder(w).Encode(resp)

}

// TestClusterHandler ...
// pgo test mycluster
func TestClusterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Debug("clusterservice.TestClusterHandler %v\n", vars)
	clustername := vars["name"]

	selector := r.URL.Query().Get("selector")
	if selector != "" {
		log.Debug("selector param was [" + selector + "]")
	}
	clientVersion := r.URL.Query().Get("version")
	if clientVersion != "" {
		log.Debug("version param was [" + clientVersion + "]")
	}

	err := apiserver.Authn(apiserver.TEST_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	var resp msgs.ClusterTestResponse
	if clientVersion != msgs.PGO_VERSION {
		resp = msgs.ClusterTestResponse{}
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
	} else {
		resp = TestCluster(clustername, selector)
	}

	json.NewEncoder(w).Encode(resp)
}
