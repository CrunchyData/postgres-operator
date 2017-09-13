package cloneservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	//"github.com/crunchydata/postgres-operator/tpr"
	//"github.com/gorilla/mux"
	"net/http"
)

type CloneResults struct {
	Results []string
}

type CreateCloneRequest struct {
	Name string
}

// pgo create clone
// parameters secretfrom
func CreateCloneHandler(w http.ResponseWriter, r *http.Request) {
	log.Infoln("cloneservice.CreateCloneHandler called")
	var request CreateCloneRequest
	_ = json.NewDecoder(r.Body).Decode(&request)

	log.Infoln("cloneservice.CreateCloneHandler got request " + request.Name)
}
