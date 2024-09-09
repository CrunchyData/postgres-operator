// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package initialize_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

func TestIntOrStringInt32(t *testing.T) {
	// Same content as the upstream constructor.
	upstream := intstr.FromInt(42)
	n := initialize.IntOrStringInt32(42)

	assert.DeepEqual(t, &upstream, n)
}

func TestIntOrStringString(t *testing.T) {
	upstream := intstr.FromString("50%")
	s := initialize.IntOrStringString("50%")

	assert.DeepEqual(t, &upstream, s)
}
func TestIntOrString(t *testing.T) {
	upstream := intstr.FromInt(0)

	ios := initialize.IntOrString(intstr.FromInt(0))
	assert.DeepEqual(t, *ios, upstream)
}
