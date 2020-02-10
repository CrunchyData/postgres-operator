package main

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/crunchydata/postgres-operator/apiserver"
	"github.com/crunchydata/postgres-operator/apiserver/routing"
	crunchylog "github.com/crunchydata/postgres-operator/logging"
	"github.com/crunchydata/postgres-operator/tlsutil"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Created as part of the apiserver.WriteTLSCert call
const serverCertPath = "/tmp/server.crt"
const serverKeyPath = "/tmp/server.key"

func main() {
	// Environment-overridden variables
	srvPort := "8443"
	tlsDisabled := false
	tlsNoVerify := false
	tlsTrustedCAs := x509.NewCertPool()

	// NOAUTH_ROUTES identifies a comma-separated list of URL routes
	// which will have authentication disabled, both system-to-system
	// via TLS and HTTP Basic used to power RBAC
	skipAuthRoutes := strings.TrimSpace(os.Getenv("NOAUTH_ROUTES"))

	// PORT overrides the server listening port
	if p, ok := os.LookupEnv("PORT"); ok && p != "" {
		srvPort = p
	}

	// CRUNCHY_DEBUG sets the logging level to Debug (more verbose)
	if debug, _ := strconv.ParseBool(os.Getenv("CRUNCHY_DEBUG")); debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag set to true")
	} else {
		log.Info("debug flag set to false")
	}

	// TLS_NO_VERIFY disables verification of SSL client certificates
	if noVerify, _ := strconv.ParseBool(os.Getenv("TLS_NO_VERIFY")); noVerify {
		tlsNoVerify = noVerify
	}
	log.Debugf("TLS_NO_VERIFY set as %t", tlsNoVerify)

	// DISABLE_TLS configures the server to listen over HTTP
	if noTLS, _ := strconv.ParseBool(os.Getenv("DISABLE_TLS")); noTLS {
		tlsDisabled = noTLS
	}
	log.Debugf("DISABLE_TLS set as %t", tlsDisabled)

	if !tlsDisabled {
		// ADD_OS_TRUSTSTORE causes the API server to trust clients with a
		// cert issued by CAs the underlying OS already trusts
		if osTrust, _ := strconv.ParseBool(os.Getenv("ADD_OS_TRUSTSTORE")); osTrust {
			if osCAs, err := x509.SystemCertPool(); err != nil {
				log.Errorf("unable to read OS truststore - [%v], ignoring option", err)
			} else {
				tlsTrustedCAs = osCAs
			}
		}

		// TLS_CA_TRUST identifies a PEM-encoded file containing certificate
		// authorities trusted to identify client SSL connections
		if tp, ok := os.LookupEnv("TLS_CA_TRUST"); ok && tp != "" {
			if trustFile, err := os.Open(tp); err != nil {
				log.Errorf("unable to load TLS trust from %s - [%v], ignoring option", tp, err)
			} else {
				err = tlsutil.ExtendTrust(tlsTrustedCAs, trustFile)
				if err != nil {
					log.Errorf("error reading %s - %v, ignoring option", tp, err)
				}
				trustFile.Close()
			}
		}
	}

	// init crunchy-formatted logger
	crunchylog.CrunchyLogger(crunchylog.SetParameters())

	// give time for pgo-event to start up
	time.Sleep(time.Duration(5) * time.Second)

	log.Infoln("postgres-operator apiserver starts")
	apiserver.Initialize()

	r := mux.NewRouter()
	routing.RegisterAllRoutes(r)

	var srv *http.Server
	if !tlsDisabled {
		// Set up deferred enforcement of certs, given Verify...IfGiven setting
		skipAuth := []string{
			"/healthz", // Required for kube probes
		}
		if len(skipAuthRoutes) > 0 {
			skipAuth = append(skipAuth, strings.Split(skipAuthRoutes, ",")...)
		}
		certEnforcer, err := apiserver.NewCertEnforcer(skipAuth)
		if err != nil {
			// Since disabling authentication would break functionality
			// dependent on the user identity, only certain routes may be
			// configured in NOAUTH_ROUTES.
			log.Fatalf("NOAUTH_ROUTES configured incorrectly: %s", err)
		}
		r.Use(certEnforcer.Enforce)

		// Cert files are used for http.ListenAndServeTLS
		err = apiserver.WriteTLSCert(serverCertPath, serverKeyPath)
		if err != nil {
			log.Fatalf("unable to open server cert at %s - %v", serverKeyPath, err)
		}

		// Add server cert to trust root, necessarily includes server
		// certificate issuer chain (intermediate and root CAs)
		if svrCertFile, err := os.Open(serverCertPath); err != nil {
			log.Fatalf("unable to open %s for reading - %v", serverCertPath, err)
		} else {
			if err = tlsutil.ExtendTrust(tlsTrustedCAs, svrCertFile); err != nil {
				log.Fatalf("error reading server cert at %s - %v", serverCertPath, err)
			}
			svrCertFile.Close()
		}

		cfg := &tls.Config{
			//specify pgo-apiserver in the CN....then, add ServerName: "pgo-apiserver",
			ServerName:         "pgo-apiserver",
			ClientAuth:         tls.VerifyClientCertIfGiven,
			InsecureSkipVerify: tlsNoVerify,
			ClientCAs:          tlsTrustedCAs,
			MinVersion:         tls.VersionTLS11,
		}

		srv = &http.Server{
			Addr:      ":" + srvPort,
			Handler:   r,
			TLSConfig: cfg,
		}
		log.Info("listening on port " + srvPort)
		log.Fatal(srv.ListenAndServeTLS(serverCertPath, serverKeyPath))
	} else {
		srv = &http.Server{
			Addr:    ":" + srvPort,
			Handler: r,
		}
		log.Info("listening on port " + srvPort)
		log.Fatal(srv.ListenAndServe())
	}
}
