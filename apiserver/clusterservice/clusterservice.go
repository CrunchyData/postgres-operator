package clusterservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type TestResults struct {
	Results []string
}

type ClusterDetail struct {
	Name string
	//deployments
	//replicasets
	//pods
	//services
	//secrets
}
type ShowClusterResponse struct {
	Items []ClusterDetail
}

type CreateClusterRequest struct {
	Name string
}

// pgo create cluster
// parameters secretfrom
func CreateClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("clusterservice.CreateClusterHandler called")
	var request CreateClusterRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("clusterservice.CreateClusterHandler got request " + request.Name)
}

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
