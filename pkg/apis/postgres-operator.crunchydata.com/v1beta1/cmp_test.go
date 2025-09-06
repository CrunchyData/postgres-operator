// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"bytes"

	"gotest.tools/v3/assert/cmp"
	"sigs.k8s.io/yaml"
)

// MarshalsTo converts x to YAML and compares that to y.
func MarshalsTo[T []byte | string](x any, y T) cmp.Comparison {
	b, err := yaml.Marshal(x)
	if err != nil {
		return func() cmp.Result { return cmp.ResultFromError(err) }
	}
	return cmp.DeepEqual(string(b), string(
		append(bytes.TrimLeft(bytes.TrimRight([]byte(y), "\t\n"), "\n"), '\n'),
	))
}
