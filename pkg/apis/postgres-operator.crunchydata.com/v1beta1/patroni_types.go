/*
 Copyright 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

type PatroniSpec struct {
	// TODO(cbandy): Find a better way to have a map[string]interface{} here.
	// See: https://github.com/kubernetes-sigs/controller-tools/commit/557da250b8

	// Patroni dynamic configuration settings. Changes to this value will be
	// automatically reloaded without validation. Changes to certain PostgreSQL
	// parameters cause PostgreSQL to restart.
	// More info: https://patroni.readthedocs.io/en/latest/SETTINGS.html
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	DynamicConfiguration runtime.RawExtension `json:"dynamicConfiguration,omitempty"`

	// TTL of the cluster leader lock. "Think of it as the
	// length of time before initiation of the automatic failover process."
	// Changing this value causes PostgreSQL to restart.
	// +optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=3
	LeaderLeaseDurationSeconds *int32 `json:"leaderLeaseDurationSeconds,omitempty"`

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

type PatroniSwitchover struct {

	// Whether or not the operator should allow switchovers in a PostgresCluster
	// +required
	Enabled bool `json:"enabled"`

	// Define the instance that the operator will target in a switchover. When attempting to
	// perform a manual switchover this field is optional. If target is specified, we will attempt
	// to get to an instance that represents that target. If it is not specified, then we will
	// attempt to get any instance. When attempting to perform a failover (i.e. Switchover.Type
	// is `failover`) this field is required.
	// +optional
	TargetInstance *string `json:"targetInstance,omitempty"`

	// Type allows you to specify the type of Patroni switchover that will be performed.
	// `patronictl` supports both `switchovers` and `failovers` where a `failover` is
	// effectively a "forced switchover". The main difference is that `failover` can be
	// used when there is not currently a leader. A TargetInstance must be specified to
	// failover.
	// NOTE: The switchover type failover is reserved as the "last resort" case.
	// +kubebuilder:validation:Enum={switchover,failover}
	// +kubebuilder:default:=switchover
	// +optional
	Type string `json:"type,omitempty"`
}

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
}
