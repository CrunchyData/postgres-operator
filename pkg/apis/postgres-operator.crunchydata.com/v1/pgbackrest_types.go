// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGBackRestArchive defines a pgBackRest archive configuration
// +kubebuilder:validation:XValidation:rule=`!has(self.log) || !has(self.log.path) || self.log.path.startsWith("/volumes/")`,message=`pgbackrest sidecar log path is restricted to an existing additional volume`
// +kubebuilder:validation:XValidation:rule=`!has(self.repoHost) || !has(self.repoHost.log) || !has(self.repoHost.log.path) || self.repoHost.volumes.additional.exists(x, self.repoHost.log.path.startsWith("/volumes/"+x.name))`,message=`repo host log path is restricted to an existing additional volume`
// +kubebuilder:validation:XValidation:rule=`!has(self.jobs) || !has(self.jobs.log) || !has(self.jobs.log.path) || self.jobs.volumes.additional.exists(x, self.jobs.log.path.startsWith("/volumes/"+x.name))`,message=`backup jobs log path is restricted to an existing additional volume`
type PGBackRestArchive struct {
	v1beta1.PGBackRestArchive `json:",inline"`
}
