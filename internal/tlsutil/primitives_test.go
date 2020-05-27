package tlsutil

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"crypto/rsa"
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

	if !keysEq(oldKey, newKey) {
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
	ExtendTrust(caTrust, bytes.NewReader(pemCert))

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
	res, err := client.Get(srv.URL)
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

func keysEq(a, b *rsa.PrivateKey) bool {
	if a.E != b.E {
		// PublicKey exponent different
		return false
	}
	if a.N.Cmp(b.N) != 0 {
		// PublicKey modulus different
		return false
	}
	if a.D.Cmp(b.D) != 0 {
		// PrivateKey exponent different
		return false
	}
	if len(a.Primes) != len(b.Primes) {
		// Prime factor difference (Tier 1)
		return false
	}
	for i, aPrime := range a.Primes {
		if aPrime.Cmp(b.Primes[i]) != 0 {
			// Prime factor difference (Tier 2)
			return false
		}
	}

	return true
}
