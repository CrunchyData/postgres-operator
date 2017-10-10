package policyservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	//crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	apiserver "github.com/crunchydata/kraken/apiserver"
	msgs "github.com/crunchydata/kraken/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// pgo create policy
// parameters secretfrom
func CreatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("policyservice.CreatePolicyHandler called")
	var request msgs.CreatePolicyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("policyservice.CreatePolicyHandler got request " + request.Name)

	err := CreatePolicy(apiserver.RestClient, request.Namespace, request.Name, request.URL, request.SQL)
	if err != nil {
		log.Error(err.Error())
		log.Infoln("error would be reported back to caller!!!!")
	}
}

// returns a ShowPolicyResponse
func ShowPolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("policyservice.ShowPolicyHandler called")
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	policyname := vars["name"]
	log.Infof(" name arg is %v\n", policyname)

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Infoln("namespace param was [" + namespace + "]")
	} else {
		log.Infoln("namespace param was null")
	}

	switch r.Method {
	case "GET":
		log.Infoln("policyservice.ShowPolicyHandler GET called")
	case "DELETE":
		log.Infoln("policyservice.ShowPolicyHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ShowPolicyResponse{}
	resp.PolicyList = ShowPolicy(apiserver.RestClient, namespace, policyname)

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
	c := new(msgs.ApplyResults)
	c.Results = []string{"one", "two"}
	json.NewEncoder(w).Encode(c)
}
