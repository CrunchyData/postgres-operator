package main

import (
	"crypto/tls"
	"crypto/x509"
	log "github.com/Sirupsen/logrus"
	"github.com/crunchydata/postgres-operator/apiserver/backupservice"
	"github.com/crunchydata/postgres-operator/apiserver/cloneservice"
	"github.com/crunchydata/postgres-operator/apiserver/clusterservice"
	"github.com/crunchydata/postgres-operator/apiserver/labelservice"
	"github.com/crunchydata/postgres-operator/apiserver/loadservice"
	"github.com/crunchydata/postgres-operator/apiserver/policyservice"
	"github.com/crunchydata/postgres-operator/apiserver/pvcservice"
	"github.com/crunchydata/postgres-operator/apiserver/upgradeservice"
	"github.com/crunchydata/postgres-operator/apiserver/userservice"
	"github.com/crunchydata/postgres-operator/apiserver/versionservice"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const serverCert = "/config/server.crt"
const serverKey = "/config/server.key"

func main() {

	PORT := "8443"
	tmp := os.Getenv("PORT")
	if tmp != "" {
		PORT = tmp
	}

	debugFlag := os.Getenv("DEBUG")
	if debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	tmp = os.Getenv("TLS_NO_VERIFY")
	if tmp == "true" {
		log.Debug("TLS_NO_VERIFY set to true")
	} else {
		tmp = "false"
		log.Debug("TLS_NO_VERIFY set to false")
	}
	tlsNoVerify, _ := strconv.ParseBool(tmp)

	log.Infoln("postgres-operator apiserver starts")
	r := mux.NewRouter()
	r.HandleFunc("/version", versionservice.VersionHandler)
	r.HandleFunc("/clones", cloneservice.CreateCloneHandler)
	r.HandleFunc("/policies", policyservice.CreatePolicyHandler)
	r.HandleFunc("/policies/{name}", policyservice.ShowPolicyHandler).Methods("GET", "DELETE")
	r.HandleFunc("/pvc/{pvcname}", pvcservice.ShowPVCHandler).Methods("GET")
	r.HandleFunc("/policies/apply", policyservice.ApplyPolicyHandler).Methods("POST")
	r.HandleFunc("/label", labelservice.LabelHandler).Methods("POST")
	r.HandleFunc("/load", loadservice.LoadHandler).Methods("POST")
	r.HandleFunc("/user", userservice.UserHandler).Methods("POST")
	r.HandleFunc("/upgrades", upgradeservice.CreateUpgradeHandler).Methods("POST")
	r.HandleFunc("/upgrades/{name}", upgradeservice.ShowUpgradeHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters", clusterservice.CreateClusterHandler).Methods("POST")
	r.HandleFunc("/clusters/{name}", clusterservice.ShowClusterHandler).Methods("GET", "DELETE")
	r.HandleFunc("/clusters/test/{name}", clusterservice.TestClusterHandler)
	r.HandleFunc("/clusters/scale/{name}", clusterservice.ScaleClusterHandler)
	r.HandleFunc("/backups/{name}", backupservice.ShowBackupHandler).Methods("GET", "DELETE")
	r.HandleFunc("/backups", backupservice.CreateBackupHandler).Methods("POST")

	caCert, err := ioutil.ReadFile(serverCert)
	if err != nil {
		log.Fatal(err)
		log.Error("could not read " + serverCert)
		os.Exit(2)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cfg := &tls.Config{
		//ClientAuth: tls.RequireAndVerifyClientCert,
		//specify pgo-apiserver in the CN....then, add ServerName: "pgo-apiserver",
		ServerName:         "pgo-apiserver",
		InsecureSkipVerify: tlsNoVerify,
		ClientCAs:          caCertPool,
	}

	log.Info("listening on port " + PORT)

	srv := &http.Server{
		Addr:      ":" + PORT,
		Handler:   r,
		TLSConfig: cfg,
	}

	_, err = ioutil.ReadFile(serverKey)
	if err != nil {
		log.Fatal(err)
		log.Error("could not read " + serverKey)
		os.Exit(2)
	}

	log.Fatal(srv.ListenAndServeTLS(serverCert, serverKey))
	//log.Fatal(http.ListenAndServe(":8080", r))
}
