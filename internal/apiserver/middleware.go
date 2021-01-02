package apiserver

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
	"fmt"
	"net/http"
	"strings"
)

// certEnforcer is a contextual middleware for deferred enforcement of
// client certificates. It assumes any certificates presented where validated
// as part of establishing the TLS connection.
type certEnforcer struct {
	skip map[string]struct{}
}

// NewCertEnforcer ensures a certEnforcer is created with skipped routes
// and validates that the configured routes are allowed
func NewCertEnforcer(reqRoutes []string) (*certEnforcer, error) {
	allowed := map[string]struct{}{
		// List of allowed routes is part of the published documentation
		"/health":  {},
		"/healthz": {},
	}

	ce := &certEnforcer{
		skip: map[string]struct{}{},
	}

	for _, route := range reqRoutes {
		r := strings.TrimSpace(route)
		if _, ok := allowed[r]; !ok {
			return nil, fmt.Errorf("Disabling auth unsupported for route [%s]", r)
		}
		ce.skip[r] = struct{}{}
	}
	return ce, nil
}

// Enforce is an HTTP middleware for selectively enforcing deferred client
// certificate checks based on the certEnforcer's skip list
func (ce *certEnforcer) Enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if _, ok := ce.skip[path]; ok {
			next.ServeHTTP(w, r)
		} else {
			clientCerts := len(r.TLS.PeerCertificates) > 0
			if !clientCerts {
				http.Error(w, "Forbidden: Client Certificate Required", http.StatusForbidden)
			} else {
				next.ServeHTTP(w, r)
			}
		}
	})
}
