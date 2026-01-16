// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGBackRestArchive defines a pgBackRest archive configuration
// +kubebuilder:validation:XValidation:rule=`!self.?log.path.hasValue() || self.log.path.startsWith("/volumes/")`,message=`pgbackrest sidecar log path is restricted to an existing additional volume`
// +kubebuilder:validation:XValidation:rule=`!self.?repoHost.log.path.hasValue() || self.repoHost.volumes.additional.exists(x, self.repoHost.log.path.startsWith("/volumes/"+x.name))`,message=`repo host log path is restricted to an existing additional volume`
// +kubebuilder:validation:XValidation:rule=`!self.?jobs.log.path.hasValue() || self.jobs.volumes.additional.exists(x, self.jobs.log.path.startsWith("/volumes/"+x.name))`,message=`backup jobs log path is restricted to an existing additional volume`
// +kubebuilder:validation:XValidation:rule=`!self.?global["log-path"].hasValue()`,message=`pgbackrest log-path must be set via the various log.path fields in the spec`
type PGBackRestArchive struct {
	v1beta1.PGBackRestArchive `json:",inline"`
}
