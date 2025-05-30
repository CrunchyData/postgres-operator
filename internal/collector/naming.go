// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

const CompactingProcessor = "groupbyattrs/compact"
const DebugExporter = "debug"
const LogsBatchProcessor = "batch/logs"
const OneSecondBatchProcessor = "batch/1s"
const SubSecondBatchProcessor = "batch/200ms"
const Prometheus = "prometheus/cpk-monitoring"
const PrometheusPort = 9187
const PGBouncerMetrics = "metrics/pgbouncer"
const PostgresMetrics = "metrics/postgres"
const PatroniMetrics = "metrics/patroni"
const ResourceDetectionProcessor = "resourcedetection"
const MonitoringUser = "ccp_monitoring"

const SqlQuery = "sqlquery"

// For slow queries, we'll use pgMonitor's default 5 minute interval.
// https://github.com/CrunchyData/pgmonitor-extension/blob/main/sql/matviews/matviews.sql
const FiveMinuteSqlQuery = "sqlquery/300s"

// We'll use pgMonitor's Prometheus collection interval for most queries.
// https://github.com/CrunchyData/pgmonitor/blob/development/prometheus/linux/crunchy-prometheus.yml
const FiveSecondSqlQuery = "sqlquery/5s"
