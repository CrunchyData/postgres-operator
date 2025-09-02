// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PGBouncerPodSpec defines the desired state of a PgBouncer connection pooler.
// +kubebuilder:validation:XValidation:rule=`!has(self.config) || !has(self.config.global) || !has(self.config.global.logfile) || self.config.global.logfile.startsWith('/tmp/logs/pgbouncer/') || self.volumes.additional.exists(x, self.config.global.logfile.startsWith("/volumes/"+x.name))`,message=`logfile destination is restricted to '/tmp/logs/pgbouncer/' or an existing additional volume`
type PGBouncerPodSpec struct {
	v1beta1.PGBouncerPodSpec `json:",inline"`
}
