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
	"net/http"
	//	"strconv"

	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// TestResults ...
type TestResults struct {
	Results []string
}

// ClusterDetail ...
type ClusterDetail struct {
	Name string
}

// CreateClusterHandler ...
// pgo create cluster
// parameters secretfrom
func CreateClusterHandler(w http.ResponseWriter, r *http.Request) {
	var ns string

	log.Debug("clusterservice.CreateClusterHandler called")
	username, err := apiserver.Authn(apiserver.CREATE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var request msgs.CreateClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	resp := msgs.CreateClusterResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if request.ClientVersion != msgs.PGO_VERSION {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = apiserver.VERSION_MISMATCH_ERROR
		json.NewEncoder(w).Encode(resp)
		return
	}
	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, request.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}
	resp = CreateCluster(&request, ns)
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
	var ns string

	var request msgs.ShowClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("clusterservice.ShowClusterHandler %v\n", request)
	clustername := request.Clustername

	selector := request.Selector
	ccpimagetag := request.Ccpimagetag
	clientVersion := request.ClientVersion
	namespace := request.Namespace
	allflag := request.AllFlag

	log.Debugf("ShowClusterHandler: parameters name [%s] selector [%s] ccpimagetag [%s] version [%s] namespace [%s] allflag [%s]", clustername, selector, ccpimagetag, clientVersion, namespace, allflag)

	username, err := apiserver.Authn(apiserver.SHOW_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("clusterservice.ShowClusterHandler GET called")

	var resp msgs.ShowClusterResponse
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]msgs.ShowClusterDetail, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		resp.Results = make([]msgs.ShowClusterDetail, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = ShowCluster(clustername, selector, ccpimagetag, ns, allflag)
	json.NewEncoder(w).Encode(resp)

}

// DeleteClusterHandler ...
// pgo delete mycluster
// parameters showsecrets
// parameters selector
// parameters postgresversion
// returns a ShowClusterResponse
func DeleteClusterHandler(w http.ResponseWriter, r *http.Request) {
	var request msgs.DeleteClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	var ns string
	log.Debugf("clusterservice.DeleteClusterHandler %v\n", request)

	clustername := request.Clustername

	selector := request.Selector
	clientVersion := request.ClientVersion
	namespace := request.Namespace

	deleteData := request.DeleteData
	deleteBackups := request.DeleteBackups

	log.Debugf("DeleteClusterHandler: parameters namespace [%s] selector [%s] delete-data [%t] delete-backups [%t]", namespace, selector, clientVersion, deleteData, deleteBackups)

	username, err := apiserver.Authn(apiserver.DELETE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("clusterservice.DeleteClusterHandler called")

	resp := msgs.DeleteClusterResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]string, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}
	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		resp.Results = make([]string, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}
	resp = DeleteCluster(clustername, selector, deleteData, deleteBackups, ns)
	json.NewEncoder(w).Encode(resp)

}

// TestClusterHandler ...
// pgo test mycluster
func TestClusterHandler(w http.ResponseWriter, r *http.Request) {

	var request msgs.ClusterTestRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("clusterservice.TestClusterHandler %v\n", request)

	var ns string
	clustername := request.Clustername

	selector := request.Selector
	namespace := request.Namespace
	clientVersion := request.ClientVersion

	log.Debugf("TestClusterHandler parameters %v", request)

	username, err := apiserver.Authn(apiserver.TEST_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := msgs.ClusterTestResponse{}
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

	resp = TestCluster(clustername, selector, ns, request.AllFlag)
	json.NewEncoder(w).Encode(resp)
}

// UpdateClusterHandler ...
// pgo update cluster mycluster --autofail=true
// pgo update cluster --selector=env=research --autofail=false
// returns a UpdateClusterResponse
func UpdateClusterHandler(w http.ResponseWriter, r *http.Request) {
	var ns string
	vars := mux.Vars(r)

	clustername := vars["name"]

	selector := r.URL.Query().Get("selector")
	namespace := r.URL.Query().Get("namespace")
	clientVersion := r.URL.Query().Get("version")

	autofailStr := r.URL.Query().Get("autofail")

	log.Debugf("UpdateClusterHandler parameters name [%s] version [%s] selector [%s] namespace [%s] autofail [%s]", clustername, clientVersion, selector, namespace, autofailStr)

	username, err := apiserver.Authn(apiserver.UPDATE_CLUSTER_PERM, w, r)
	if err != nil {
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.Debug("clusterservice.UpdateClusterHandler called")

	resp := msgs.UpdateClusterResponse{}
	resp.Status = msgs.Status{Code: msgs.Ok, Msg: ""}

	if clientVersion != msgs.PGO_VERSION {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: apiserver.VERSION_MISMATCH_ERROR}
		resp.Results = make([]string, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	ns, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		resp.Results = make([]string, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if autofailStr != "" {
		if autofailStr == "true" || autofailStr == "false" {
		} else {
			resp.Status = msgs.Status{
				Code: msgs.Error,
				Msg:  "autofail parameter is not true or false, boolean is required"}
			resp.Results = make([]string, 0)
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	resp = UpdateCluster(clustername, selector, autofailStr, ns)
	json.NewEncoder(w).Encode(resp)

}
