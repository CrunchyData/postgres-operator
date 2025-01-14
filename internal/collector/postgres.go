// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func NewConfigForPostgresPod(ctx context.Context, inCluster *v1beta1.PostgresCluster) *Config {
	config := NewConfig()

	EnablePatroniMetrics(ctx, inCluster, config)

	return config
}
