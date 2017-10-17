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

// ShowClusterResponse ...
type ShowClusterResponse struct {
	Items []ClusterDetail
}

// CreateClusterRequest ...
type CreateClusterRequest struct {
	Name string
}

// CreateClusterHandler ...
// pgo create cluster
// parameters secretfrom
func CreateClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("clusterservice.CreateClusterHandler called")
	var request CreateClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("clusterservice.CreateClusterHandler got request " + request.Name)
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
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	switch r.Method {
	case "GET":
		log.Infoln("clusterservice.ShowClusterHandler GET called")
	case "DELETE":
		log.Infoln("clusterservice.ShowClusterHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := new(ShowClusterResponse)
	resp.Items = []ClusterDetail{}
	c := ClusterDetail{}
	c.Name = "somecluster"
	resp.Items = append(resp.Items, c)

	json.NewEncoder(w).Encode(resp)
}

// TestClusterHandler ...
// pgo test mycluster
func TestClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("clusterservice.TestClusterHandler called")
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	c := new(TestResults)
	c.Results = []string{"one", "two"}
	json.NewEncoder(w).Encode(c)
}
