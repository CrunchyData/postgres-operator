package cmd

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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/tlsutil"

	log "github.com/sirupsen/logrus"
)

const (
	pgoUserFileEnvVar     = "PGOUSER"
	pgoUserNameEnvVar     = "PGOUSERNAME"
	pgoUserPasswordEnvVar = "PGOUSERPASS"
)

// SessionCredentials stores the PGO user, PGO password and the PGO APIServer URL
var SessionCredentials msgs.BasicAuthCredentials

// Globally shared Operator API HTTP client
var httpclient *http.Client

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

// userHomeDir updates the env variable with the appropriate home directory
// depending on the host operating system the PGO client is running on.
func userHomeDir() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

func parseCredentials(dat string) msgs.BasicAuthCredentials {
	// splits by new line to ensure that user on has one line in pgouser file/creds
	// this split does not take into account newline conventions of different systems
	// ex. windows new lines ("\r\n")
	lines := strings.Split(strings.TrimSpace(dat), "\n")
	if len(lines) != 1 {
		log.Debugf("expected one and only one line in pgouser file - found %d", len(lines))
		fmt.Println("unable to parse credentials in pgouser file")
		os.Exit(2) // TODO: graceful exit
	}

	// the delimiting char ":" is a valid password char so SplitN will handle if
	// ":" is used by always splitting into two substrings including the username
	// and everything after the first ":"
	fields := strings.SplitN(lines[0], ":", 2)
	if len(fields) != 2 {
		log.Debug("invalid credential format: expecting \"<username>:<password>\"")
		fmt.Println("unable to parse credentials in pgouser file")
		os.Exit(2) // TODO: graceful exit
	}
	log.Debugf("%v", fields)
	log.Debugf("username=[%s] password=[%s]", fields[0], fields[1])

	creds := msgs.BasicAuthCredentials{
		Username:     fields[0],
		Password:     fields[1],
		APIServerURL: APIServerURL,
	}
	return creds
}

// getCredentialsFromFile reads the pgouser and password from the .pgouser file,
// checking in the various locations that file can be expected, and then returns
// the credentials
func getCredentialsFromFile() msgs.BasicAuthCredentials {
	found := false
	dir := userHomeDir()
	fullPath := dir + "/" + ".pgouser"
	var creds msgs.BasicAuthCredentials

	//look in env var for pgouser file
	pgoUser := os.Getenv(pgoUserFileEnvVar)
	if pgoUser != "" {
		fullPath = pgoUser
		log.Debugf("%s environment variable is being used at %s", pgoUserFileEnvVar, fullPath)
		dat, err := ioutil.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("Error: %s file not found", fullPath)
			os.Exit(2)
		}

		log.Debugf("pgouser file found at %s contains %s", fullPath, string(dat))
		creds = parseCredentials(string(dat))
		found = true
	}

	//look in home directory for .pgouser file
	if !found {
		log.Debugf("looking in %s for credentials", fullPath)
		dat, err := ioutil.ReadFile(fullPath)
		if err != nil {
			log.Debugf("%s not found", fullPath)
		} else {
			log.Debugf("%s found", fullPath)
			log.Debugf("pgouser file found at %s contains %s", fullPath, string(dat))
			creds = parseCredentials(string(dat))
			found = true

		}
	}

	//look in etc for pgouser file
	if !found {
		fullPath = "/etc/pgo/pgouser"
		dat, err := ioutil.ReadFile(fullPath)
		if err != nil {
			log.Debugf("%s not found", fullPath)
		} else {
			log.Debugf("%s found", fullPath)
			log.Debugf("pgouser file found at %s contains %s", fullPath, string(dat))
			creds = parseCredentials(string(dat))
			found = true
		}
	}

	if !found {
		fmt.Println("could not find pgouser file")
		os.Exit(2)
	}

	return creds
}

