// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgupgrade

import (
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// ConditionPGUpgradeProgressing is the type used in a condition to indicate that
	// an Postgres major upgrade is in progress.
	ConditionPGUpgradeProgressing = "Progressing"

	// ConditionPGUpgradeSucceeded is the type used in a condition to indicate the
	// status of a Postgres major upgrade.
	ConditionPGUpgradeSucceeded = "Succeeded"

	labelPrefix           = "postgres-operator.crunchydata.com/"
	LabelPGUpgrade        = labelPrefix + "pgupgrade"
	LabelCluster          = labelPrefix + "cluster"
	LabelRole             = labelPrefix + "role"
	LabelVersion          = labelPrefix + "version"
	LabelPatroni          = labelPrefix + "patroni"
	LabelPGBackRestBackup = labelPrefix + "pgbackrest-backup"
	LabelInstance         = labelPrefix + "instance"

	ReplicaCreate     = "replica-create"
	ContainerDatabase = "database"

	pgUpgrade  = "pgupgrade"
	removeData = "removedata"
)

func commonLabels(role string, upgrade *v1beta1.PGUpgrade) map[string]string {
	return map[string]string{
		LabelPGUpgrade: upgrade.Name,
		LabelCluster:   upgrade.Spec.PostgresClusterName,
		LabelRole:      role,
	}
}
