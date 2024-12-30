// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import "k8s.io/apimachinery/pkg/api/resource"

type PatroniSpec struct {
	// Patroni dynamic configuration settings. Changes to this value will be
	// automatically reloaded without validation. Changes to certain PostgreSQL
	// parameters cause PostgreSQL to restart.
	// More info: https://patroni.readthedocs.io/en/latest/dynamic_configuration.html
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	DynamicConfiguration SchemalessObject `json:"dynamicConfiguration,omitempty"`

	// TTL of the cluster leader lock. "Think of it as the
	// length of time before initiation of the automatic failover process."
	// Changing this value causes PostgreSQL to restart.
	// +optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=3
	LeaderLeaseDurationSeconds *int32 `json:"leaderLeaseDurationSeconds,omitempty"`

	// Patroni log configuration settings.
	// +optional
	Logging *PatroniLogConfig `json:"logging,omitempty"`

	// The port on which Patroni should listen.
	// Changing this value causes PostgreSQL to restart.
	// +optional
	// +kubebuilder:default=8008
	// +kubebuilder:validation:Minimum=1024
	Port *int32 `json:"port,omitempty"`

	// The interval for refreshing the leader lock and applying
	// dynamicConfiguration. Must be less than leaderLeaseDurationSeconds.
	// Changing this value causes PostgreSQL to restart.
	// +optional
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=1
	SyncPeriodSeconds *int32 `json:"syncPeriodSeconds,omitempty"`

	// Switchover gives options to perform ad hoc switchovers in a PostgresCluster.
	// +optional
	Switchover *PatroniSwitchover `json:"switchover,omitempty"`

	// TODO(cbandy): Add UseConfigMaps bool, default false.
	// TODO(cbandy): Allow other DCS: etcd, raft, etc?
	// N.B. changing this will cause downtime.
	// - https://patroni.readthedocs.io/en/latest/kubernetes.html
}

type PatroniLogConfig struct {

	// Limits the total amount of space taken by Patroni log files.
	// Minimum value is 25MB.
	// More info: https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity
	// ---
	// TODO(validation) TODO(k8s-1.29): Validate the minimum using CEL libraries.
	//
	// +required
	StorageLimit *resource.Quantity `json:"storageLimit"`

	// The Patroni log level.
	// More info: https://docs.python.org/3/library/logging.html#levels
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=10
	//
	// +default="INFO"
	// +kubebuilder:validation:Enum={CRITICAL,ERROR,WARNING,INFO,DEBUG,NOTSET}
	// +optional
	Level *string `json:"level,omitempty"`
}

type PatroniSwitchover struct {

	// Whether or not the operator should allow switchovers in a PostgresCluster
	// +required
	Enabled bool `json:"enabled"`

	// The instance that should become primary during a switchover. This field is
	// optional when Type is "Switchover" and required when Type is "Failover".
	// When it is not specified, a healthy replica is automatically selected.
	// +optional
	TargetInstance *string `json:"targetInstance,omitempty"`

	// Type of switchover to perform. Valid options are Switchover and Failover.
	// "Switchover" changes the primary instance of a healthy PostgresCluster.
	// "Failover" forces a particular instance to be primary, regardless of other
	// factors. A TargetInstance must be specified to failover.
	// NOTE: The Failover type is reserved as the "last resort" case.
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	//
	// +kubebuilder:validation:Enum={Switchover,Failover}
	// +kubebuilder:default:=Switchover
	// +optional
	Type string `json:"type,omitempty"`
}

// PatroniSwitchover types.
const (
	PatroniSwitchoverTypeFailover   = "Failover"
	PatroniSwitchoverTypeSwitchover = "Switchover"
)

// Default sets the default values for certain Patroni configuration attributes,
// including:
// - Lock Lease Duration
// - Patroni's API port
// - Frequency of syncing with Kube API
func (s *PatroniSpec) Default() {
	if s.LeaderLeaseDurationSeconds == nil {
		s.LeaderLeaseDurationSeconds = new(int32)
		*s.LeaderLeaseDurationSeconds = 30
	}
	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 8008
	}
	if s.SyncPeriodSeconds == nil {
		s.SyncPeriodSeconds = new(int32)
		*s.SyncPeriodSeconds = 10
	}
}

type PatroniStatus struct {

	// - "database_system_identifier" of https://github.com/zalando/patroni/blob/v2.0.1/docs/rest_api.rst#monitoring-endpoint
	// - "system_identifier" of https://www.postgresql.org/docs/current/functions-info.html#FUNCTIONS-PG-CONTROL-SYSTEM
	// - "systemid" of https://www.postgresql.org/docs/current/protocol-replication.html

	// The PostgreSQL system identifier reported by Patroni.
	// +optional
	SystemIdentifier string `json:"systemIdentifier,omitempty"`

	// Tracks the execution of the switchover requests.
	// +optional
	Switchover *string `json:"switchover,omitempty"`

	// Tracks the current timeline during switchovers
	// +optional
	SwitchoverTimeline *int64 `json:"switchoverTimeline,omitempty"`
}
