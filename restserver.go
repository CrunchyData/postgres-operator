package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/backupservice"
	"github.com/crunchydata/postgres-operator/cloneservice"
	"github.com/crunchydata/postgres-operator/clusterservice"
	"github.com/crunchydata/postgres-operator/policyservice"
	"github.com/crunchydata/postgres-operator/upgradeservice"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {

	log.Infoln("restserver starts")
	r := mux.NewRouter()
	r.HandleFunc("/clones", cloneservice.CreateCloneHandler)
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/policies/{name}", policyservice.ShowPolicyHandler).Methods("GET", "DELETE")
	r.HandleFunc("/policies/apply/{name}", policyservice.ApplyPolicyHandler)
	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler)
	r.HandleFunc("/upgrades/{name}", upgradeservice.ShowUpgradeHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler)
	r.HandleFunc("/clusters/{name}", clusterservice.ShowClusterHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters/test/{name}", clusterservice.TestClusterHandler)
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET", "DELETE")
	log.Fatal(http.ListenAndServe(":8080", r))
}
