package cmd

/*
 Copyright 2017-2019 Crunchy Data Solutions, Inc.
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
	"fmt"
	log "github.com/Sirupsen/logrus"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const etcpath = "/etc/pgo/pgouser"
const pgouserenvvar = "PGOUSER"

// BasicAuthUsername and BasicAuthPassword are for BasicAuth, they are fetched from a file

var SessionCredentials msgs.BasicAuthCredentials

//var BasicAuthUsername, BasicAuthPassword string

var caCertPool *x509.CertPool
var cert tls.Certificate
var httpclient *http.Client
var caCertPath, clientCertPath, clientKeyPath string

// StatusCheck ...
func StatusCheck(resp *http.Response) {
	log.Debugf("HTTP status code is %d", resp.StatusCode)
	if resp.StatusCode == 401 {
		fmt.Printf("Error: Authentication Failed: %d\n", resp.StatusCode)
		os.Exit(2)
	} else if resp.StatusCode != 200 {
		fmt.Printf("Error: Invalid Status Code: %d\n", resp.StatusCode)
		os.Exit(2)
	}
}

func UserHomeDir() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

func parseCredentials(dat string) msgs.BasicAuthCredentials {

	fields := strings.Split(strings.TrimSpace(dat), ":")
	log.Debugf("%v", fields)
	log.Debugf("username=[%s] password=[%s]", fields[0], fields[1])
	//return fields[0], fields[1]
	creds := msgs.BasicAuthCredentials{
		Username:     fields[0],
		Password:     fields[1],
		APIServerURL: APIServerURL,
	}
	return creds
}

func GetCredentials() {
	log.Debug("GetCredentials called")

	dir := UserHomeDir()
	fullPath := dir + "/" + ".pgouser"
	log.Debugf("looking in %s for credentials", fullPath)
	dat, err := ioutil.ReadFile(fullPath)
	found := false
	if err != nil {
		log.Debugf("%s not found", fullPath)
	} else {
		log.Debugf("%s found", fullPath)
		log.Debugf("pgouser file found at %s contains %s", fullPath, string(dat))
		SessionCredentials = parseCredentials(string(dat))
		found = true

	}

	if !found {
		fullPath = etcpath
		dat, err = ioutil.ReadFile(fullPath)
		if err != nil {
			log.Debugf("%s not found", etcpath)
		} else {
			log.Debugf("%s found", fullPath)
			log.Debugf("pgouser file found at %s contains ", fullPath, string(dat))
			SessionCredentials = parseCredentials(string(dat))
			found = true
		}
	}

	if !found {
		pgoUser := os.Getenv(pgouserenvvar)
		if pgoUser == "" {
			fmt.Printf("Error: %s environment variable not set", pgouserenvvar)
			os.Exit(2)
		}

		fullPath = pgoUser
		log.Debugf("%s environment variable is being used at %s", pgouserenvvar, fullPath)
		dat, err = ioutil.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("Error: %s file not found", fullPath)
			os.Exit(2)
		}

		log.Debugf("pgouser file found at %s contains ", fullPath, string(dat))
		SessionCredentials = parseCredentials(string(dat))
	}

	if PGO_CA_CERT != "" {
		caCertPath = PGO_CA_CERT
	} else {
		caCertPath = os.Getenv("PGO_CA_CERT")
	}

	if caCertPath == "" {
		fmt.Println("Error: PGO_CA_CERT not specified")
		os.Exit(2)
	}
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		fmt.Printf("Error: %s", err)
		fmt.Println("could not read ca certificate")
		os.Exit(2)
	}
	caCertPool = x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	if PGO_CLIENT_CERT != "" {
		clientCertPath = PGO_CLIENT_CERT
	} else {
		clientCertPath = os.Getenv("PGO_CLIENT_CERT")
	}

	if clientCertPath == "" {
		fmt.Println("Error: PGO_CLIENT_CERT not specified")
		os.Exit(2)
	}

	_, err = ioutil.ReadFile(clientCertPath)
	if err != nil {
		log.Debugf("%s not found", clientCertPath)
		os.Exit(2)
	}

	if PGO_CLIENT_KEY != "" {
		clientKeyPath = PGO_CLIENT_KEY
	} else {
		clientKeyPath = os.Getenv("PGO_CLIENT_KEY")
	}

	if clientKeyPath == "" {
		fmt.Println("Error: PGO_CLIENT_KEY not specified")
		os.Exit(2)
	}

	_, err = ioutil.ReadFile(clientKeyPath)
	if err != nil {
		log.Debugf("%s not found", clientKeyPath)
		os.Exit(2)
	}
	cert, err = tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		fmt.Println("Error: could not load example.com.crt and example.com.key")
		os.Exit(2)
	}

	log.Debug("setting up httpclient with TLS")
	httpclient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            caCertPool,
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cert},
			},
		},
	}

}
