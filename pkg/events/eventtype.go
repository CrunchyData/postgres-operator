package events

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

import (
	"fmt"
	"time"
)

const (
	EventTopicAll       = "alltopic"
	EventTopicCluster   = "clustertopic"
	EventTopicBackup    = "backuptopic"
	EventTopicLoad      = "loadtopic"
	EventTopicUser      = "postgresusertopic"
	EventTopicPolicy    = "policytopic"
	EventTopicPgAdmin   = "pgadmintopic"
	EventTopicPgbouncer = "pgbouncertopic"
	EventTopicPGO       = "pgotopic"
	EventTopicPGOUser   = "pgousertopic"
	EventTopicUpgrade   = "upgradetopic"
)

const (
	EventReloadCluster                 = "ReloadCluster"
	EventPrimaryNotReady               = "PrimaryNotReady"
	EventPrimaryDeleted                = "PrimaryDeleted"
	EventCreateCluster                 = "CreateCluster"
	EventCreateClusterCompleted        = "CreateClusterCompleted"
	EventCreateClusterFailure          = "CreateClusterFailure"
	EventScaleCluster                  = "ScaleCluster"
	EventScaleClusterFailure           = "ScaleClusterFailure"
	EventScaleDownCluster              = "ScaleDownCluster"
	EventShutdownCluster               = "ShutdownCluster"
	EventRestoreCluster                = "RestoreCluster"
	EventRestoreClusterCompleted       = "RestoreClusterCompleted"
	EventUpgradeCluster                = "UpgradeCluster"
	EventUpgradeClusterCreateSubmitted = "UpgradeClusterCreateSubmitted"
	EventUpgradeClusterFailure         = "UpgradeClusterFailure"
	EventDeleteCluster                 = "DeleteCluster"
	EventDeleteClusterCompleted        = "DeleteClusterCompleted"

	EventCreateBackup          = "CreateBackup"
	EventCreateBackupCompleted = "CreateBackupCompleted"

	EventCreatePolicy = "CreatePolicy"
	EventApplyPolicy  = "ApplyPolicy"
	EventDeletePolicy = "DeletePolicy"

	EventCreatePgAdmin = "CreatePgAdmin"
	EventDeletePgAdmin = "DeletePgAdmin"

	EventCreatePgbouncer = "CreatePgbouncer"
	EventDeletePgbouncer = "DeletePgbouncer"
	EventUpdatePgbouncer = "UpdatePgbouncer"

	EventPGOCreateUser      = "PGOCreateUser"
	EventPGOUpdateUser      = "PGOUpdateUser"
	EventPGODeleteUser      = "PGODeleteUser"
	EventPGOCreateRole      = "PGOCreateRole"
	EventPGOUpdateRole      = "PGOUpdateRole"
	EventPGODeleteRole      = "PGODeleteRole"
	EventPGOStart           = "PGOStart"
	EventPGOStop            = "PGOStop"
	EventPGOUpdateConfig    = "PGOUpdateConfig"
	EventPGODeleteNamespace = "PGODeleteNamespace"
	EventPGOCreateNamespace = "PGOCreateNamespace"

	EventStandbyEnabled  = "StandbyEnabled"
	EventStandbyDisabled = "StandbyDisabled"
)

