// Copyright 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestConfigToYAML(t *testing.T) {
	result, err := NewConfig().ToYAML()
	assert.NilError(t, err)
	assert.DeepEqual(t, result, `# Generated by postgres-operator. DO NOT EDIT.
# Your changes will not be saved.
exporters:
  debug:
    verbosity: detailed
extensions: {}
processors:
  batch/1s:
    timeout: 1s
  batch/200ms:
    timeout: 200ms
  groupbyattrs/compact: {}
receivers: {}
service:
  extensions: []
  pipelines: {}
`)
}
