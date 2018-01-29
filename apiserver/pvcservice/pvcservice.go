package pvcservice

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/gorilla/mux"
	"net/http"
)

// ShowPVCHandler ...
// returns a ShowPVCResponse
func ShowPVCHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	vars := mux.Vars(r)
	log.Debugf("pvcervice.ShowPVCHandler %v\n", vars)

	pvcname := vars["pvcname"]

	pvcroot := r.URL.Query().Get("pvcroot")
	if pvcroot != "" {
		log.Debug("pvcroot param was [" + pvcroot + "]")
	}
	switch r.Method {
	case "GET":
		log.Debug("pvcservice.ShowPVCHandler GET called")
	case "DELETE":
		log.Debug("pvcservice.ShowPVCHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ShowPVCResponse{}
	resp.Results, err = ShowPVC(pvcname, pvcroot)
	if err != nil {
		resp.Status.Code = "error"
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
