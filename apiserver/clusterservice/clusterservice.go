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

	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	log "github.com/sirupsen/logrus"
)

// CreateClusterHandler ...
// pgo create cluster
// parameters secretfrom
func CreateClusterHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /clusters clusterservice clusters
	/*```
	  Create a PostgreSQL cluster consisting of a primary and a number of replica backends
	*/
	// ---
	//	Produces:
	//	- application/json
	//
	//	parameters:
	//	- name: "Cluster Create Request"
	//	  in: "body"
	//	  schema:
	//	    "$ref": "#/definitions/CreateClusterRequest"
	//	responses:
	//	  '200':
	//	    description: Output
	//	    schema:
	//	      "$ref": "#/definitions/CreateClusterResponse"

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
	resp = CreateCluster(&request, ns, username)
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
	// swagger:operation POST /showclusters clusterservice showclusters
	/*```
	  Show a PostgreSQL cluster
	*/
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: "Show Cluster Request"
	//   in: "body"
	//   schema:
	//     "$ref": "#/definitions/ShowClusterRequest"
	//	responses:
	//	  '200':
	//	    description: Output
	//	    schema:
	//	      "$ref": "#/definitions/ShowClusterResponse"
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
	// swagger:operation POST /clustersdelete clusterservice clustersdelete
	/*```
	  Delete a PostgreSQL cluster
	*/
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: "Delete Cluster Request"
	//   in: "body"
	//   schema:
	//     "$ref": "#/definitions/DeleteClusterRequest"
	//	responses:
	//	  '200':
	//	    description: Output
	//	    schema:
	//	      "$ref": "#/definitions/DeleteClusterResponse"
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
	resp = DeleteCluster(clustername, selector, deleteData, deleteBackups, ns, username)
	json.NewEncoder(w).Encode(resp)

}

// TestClusterHandler ...
// pgo test mycluster
func TestClusterHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /testclusters clusterservice testclusters
	/*```
	TEST allows you to test the connectivity for a cluster.

	If you set the AllFlag to true in the request it will test connectivity for all clusters in the namespace.
	*/
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: "Cluster Test Request"
	//   in: "body"
	//   schema:
	//     "$ref": "#/definitions/ClusterTestRequest"
	//	responses:
	//	  '200':
	//	    description: Output
	//	    schema:
	//	      "$ref": "#/definitions/ClusterTestResponse"
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

	resp = TestCluster(clustername, selector, ns, username, request.AllFlag)
	json.NewEncoder(w).Encode(resp)
}

// UpdateClusterHandler ...
// pgo update cluster mycluster --autofail=true
// pgo update cluster --selector=env=research --autofail=false
// returns a UpdateClusterResponse
func UpdateClusterHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /clustersupdate clusterservice clustersupdate
	/*```
	  Update a PostgreSQL cluster
	*/
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: "Update Request"
	//   in: "body"
	//   schema:
	//     "$ref": "#/definitions/UpdateClusterRequest"
	//	responses:
	//	  '200':
	//	    description: Output
	//	    schema:
	//	      "$ref": "#/definitions/UpdateClusterResponse"
	var request msgs.UpdateClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Debugf("clusterservice.UpdateClusterHandler %v\n", request)

	namespace := request.Namespace
	clientVersion := request.ClientVersion

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

	_, err = apiserver.GetNamespace(apiserver.Clientset, username, namespace)
	if err != nil {
		resp.Status = msgs.Status{Code: msgs.Error, Msg: err.Error()}
		resp.Results = make([]string, 0)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp = UpdateCluster(&request)
	json.NewEncoder(w).Encode(resp)

}
