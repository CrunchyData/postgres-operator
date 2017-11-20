package cmd

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

// StatusCheck ...
func StatusCheck(resp *http.Response) {
	log.Debugf("http status code is %d\n", resp.StatusCode)
	if resp.StatusCode == 401 {
		log.Fatalf("Authentication Failed: %d\n", resp.StatusCode)
		os.Exit(2)
	} else if resp.StatusCode != 200 {
		log.Fatalf("Invalid Status Code: %d\n", resp.StatusCode)
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
	if err != nil {
		log.Debug(fullPath + " not found")
	} else {
		log.Debug(fullPath + " found")
		log.Debug("pgouser file found at " + fullPath + "contains " + string(dat))
		BasicAuthUsername, BasicAuthPassword = parseCredentials(string(dat))
		return
	}

	fullPath = etcpath
	dat, err = ioutil.ReadFile(fullPath)
	if err != nil {
		log.Debug(etcpath + " not found")
	} else {
		log.Debug(fullPath + " found")
		log.Debug("pgouser file found at " + fullPath + "contains " + string(dat))
		BasicAuthUsername, BasicAuthPassword = parseCredentials(string(dat))
		return
	}

	pgoUser := os.Getenv(pgouserenvvar)
	if pgoUser == "" {
		log.Error(pgouserenvvar + " env var not set")
		os.Exit(2)
	}

	fullPath = pgoUser
	log.Debug(pgouserenvvar + " env var is being used at " + fullPath)
	dat, err = ioutil.ReadFile(fullPath)
	if err != nil {
		log.Error(fullPath + " file not found")
		os.Exit(2)
	}

	log.Debug("pgouser file found at " + fullPath + "contains " + string(dat))
	BasicAuthUsername, BasicAuthPassword = parseCredentials(string(dat))

	/**
	caCert, err := ioutil.ReadFile("/tmp/server.crt")
	if err != nil {
		log.Error(err)
		log.Error("could not read ca certificate")
		os.Exit(2)
	}
	caCertPool = x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err = tls.LoadX509KeyPair("/tmp/client.crt", "/tmp/client.key")
	if err != nil {
		log.Fatal(err)
		log.Error("could not load client.crt and client.key")
		os.Exit(2)
	} */

}