// getCredentialsFromEnvironment reads the pgouser and password from relevant environment
// variables and then returns a created BasicAuthCredentials object with both values,
// as well as the APIServer URL.
func getCredentialsFromEnvironment() msgs.BasicAuthCredentials {
	pgoUser := os.Getenv(pgoUserNameEnvVar)
	pgoPass := os.Getenv(pgoUserPasswordEnvVar)

	if len(pgoUser) > 0 && len(pgoPass) < 1 {
		fmt.Println("Error: PGOUSERPASS needs to be specified if PGOUSERNAME is provided")
		os.Exit(2)
	}
	if len(pgoPass) > 0 && len(pgoUser) < 1 {
		fmt.Println("Error: PGOUSERNAME needs to be specified if PGOUSERPASS is provided")
		os.Exit(2)
	}

	creds := msgs.BasicAuthCredentials{
		Username:     os.Getenv(pgoUserNameEnvVar),
		Password:     os.Getenv(pgoUserPasswordEnvVar),
		APIServerURL: APIServerURL,
	}
	return creds
}

// SetSessionUserCredentials gathers the pgouser and password information
// and stores them for use by the PGO client
func SetSessionUserCredentials() {
	log.Debug("GetSessionCredentials called")

	SessionCredentials = getCredentialsFromEnvironment()

	if !SessionCredentials.HasUsernameAndPassword() {
		SessionCredentials = getCredentialsFromFile()
	}
}

// GetTLSTransport returns an http.Transport configured with environmental
// TLS client settings
func GetTLSTransport() (*http.Transport, error) {
	log.Debug("GetTLSTransport called")

	// By default, load the OS CA cert truststore unless explictly disabled
	// Reasonable default given the client controls to whom it is connecting
	var caCertPool *x509.CertPool
	if noTrust, _ := strconv.ParseBool(os.Getenv("EXCLUDE_OS_TRUST")); noTrust || EXCLUDE_OS_TRUST {
		caCertPool = x509.NewCertPool()
	} else {
		if pool, err := x509.SystemCertPool(); err != nil {
			return nil, fmt.Errorf("while loading System CA pool - %s", err)
		} else {
			caCertPool = pool
		}
	}

	// Priority: Flag -> ENV
	caCertPath := PGO_CA_CERT
	if caCertPath == "" {
		caCertPath = os.Getenv("PGO_CA_CERT")
		if caCertPath == "" {
			return nil, fmt.Errorf("PGO_CA_CERT not specified")
		}
	}

	// Open trust file and extend trust pool
	if trustFile, err := os.Open(caCertPath); err != nil {
		newErr := fmt.Errorf("unable to load TLS trust from %s - [%v]", caCertPath, err)
		return nil, newErr
	} else {
		err = tlsutil.ExtendTrust(caCertPool, trustFile)
		if err != nil {
			newErr := fmt.Errorf("error reading %s - %v", caCertPath, err)
			return nil, newErr
		}
		trustFile.Close()
	}

	// Priority: Flag -> ENV
	clientCertPath := PGO_CLIENT_CERT
	if clientCertPath == "" {
		clientCertPath = os.Getenv("PGO_CLIENT_CERT")
		if clientCertPath == "" {
			return nil, fmt.Errorf("PGO_CLIENT_CERT not specified")
		}
	}

	// Priority: Flag -> ENV
	clientKeyPath := PGO_CLIENT_KEY
	if clientKeyPath == "" {
		clientKeyPath = os.Getenv("PGO_CLIENT_KEY")
		if clientKeyPath == "" {
			return nil, fmt.Errorf("PGO_CLIENT_KEY not specified")
		}
	}

	certPair, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("client certificate/key loading: %s", err)
	}

	// create a Transport object for use by the HTTP client
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{certPair},
			MinVersion:         tls.VersionTLS11,
		},
	}, nil
}

// NewAPIClient returns an http client configured with a tls.Config
// based on environmental settings and a default timeout
func NewAPIClient() *http.Client {
	defaultTimeout := 60 * time.Second
	return &http.Client{
		Timeout: defaultTimeout,
	}
}

// NewAPIClientTLS returns an http client configured with a tls.Config
// based on environmental settings and a default timeout
// It returns an error if required environmental settings are missing
func NewAPIClientTLS() (*http.Client, error) {
	client := NewAPIClient()
	if tp, err := GetTLSTransport(); err != nil {
		return nil, err
	} else {
		client.Transport = tp
	}

	return client, nil
}
