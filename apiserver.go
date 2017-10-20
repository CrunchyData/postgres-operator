package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver/backupservice"
	"github.com/crunchydata/postgres-operator/apiserver/cloneservice"
	"github.com/crunchydata/postgres-operator/apiserver/clusterservice"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	"github.com/crunchydata/postgres-operator/apiserver/pvcservice"
	"github.com/crunchydata/postgres-operator/apiserver/upgradeservice"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {

	log.Infoln("postgres-operator apiserver starts")
	r := mux.NewRouter()
	r.HandleFunc("/clones", cloneservice.CreateCloneHandler)
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	//r.HandleFunc("/policies/{name}", policyservice.ShowPolicyHandler).
	//Queries("selector", "{selector}").Methods("GET", "DELETE")
	r.HandleFunc("/policies/{name}", policyservice.ShowPolicyHandler).Methods("GET", "DELETE")
	r.HandleFunc("/pvc/{pvcname}", pvcservice.ShowPVCHandler).Methods("GET")
	r.HandleFunc("/policies/apply", policyservice.ApplyPolicyHandler).Methods("POST")
	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler)
	r.HandleFunc("/upgrades/{name}", upgradeservice.ShowUpgradeHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler)
	r.HandleFunc("/clusters/{name}", clusterservice.ShowClusterHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters/test/{name}", clusterservice.TestClusterHandler)
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET", "DELETE")
	log.Fatal(http.ListenAndServe(":8080", r))
}
