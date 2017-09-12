package policyservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	//"github.com/crunchydata/postgres-operator/tpr"
	"github.com/gorilla/mux"
	"net/http"
)

type ApplyResults struct {
	Results []string
}

type PolicyDetail struct {
	Name string
	//deployments
	//replicasets
	//pods
	//services
	//secrets
}
type ShowPolicyResponse struct {
	Items []PolicyDetail
}

type CreatePolicyRequest struct {
	Name string
}

// pgo create policy
// parameters secretfrom
func CreatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("policyservice.CreatePolicyHandler called")
	var request CreatePolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("policyservice.CreatePolicyHandler got request " + request.Name)
}

// pgo show policy
// pgo delete mypolicy
// parameters showsecrets
// parameters selector
// parameters namespace
// parameters postgresversion
// returns a ShowPolicyResponse
func ShowPolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("policyservice.ShowPolicyHandler called")
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	switch r.Method {
	case "GET":
		log.Infoln("policyservice.ShowPolicyHandler GET called")
	case "DELETE":
		log.Infoln("policyservice.ShowPolicyHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := new(ShowPolicyResponse)
	resp.Items = []PolicyDetail{}
	c := PolicyDetail{}
	c.Name = "somepolicy"
	resp.Items = append(resp.Items, c)

	json.NewEncoder(w).Encode(resp)
}

// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("policyservice.ApplyPolicyHandler called")
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	c := new(ApplyResults)
	c.Results = []string{"one", "two"}
	json.NewEncoder(w).Encode(c)
}
