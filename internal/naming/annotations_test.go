// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"testing"

	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestAnnotationsValid(t *testing.T) {
	assert.Assert(t, nil == validation.IsQualifiedName(AuthorizeBackupRemovalAnnotation))
	assert.Assert(t, nil == validation.IsQualifiedName(AutoCreateUserSchemaAnnotation))
	assert.Assert(t, nil == validation.IsQualifiedName(CrunchyBridgeClusterAdoptionAnnotation))
	assert.Assert(t, nil == validation.IsQualifiedName(Finalizer))
	assert.Assert(t, nil == validation.IsQualifiedName(PatroniSwitchover))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestBackup))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestBackupJobCompletion))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestConfigHash))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestCurrentConfig))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestIPVersion))
	assert.Assert(t, nil == validation.IsQualifiedName(PGBackRestRestore))
	assert.Assert(t, nil == validation.IsQualifiedName(PostgresExporterCollectorsAnnotation))
}
