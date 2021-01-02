package tlsutil

/*
Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeyPEMSymmetry(t *testing.T) {
	oldKey, err := NewPrivateKey()
	if err != nil {
		t.Fatalf("unable to generate new key - %s", err)
	}

	pemKey := EncodePrivateKeyPEM(oldKey)
	newKey, err := ParsePEMEncodedPrivateKey(pemKey)
	if err != nil {
		t.Fatalf("unable to parse pem key - %s", err)
	}

	t.Log(base64.StdEncoding.EncodeToString(pemKey))

	if !(oldKey.Equal(newKey) && oldKey.PublicKey.Equal(newKey.Public())) {
		t.Fatal("Decoded key did not match its input source")
	}
}

func TestCertPEMSymmetry(t *testing.T) {
	privKey, err := NewPrivateKey()
	if err != nil {
		t.Fatalf("unable to generate new key - %s", err)
	}

	oldCert, err := NewSelfSignedCACertificate(privKey)
	if err != nil {
		t.Fatalf("unable to generate cert - %s", err)
	}

	pemCert := EncodeCertificatePEM(oldCert)

	newCert, err := ParsePEMEncodedCert(pemCert)
	if err != nil {
		t.Fatalf("error decoding cert PEM - %s", err)
	}

	if !oldCert.Equal(newCert) {
		t.Fatal("decoded cert did not match its input source")
	}
}

func TestExtendedTrust(t *testing.T) {
	expected := "You do that very well. It's as if i was looking in a mirror."

	// Create x509 certificate pair (key, cert)
	key, err := NewPrivateKey()
	if err != nil {
		t.Fatalf("error creating private key - %s\n", err)
	}
	pemKey := EncodePrivateKeyPEM(key)

	cert, err := NewSelfSignedCACertificate(key)
	if err != nil {
		t.Fatalf("error creating cert - %s\n", err)
	}
	pemCert := EncodeCertificatePEM(cert)

	// Set up and start server
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, expected)
	}))
	defer srv.Close()

	caTrust := x509.NewCertPool()
	_ = ExtendTrust(caTrust, bytes.NewReader(pemCert))

	// #nosec G402
	srv.TLS = &tls.Config{
		ServerName:         "Stom",
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: true, // because self-signed, naturally
		ClientCAs:          caTrust,
		MinVersion:         tls.VersionTLS11,
	}
	srv.StartTLS()

	// Set up client
	clientCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		t.Fatalf("unable to prepare client cert - %s\n", err)
	}

	client := srv.Client()
	// #nosec G402
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{
				clientCert,
			},
			RootCAs:            caTrust,
			InsecureSkipVerify: true, // because self-signed, naturally
		},
	}

	// Confirm server response
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("error getting request - %s\n", err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("error getting response - %s\n", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatalf("error reading response -%s\n", err)
	}

	if recv := string(bytes.TrimSpace(body)); recv != expected {
		t.Fatalf("expected [%s], got [%s] instead\n", expected, recv)
	}
}
