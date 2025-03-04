// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestOptionalConfigMapKeyRefAsProjection(t *testing.T) {
	t.Run("Null", func(t *testing.T) {
		in := v1beta1.OptionalConfigMapKeyRef{}
		in.Name, in.Key = "one", "two"

		out := in.AsProjection("three")
		b, err := yaml.Marshal(out)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: two
  path: three
name: one
		`)+"\n")
	})

	t.Run("True", func(t *testing.T) {
		True := true
		in := v1beta1.OptionalConfigMapKeyRef{Optional: &True}
		in.Name, in.Key = "one", "two"

		out := in.AsProjection("three")
		b, err := yaml.Marshal(out)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: two
  path: three
name: one
optional: true
		`)+"\n")
	})

	t.Run("False", func(t *testing.T) {
		False := false
		in := v1beta1.OptionalConfigMapKeyRef{Optional: &False}
		in.Name, in.Key = "one", "two"

		out := in.AsProjection("three")
		b, err := yaml.Marshal(out)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: two
  path: three
name: one
optional: false
		`)+"\n")
	})
}

func TestConfigMapKeyRefAsProjection(t *testing.T) {
	in := v1beta1.ConfigMapKeyRef{Name: "asdf", Key: "foobar"}
	out := in.AsProjection("some-path")

	b, err := yaml.Marshal(out)
	assert.NilError(t, err)
	assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: foobar
  path: some-path
name: asdf
	`)+"\n")
}

func TestOptionalSecretKeyRefAsProjection(t *testing.T) {
	t.Run("Null", func(t *testing.T) {
		in := v1beta1.OptionalSecretKeyRef{}
		in.Name, in.Key = "one", "two"

		out := in.AsProjection("three")
		b, err := yaml.Marshal(out)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: two
  path: three
name: one
		`)+"\n")
	})

	t.Run("True", func(t *testing.T) {
		True := true
		in := v1beta1.OptionalSecretKeyRef{Optional: &True}
		in.Name, in.Key = "one", "two"

		out := in.AsProjection("three")
		b, err := yaml.Marshal(out)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: two
  path: three
name: one
optional: true
		`)+"\n")
	})

	t.Run("False", func(t *testing.T) {
		False := false
		in := v1beta1.OptionalSecretKeyRef{Optional: &False}
		in.Name, in.Key = "one", "two"

		out := in.AsProjection("three")
		b, err := yaml.Marshal(out)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: two
  path: three
name: one
optional: false
		`)+"\n")
	})
}

func TestSecretKeyRefAsProjection(t *testing.T) {
	in := v1beta1.SecretKeyRef{Name: "asdf", Key: "foobar"}
	out := in.AsProjection("some-path")

	b, err := yaml.Marshal(out)
	assert.NilError(t, err)
	assert.DeepEqual(t, string(b), strings.TrimSpace(`
items:
- key: foobar
  path: some-path
name: asdf
	`)+"\n")
}
