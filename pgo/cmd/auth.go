package cmd

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"fmt"
	"crypto/tls"
	"crypto/x509"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const etcpath = "/etc/pgo/pgouser"
const pgouserenvvar = "PGOUSER"

// BasicAuthUsername and BasicAuthPassword are for BasicAuth, they are fetched from a file
var BasicAuthUsername, BasicAuthPassword string

var caCertPool *x509.CertPool
var cert tls.Certificate
var httpclient *http.Client
var caCertPath, clientCertPath, clientKeyPath string

// StatusCheck ...
func StatusCheck(resp *http.Response) {
	log.Debugf("http status code is %d\n", resp.StatusCode)
	if resp.StatusCode == 401 {
		fmt.Println("Error: Authentication Failed: %d\n", resp.StatusCode)
		os.Exit(2)
	} else if resp.StatusCode != 200 {
		fmt.Println("Error: Invalid Status Code: %d\n", resp.StatusCode)
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

func parseCredentials(dat string) (string, string) {

	fields := strings.Split(strings.TrimSpace(dat), ":")
	log.Debugf("%v\n", fields)
	log.Debugf("username=[%s] password=[%s]\n", fields[0], fields[1])
	return fields[0], fields[1]
}

func GetCredentials() {
	log.Debug("GetCredentials called")

	dir := UserHomeDir()
	fullPath := dir + "/" + ".pgouser"
	log.Debug("looking in " + fullPath + " for credentials")
	dat, err := ioutil.ReadFile(fullPath)
	found := false
	if err != nil {
		log.Debug(fullPath + " not found")
	} else {
		log.Debug(fullPath + " found")
		log.Debug("pgouser file found at " + fullPath + "contains " + string(dat))
		BasicAuthUsername, BasicAuthPassword = parseCredentials(string(dat))
		found = true

	}

	if !found {
		fullPath = etcpath
		dat, err = ioutil.ReadFile(fullPath)
		if err != nil {
			log.Debug(etcpath + " not found")
		} else {
			log.Debug(fullPath + " found")
			log.Debug("pgouser file found at " + fullPath + "contains " + string(dat))
			BasicAuthUsername, BasicAuthPassword = parseCredentials(string(dat))
			found = true
		}
	}

	if !found {
		pgoUser := os.Getenv(pgouserenvvar)
		if pgoUser == "" {
			fmt.Println("Error: " + pgouserenvvar + " env var not set")
			os.Exit(2)
		}

		fullPath = pgoUser
		log.Debug(pgouserenvvar + " environment variable is being used at " + fullPath)
		dat, err = ioutil.ReadFile(fullPath)
		if err != nil {
			fmt.Println("Error: " + fullPath + " file not found")
			os.Exit(2)
		}

		log.Debug("pgouser file found at " + fullPath + "contains " + string(dat))
		BasicAuthUsername, BasicAuthPassword = parseCredentials(string(dat))
	}

	caCertPath = os.Getenv("PGO_CA_CERT")

	if caCertPath == "" {
		fmt.Println("Error: PGO_CA_CERT not specified")
		os.Exit(2)
	}
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		fmt.Println("Error: ", err)
		fmt.Println("could not read ca certificate")
		os.Exit(2)
	}
	caCertPool = x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCertPath = os.Getenv("PGO_CLIENT_CERT")

	if clientCertPath == "" {
		fmt.Println("Error: PGO_CLIENT_CERT not specified")
		os.Exit(2)
	}

	_, err = ioutil.ReadFile(clientCertPath)
	if err != nil {
		log.Debug(clientCertPath + " not found")
		os.Exit(2)
	}

	clientKeyPath = os.Getenv("PGO_CLIENT_KEY")

	if clientKeyPath == "" {
		fmt.Println("Error: PGO_CLIENT_KEY not specified")
		os.Exit(2)
	}

	_, err = ioutil.ReadFile(clientKeyPath)
	if err != nil {
		log.Debug(clientKeyPath + " not found")
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
