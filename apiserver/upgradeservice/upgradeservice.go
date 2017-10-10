package upgradeservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type UpgradeDetail struct {
	Name string
	//deployments
	//replicasets
	//pods
	//services
	//secrets
}
type ShowUpgradeResponse struct {
	Items []UpgradeDetail
}

type CreateUpgradeRequest struct {
	Name string
}

// pgo upgrade mycluster
// parameters --upgrade-type
// parameters --ccp-image-tag
func CreateUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("upgradeservice.CreateUpgradeHandler called")
	var request CreateUpgradeRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("upgradeservice.CreateUpgradeHandler got request " + request.Name)
}

// pgo show upgrade
// pgo delete myupgrade
// parameters showsecrets
// parameters selector
// parameters namespace
// parameters postgresversion
// returns a ShowUpgradeResponse
func ShowUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("upgradeservice.ShowUpgradeHandler called")
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	switch r.Method {
	case "GET":
		log.Infoln("upgradeservice.ShowUpgradeHandler GET called")
	case "DELETE":
		log.Infoln("upgradeservice.ShowUpgradeHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := new(ShowUpgradeResponse)
	resp.Items = []UpgradeDetail{}
	c := UpgradeDetail{}
	c.Name = "someupgrade"
	resp.Items = append(resp.Items, c)

	json.NewEncoder(w).Encode(resp)
}
