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

package pki

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
)

func TestCertificateTextMarshaling(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		// Zero cannot marshal.
		_, err := Certificate{}.MarshalText()
		assert.ErrorContains(t, err, "malformed")

		// Empty cannot unmarshal.
		var sink Certificate
		assert.ErrorContains(t, sink.UnmarshalText(nil), "PEM-encoded")
		assert.ErrorContains(t, sink.UnmarshalText([]byte{}), "PEM-encoded")
	})

	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	cert := root.Certificate
	txt, err := cert.MarshalText()
	assert.NilError(t, err)
	assert.Assert(t, bytes.HasPrefix(txt, []byte("-----BEGIN CERTIFICATE-----\n")), "got %q", txt)
	assert.Assert(t, bytes.HasSuffix(txt, []byte("\n-----END CERTIFICATE-----\n")), "got %q", txt)

	t.Run("RoundTrip", func(t *testing.T) {
		var sink Certificate
		assert.NilError(t, sink.UnmarshalText(txt))
		assert.DeepEqual(t, cert, sink)
	})

	t.Run("Bundle", func(t *testing.T) {
		other, _ := NewRootCertificateAuthority()
		otherText, err := other.Certificate.MarshalText()
		assert.NilError(t, err)

		bundle := bytes.Join([][]byte{txt, otherText}, nil)

		// Only the first certificate of a bundle is parsed.
		var sink Certificate
		assert.NilError(t, sink.UnmarshalText(bundle))
		assert.DeepEqual(t, cert, sink)
	})

	t.Run("EncodedEmpty", func(t *testing.T) {
		txt := []byte("-----BEGIN CERTIFICATE-----\n\n-----END CERTIFICATE-----\n")

		var sink Certificate
		assert.ErrorContains(t, sink.UnmarshalText(txt), "malformed")
	})

	t.Run("EncodedGarbage", func(t *testing.T) {
		txt := []byte("-----BEGIN CERTIFICATE-----\nasdfasdf\n-----END CERTIFICATE-----\n")

		var sink Certificate
		assert.ErrorContains(t, sink.UnmarshalText(txt), "malformed")
	})
}

func TestPrivateKeyTextMarshaling(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		// Zero cannot marshal.
		_, err := PrivateKey{}.MarshalText()
		assert.ErrorContains(t, err, "unknown")

		// Empty cannot unmarshal.
		var sink PrivateKey
		assert.ErrorContains(t, sink.UnmarshalText(nil), "PEM-encoded")
		assert.ErrorContains(t, sink.UnmarshalText([]byte{}), "PEM-encoded")
	})

	root, err := NewRootCertificateAuthority()
	assert.NilError(t, err)

	key := root.PrivateKey
	txt, err := key.MarshalText()
	assert.NilError(t, err)
	assert.Assert(t, bytes.HasPrefix(txt, []byte("-----BEGIN EC PRIVATE KEY-----\n")), "got %q", txt)
	assert.Assert(t, bytes.HasSuffix(txt, []byte("\n-----END EC PRIVATE KEY-----\n")), "got %q", txt)

	t.Run("RoundTrip", func(t *testing.T) {
		var sink PrivateKey
		assert.NilError(t, sink.UnmarshalText(txt))
		assert.DeepEqual(t, key, sink)
	})

	t.Run("Bundle", func(t *testing.T) {
		other, _ := NewRootCertificateAuthority()
		otherText, err := other.PrivateKey.MarshalText()
		assert.NilError(t, err)

		bundle := bytes.Join([][]byte{txt, otherText}, nil)

		// Only the first key of a bundle is parsed.
		var sink PrivateKey
		assert.NilError(t, sink.UnmarshalText(bundle))
		assert.DeepEqual(t, key, sink)
	})

	t.Run("EncodedEmpty", func(t *testing.T) {
		txt := []byte("-----BEGIN EC PRIVATE KEY-----\n\n-----END EC PRIVATE KEY-----\n")

		var sink PrivateKey
		assert.ErrorContains(t, sink.UnmarshalText(txt), "asn1")
	})

	t.Run("EncodedGarbage", func(t *testing.T) {
		txt := []byte("-----BEGIN EC PRIVATE KEY-----\nasdfasdf\n-----END EC PRIVATE KEY-----\n")

		var sink PrivateKey
		assert.ErrorContains(t, sink.UnmarshalText(txt), "asn1")
	})
}
