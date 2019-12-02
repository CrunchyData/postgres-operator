package main

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/routing"
	crunchylog "github.com/crunchydata/postgres-operator/logging"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Created as part of the apiserver.WriteTLSCert call
const serverCertPath = "/tmp/server.crt"
const serverKeyPath = "/tmp/server.key"

func main() {

	PORT := "8443"
	tmp := os.Getenv("PORT")
	if tmp != "" {
		PORT = tmp
	}

	//give time for pgo-event to start up
	time.Sleep(time.Duration(5) * time.Second)

	debugFlag := os.Getenv("CRUNCHY_DEBUG")
	//add logging configuration
	crunchylog.CrunchyLogger(crunchylog.SetParameters())
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

	tmp = os.Getenv("DISABLE_TLS")
	if tmp == "true" {
		log.Debug("DISABLE_TLS set to true")
	} else {
		tmp = "false"
		log.Debug("DISABLE_TLS set to false")
	}
	disableTLS, _ := strconv.ParseBool(tmp)

	skipAuthRoutes := strings.TrimSpace(os.Getenv("NOAUTH_ROUTES"))

	log.Infoln("postgres-operator apiserver starts")

	apiserver.Initialize()

	r := mux.NewRouter()
	routing.RegisterAllRoutes(r)

	certsVerify := tls.VerifyClientCertIfGiven
	skipAuth := []string{
		"/healthz", // Required for kube probes
	}
	if !disableTLS {
		if len(skipAuthRoutes) > 0 {
			skipAuth = append(skipAuth, strings.Split(skipAuthRoutes, ",")...)
		}
		optCertEnforcer, err := apiserver.NewCertEnforcer(skipAuth)
		if err != nil {
			log.Fatalf("NOAUTH_ROUTES configured incorrectly: %s", err)
		}
		r.Use(optCertEnforcer.Enforce)
	}

	err := apiserver.WriteTLSCert(serverCertPath, serverKeyPath)
	if err != nil {
		log.Fatal(err)
	}

	var caCert []byte

	caCert, err = ioutil.ReadFile(serverCertPath)
	if err != nil {
		log.Fatalf("could not read %s - %v", serverCertPath, err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cfg := &tls.Config{
		//specify pgo-apiserver in the CN....then, add ServerName: "pgo-apiserver",
		ServerName:         "pgo-apiserver",
		ClientAuth:         certsVerify,
		InsecureSkipVerify: tlsNoVerify,
		ClientCAs:          caCertPool,
		MinVersion:         tls.VersionTLS11,
	}

	log.Info("listening on port " + PORT)

	_, err = ioutil.ReadFile(serverKeyPath)
	if err != nil {
		log.Fatalf("could not read %s - %v", serverKeyPath, err)
	}

	var srv *http.Server
	if !disableTLS {
		srv = &http.Server{
			Addr:      ":" + PORT,
			Handler:   r,
			TLSConfig: cfg,
		}
		log.Fatal(srv.ListenAndServeTLS(serverCertPath, serverKeyPath))
	} else {
		srv = &http.Server{
			Addr:    ":" + PORT,
			Handler: r,
		}
		log.Fatal(srv.ListenAndServe())
	}

}
