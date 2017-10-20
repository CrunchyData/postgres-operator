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
	log.Infoln("pvcervice.ShowPVCHandler called")
	vars := mux.Vars(r)
	log.Infof(" vars are %v\n", vars)

	pvcname := vars["pvcname"]
	log.Infof(" pvcname arg is %v\n", pvcname)

	namespace := r.URL.Query().Get("namespace")
	if namespace != "" {
		log.Infoln("namespace param was [" + namespace + "]")
	} else {
		log.Infoln("namespace param was null")
	}
	pvcroot := r.URL.Query().Get("pvcroot")
	if pvcroot != "" {
		log.Infoln("pvcroot param was [" + pvcroot + "]")
	} else {
		log.Infoln("pvcroot param was null")
	}
	switch r.Method {
	case "GET":
		log.Infoln("pvcservice.ShowPVCHandler GET called")
	case "DELETE":
		log.Infoln("pvcservice.ShowPVCHandler DELETE called")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	resp := msgs.ShowPVCResponse{}
	resp.Results, err = ShowPVC(namespace, pvcname, pvcroot)
	if err != nil {
		resp.Status.Code = "error"
		resp.Status.Msg = err.Error()
	}

	json.NewEncoder(w).Encode(resp)
}
