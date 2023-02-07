package cluster

/*
 Copyright 2020 - 2023 Crunchy Data Solutions, Inc.
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
	"testing"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
)

func TestIsPgBouncerTLSEnabled(t *testing.T) {
	cluster := &crv1.Pgcluster{
		Spec: crv1.PgclusterSpec{
			PgBouncer: crv1.PgBouncerSpec{},
			TLS:       crv1.TLSSpec{},
		},
	}

	t.Run("true", func(t *testing.T) {
		cluster.Spec.PgBouncer.TLSSecret = "pgbouncer-tls"
		cluster.Spec.TLS.CASecret = "ca"
		cluster.Spec.TLS.TLSSecret = "postgres-tls"

		if !isPgBouncerTLSEnabled(cluster) {
			t.Errorf("expected true")
		}
	})

	t.Run("false", func(t *testing.T) {
		t.Run("neither enabled", func(t *testing.T) {
			cluster.Spec.PgBouncer.TLSSecret = ""
			cluster.Spec.TLS.CASecret = ""
			cluster.Spec.TLS.TLSSecret = ""

			if isPgBouncerTLSEnabled(cluster) {
				t.Errorf("expected false")
			}
		})

		t.Run("postgres TLS enabled only", func(t *testing.T) {
			cluster.Spec.PgBouncer.TLSSecret = ""
			cluster.Spec.TLS.CASecret = "ca"
			cluster.Spec.TLS.TLSSecret = "postgres-tls"

			if isPgBouncerTLSEnabled(cluster) {
				t.Errorf("expected false")
			}
		})

		t.Run("pgbouncer TLS enabled only", func(t *testing.T) {
			cluster.Spec.PgBouncer.TLSSecret = "pgbouncer-tls"
			cluster.Spec.TLS.CASecret = ""
			cluster.Spec.TLS.TLSSecret = ""

			if isPgBouncerTLSEnabled(cluster) {
				t.Errorf("expected false")
			}
		})
	})
}
