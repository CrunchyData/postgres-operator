/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
