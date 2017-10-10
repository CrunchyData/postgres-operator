package clusterservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

type ScaleResponse struct {
	Results string
}

type ScaleRequest struct {
	Name string
}

// pgo scale mycluster --replica-count=1
// parameters showsecrets
// returns a ScaleResponse
func ScaleClusterHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("clusterservice.ScaleClusterHandler called")
	//log.Infoln("showsecrets=" + showsecrets)
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	switch r.Method {
	case "GET":
		log.Infoln("clusterservice.ScaleClusterHandler GET called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := new(ScaleResponse)
	resp.Results = "ok it worked"

	json.NewEncoder(w).Encode(resp)
}
