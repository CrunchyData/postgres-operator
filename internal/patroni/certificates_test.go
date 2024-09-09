// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

type funcMarshaler func() ([]byte, error)

func (f funcMarshaler) MarshalText() ([]byte, error) { return f() }

func TestCertFile(t *testing.T) {
	expected := errors.New("boom")
	var short funcMarshaler = func() ([]byte, error) { return []byte(`one`), nil }
	var fail funcMarshaler = func() ([]byte, error) { return nil, expected }

	text, err := certFile(short, short, short)
	assert.NilError(t, err)
	assert.DeepEqual(t, text, []byte(`oneoneone`))

	text, err = certFile(short, fail, short)
	assert.Equal(t, err, expected)
	assert.DeepEqual(t, text, []byte(nil))
}

func TestInstanceCertificates(t *testing.T) {
	certs := new(corev1.Secret)
	certs.Name = "some-name"

	projections := instanceCertificates(certs)

	assert.Assert(t, cmp.MarshalMatches(projections, `
- secret:
    items:
    - key: patroni.ca-roots
      path: ~postgres-operator/patroni.ca-roots
    - key: patroni.crt-combined
      path: ~postgres-operator/patroni.crt+key
    name: some-name
	`))
}