type EventHeader struct {
	EventType string
	Namespace string    `json:"namespace"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
	Topic     []string  `json:"topic"`
}

func (lvl EventHeader) String() string {
	msg := fmt.Sprintf("Event %s - ns [%s] - user [%s] topics [%v] timestamp [%s]", lvl.EventType, lvl.Namespace, lvl.Username, lvl.Topic, lvl.Timestamp)
	return msg
}

type EventInterface interface {
	GetHeader() EventHeader
	String() string
}

//----------------------------
type EventCreateClusterFailureFormat struct {
	EventHeader  `json:"eventheader"`
	Clustername  string `json:"clustername"`
	ErrorMessage string `json:"errormessage"`
	WorkflowID   string `json:"workflowid"`
}

func (p EventCreateClusterFailureFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateClusterFailureFormat) String() string {
	msg := fmt.Sprintf("Event %s - (create cluster failure) clustername %s workflow %s error %s", lvl.EventHeader, lvl.Clustername, lvl.WorkflowID, lvl.ErrorMessage)
	return msg
}

//----------------------------
type EventCreateClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	WorkflowID  string `json:"workflowid"`
}

func (p EventCreateClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s - (create cluster) clustername %s workflow %s", lvl.EventHeader, lvl.Clustername, lvl.WorkflowID)
	return msg
}

//----------------------------
type EventCreateClusterCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	WorkflowID  string `json:"workflowid"`
}

func (p EventCreateClusterCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateClusterCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s - (create cluster completed) clustername %s workflow %s", lvl.EventHeader, lvl.Clustername, lvl.WorkflowID)
	return msg
}

//----------------------------
type EventScaleClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Replicaname string `json:"replicaname"`
}

func (p EventScaleClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventScaleClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (scale) - clustername %s - replicaname %s", lvl.EventHeader, lvl.Clustername, lvl.Replicaname)
	return msg
}

//----------------------------
type EventScaleClusterFailureFormat struct {
	EventHeader  `json:"eventheader"`
	Clustername  string `json:"clustername"`
	Replicaname  string `json:"replicaname"`
	ErrorMessage string `json:"errormessage"`
}

func (p EventScaleClusterFailureFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventScaleClusterFailureFormat) String() string {
	msg := fmt.Sprintf("Event %s (scale failure) - clustername %s - replicaname %s error %s", lvl.EventHeader, lvl.Clustername, lvl.Replicaname, lvl.ErrorMessage)
	return msg
}

//----------------------------
type EventScaleDownClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Replicaname string `json:"replicaname"`
}

func (p EventScaleDownClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventScaleDownClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (scaledown) - clustername %s - replicaname %s", lvl.EventHeader, lvl.Clustername, lvl.Replicaname)
	return msg
}

//----------------------------
type EventUpgradeClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	WorkflowID  string `json:"workflowid"`
}

func (p EventUpgradeClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpgradeClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (upgrade) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventUpgradeClusterCreateFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	WorkflowID  string `json:"workflowid"`
}

func (p EventUpgradeClusterCreateFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpgradeClusterCreateFormat) String() string {
	msg := fmt.Sprintf("Event %s (upgraded pgcluster submitted for creation) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventUpgradeClusterFailureFormat struct {
	EventHeader  `json:"eventheader"`
	Clustername  string `json:"clustername"`
	WorkflowID   string `json:"workflowid"`
	ErrorMessage string `json:"errormessage"`
}

func (p EventUpgradeClusterFailureFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpgradeClusterFailureFormat) String() string {
	return fmt.Sprintf(
		"Event %s - (upgrade cluster failure) clustername %s workflow %s error %s",
		lvl.EventHeader, lvl.Clustername, lvl.WorkflowID, lvl.ErrorMessage)
}

//----------------------------
type EventDeleteClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventDeleteClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeleteClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventDeleteClusterCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventDeleteClusterCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeleteClusterCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete completed) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreateBackupFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	BackupType  string `json:"backuptype"`
}

func (p EventCreateBackupFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateBackupFormat) String() string {
	msg := fmt.Sprintf("Event %s (create backup) - clustername %s - backuptype %s", lvl.EventHeader, lvl.Clustername, lvl.BackupType)
	return msg
}

//----------------------------
type EventCreateBackupCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	BackupType  string `json:"backuptype"`
	Path        string `json:"path"`
}

func (p EventCreateBackupCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateBackupCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (create backup completed) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreatePolicyFormat struct {
	EventHeader `json:"eventheader"`
	Policyname  string `json:"policyname"`
}

func (p EventCreatePolicyFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreatePolicyFormat) String() string {
	msg := fmt.Sprintf("Event %s (create policy) - policy [%s]", lvl.EventHeader, lvl.Policyname)
	return msg
}

//----------------------------
type EventDeletePolicyFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Policyname  string `json:"policyname"`
}

func (p EventDeletePolicyFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeletePolicyFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete policy) - clustername %s - policy [%s]", lvl.EventHeader, lvl.Clustername, lvl.Policyname)
	return msg
}

//----------------------------
type EventApplyPolicyFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Policyname  string `json:"policyname"`
}

func (p EventApplyPolicyFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventApplyPolicyFormat) String() string {
	msg := fmt.Sprintf("Event %s (apply policy) - clustername %s - policy [%s]", lvl.EventHeader, lvl.Clustername, lvl.Policyname)
	return msg
}

//----------------------------
type EventCreatePgAdminFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventCreatePgAdminFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreatePgAdminFormat) String() string {
	msg := fmt.Sprintf("Event %s (create pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventDeletePgAdminFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventDeletePgAdminFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeletePgAdminFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreatePgbouncerFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventCreatePgbouncerFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreatePgbouncerFormat) String() string {
	msg := fmt.Sprintf("Event %s (create pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventDeletePgbouncerFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventDeletePgbouncerFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeletePgbouncerFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventUpdatePgbouncerFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventUpdatePgbouncerFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpdatePgbouncerFormat) String() string {
	msg := fmt.Sprintf("Event %s (update pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventRestoreClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventRestoreClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventRestoreClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (restore) - clustername %s ", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventRestoreClusterCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventRestoreClusterCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventRestoreClusterCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (restore completed) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventPrimaryNotReadyFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventPrimaryNotReadyFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPrimaryNotReadyFormat) String() string {
	msg := fmt.Sprintf("Event %s - (primary not ready) clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventPrimaryDeletedFormat struct {
	EventHeader    `json:"eventheader"`
	Clustername    string `json:"clustername"`
	Deploymentname string `json:"deploymentname"`
}

func (p EventPrimaryDeletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPrimaryDeletedFormat) String() string {
	msg := fmt.Sprintf("Event %s - (primary deleted) clustername %s deployment %s", lvl.EventHeader, lvl.Clustername, lvl.Deploymentname)
	return msg
}

//----------------------------
type EventClusterShutdownFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventClusterShutdownFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventClusterShutdownFormat) String() string {
	msg := fmt.Sprintf("Event %s - (cluster shutdown) clustername %s", lvl.EventHeader,
		lvl.Clustername)
	return msg
}

//----------------------------
type EventStandbyEnabledFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventStandbyEnabledFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventStandbyEnabledFormat) String() string {
	msg := fmt.Sprintf("Event %s - (standby mode enabled) clustername %s", lvl.EventHeader,
		lvl.Clustername)
	return msg
}

//----------------------------
type EventStandbyDisabledFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventStandbyDisabledFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventStandbyDisabledFormat) String() string {
	msg := fmt.Sprintf("Event %s - (standby mode disabled) clustername %s", lvl.EventHeader,
		lvl.Clustername)
	return msg
}

//----------------------------
type EventShutdownClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventShutdownClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventShutdownClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s - (cluster shutdown) clustername %s", lvl.EventHeader,
		lvl.Clustername)
	return msg
}
