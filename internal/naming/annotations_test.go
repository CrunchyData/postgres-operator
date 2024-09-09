// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestAnnotationsValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsQualifiedName(Finalizer))
	assert.Assert(t, nil == validation.IsQualifiedName(PatroniSwitchover))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestBackup))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestBackupJobId))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestConfigHash))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestRestore))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestIPVersion))
	assert.Assert(t, nil == validation.IsQualifiedName(PostgresExporterCollectorsAnnotation))
	assert.Assert(t, nil == validation.IsQualifiedName(CrunchyBridgeClusterAdoptionAnnotation))
}
