// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGBackRestArchive defines a pgBackRest archive configuration
// +kubebuilder:validation:XValidation:rule=`!has(self.repoHost) || !has(self.repoHost.log) || !has(self.repoHost.log.path) || self.repoHost.volumes.additional.exists(x, self.repoHost.log.path.startsWith("/volumes/"+x.name))`,message=`log path is restricted to an existing additional volume`
type PGBackRestArchive struct {
	v1beta1.PGBackRestArchive `json:",inline"`
}
