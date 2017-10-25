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
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
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
	log.Infoln("clusterservice.CreateClusterHandler called")
	var request msgs.CreateClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.CreateClusterResponse{}
	resp = CreateCluster(&request)
	json.NewEncoder(w).Encode(resp)

}

// ShowClusterHandler ...
// pgo show cluster
// pgo delete mycluster
// parameters showsecrets
// parameters selector
// parameters namespace
// parameters postgresversion
// returns a ShowClusterResponse
func ShowClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("clusterservice.ShowClusterHandler called")
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	clustername := vars["name"]
	log.Infof(" name arg is %v\n", clustername)

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Infoln("namespace param was [" + namespace + "]")
	} else {
		log.Infoln("namespace param was null")
	}
	selector := r.URL.Query().Get("selector")
	if namespace != "" {
		log.Infoln("selector param was [" + selector + "]")
	} else {
		log.Infoln("selector param was null")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		log.Infoln("clusterservice.ShowClusterHandler GET called")
		resp := ShowCluster(namespace, clustername, selector)
		json.NewEncoder(w).Encode(resp)
	case "DELETE":
		log.Infoln("clusterservice.DeleteClusterHandler DELETE called")
		resp := DeleteCluster(namespace, clustername, selector)
		json.NewEncoder(w).Encode(resp)
	}

}

// TestClusterHandler ...
// pgo test mycluster
func TestClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("clusterservice.TestClusterHandler called")
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	clustername := vars["name"]
	log.Infof(" name arg is %v\n", clustername)
	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Infoln("namespace param was [" + namespace + "]")
	} else {
		log.Infoln("namespace param was null")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := TestCluster(namespace, clustername)

	json.NewEncoder(w).Encode(resp)
}
