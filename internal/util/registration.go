/*
 Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package util

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"

	"github.com/crunchydata/postgres-operator/internal/config"
)

// Registration is required only for OLM installations of the operator.
type Registration struct {
	// Registration token status.
	Authenticated  bool `json:"authenticated"`
	TokenFileFound bool `json:"tokenFileFound"`

	// Token claims.
	Aud string `json:"aud"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
	Iss string `json:"iss"`
	Nbf int64  `json:"nbf"`
	Sub string `json:"sub"`
}

func parseRSAPublicKey(rawKey string) (*rsa.PublicKey, error) {
	var rsaPublicKey *rsa.PublicKey
	rsaPublicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(rawKey))
	return rsaPublicKey, err
}

func getToken(tokenPath string) (string, error) {
	if _, err := os.Stat(tokenPath); err != nil {
		return "", err
	}

	bs, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}

	token := string(bs)
	if token == "" {
		return "", errors.New("token cannot be empty")
	}

	return token, nil
}

// GetRegistration returns an empty struct if registration is not required.
func GetRegistration(rawKey string, tokenPath string, log logr.Logger) Registration {
	registration := Registration{}

	if !config.RegistrationRequired() {
		return registration
	}

	// If the key is invalid, registration cannot be enforced.
	key, err := parseRSAPublicKey(rawKey)
	if err != nil {
		log.Error(err, "Error parsing RSA key")
		return registration
	}

	// If there is no token, an operator installation cannot be registered.
	token, err := getToken(tokenPath)
	if err != nil {
		log.Error(err, "Error getting token: "+tokenPath)
		return registration
	}

	// Acknowledge that a token was provided, even if it isn't valid.
	registration.TokenFileFound = true

	// Decode the token signature.
	parts := strings.Split(token, ".")
	sig, _ := jwt.NewParser().DecodeSegment(parts[2])

	// Claims consist of header and payload.
	claims := strings.Join(parts[0:2], ".")

	// Verify the token.
	method := jwt.GetSigningMethod("RS256")
	err = method.Verify(claims, sig, key)
	if err == nil {
		log.Info("token authentication succeeded")
		registration.Authenticated = true
	} else {
		log.Error(err, "token authentication failed")
	}

	// Populate Registration with token payload.
	payloadStr, _ := jwt.NewParser().DecodeSegment(parts[1])
	err = json.Unmarshal(payloadStr, &registration)
	if err != nil {
		log.Error(err, "token error")
	}

	return registration
}
