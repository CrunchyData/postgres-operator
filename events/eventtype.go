package events

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	EventTopicPgbouncer = "pgbouncertopic"
	EventTopicPGO       = "pgotopic"
	EventTopicPGOUser   = "pgousertopic"
)
const (
	EventReloadCluster            = "ReloadCluster"
	EventPrimaryNotReady          = "PrimaryNotReady"
	EventPrimaryDeleted           = "PrimaryDeleted"
	EventCloneCluster             = "CloneCluster"
	EventCloneClusterCompleted    = "CloneClusterCompleted"
	EventCloneClusterFailure      = "CloneClusterFailure"
	EventCreateCluster            = "CreateCluster"
	EventCreateClusterCompleted   = "CreateClusterCompleted"
	EventCreateClusterFailure     = "CreateClusterFailure"
	EventScaleCluster             = "ScaleCluster"
	EventScaleClusterFailure      = "ScaleClusterFailure"
	EventScaleDownCluster         = "ScaleDownCluster"
	EventFailoverCluster          = "FailoverCluster"
	EventFailoverClusterCompleted = "FailoverClusterCompleted"
	EventRestoreCluster           = "RestoreCluster"
	EventRestoreClusterCompleted  = "RestoreClusterCompleted"
	EventUpgradeCluster           = "UpgradeCluster"
	EventUpgradeClusterCompleted  = "UpgradeClusterCompleted"
	EventDeleteCluster            = "DeleteCluster"
	EventDeleteClusterCompleted   = "DeleteClusterCompleted"
	EventCreateLabel              = "CreateLabel"
	EventLoad                     = "Load"
	EventLoadCompleted            = "LoadCompleted"
	EventBenchmark                = "Benchmark"
	EventBenchmarkCompleted       = "BenchmarkCompleted"

	EventCreateBackup          = "CreateBackup"
	EventCreateBackupCompleted = "CreateBackupCompleted"

	EventCreateUser         = "CreateUser"
	EventDeleteUser         = "DeleteUser"
	EventChangePasswordUser = "ChangePasswordUser"

	EventCreatePolicy = "CreatePolicy"
	EventApplyPolicy  = "ApplyPolicy"
	EventDeletePolicy = "DeletePolicy"

	EventCreatePgbouncer = "CreatePgbouncer"
	EventDeletePgbouncer = "DeletePgbouncer"

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
)

type EventHeader struct {
	EventType string    `json:eventtype`
	Namespace string    `json:"namespace"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
	Topic     []string  `json:"topic"`
}

func (lvl EventHeader) String() string {
	msg := fmt.Sprintf("Event %s - ns [%s] - user [%s] topics [%v] timestampe [%s]", lvl.EventType, lvl.Namespace, lvl.Username, lvl.Topic, lvl.Timestamp)
	return msg
}

type EventInterface interface {
	GetHeader() EventHeader
	String() string
}

//--------
type EventReloadClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventReloadClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventReloadClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s - (reload) name %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCloneClusterFailureFormat struct {
	EventHeader       `json:"eventheader"`
	SourceClusterName string `json:"sourceClusterName"`
	TargetClusterName string `json:"targetClusterName"`
	ErrorMessage      string `json:"errormessage"`
	WorkflowID        string `json:"workflowid"`
}

func (p EventCloneClusterFailureFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCloneClusterFailureFormat) String() string {
	return fmt.Sprintf(
		"Event %s - (clone cluster failure) sourceclustername %s targetclustername %s workflow %s error %s",
		lvl.EventHeader, lvl.SourceClusterName, lvl.TargetClusterName, lvl.WorkflowID, lvl.ErrorMessage)
}

//----------------------------
type EventCloneClusterFormat struct {
	EventHeader       `json:"eventheader"`
	SourceClusterName string `json:"sourceClusterName"`
	TargetClusterName string `json:"targetClusterName"`
	WorkflowID        string `json:"workflowid"`
}

func (p EventCloneClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCloneClusterFormat) String() string {
	return fmt.Sprintf(
		"Event %s - (Clone cluster) sourceclustername %s targetclustername %s workflow %s",
		lvl.EventHeader, lvl.SourceClusterName, lvl.TargetClusterName, lvl.WorkflowID)
}

//----------------------------
type EventCloneClusterCompletedFormat struct {
	EventHeader       `json:"eventheader"`
	SourceClusterName string `json:"sourceClusterName"`
	TargetClusterName string `json:"targetClusterName"`
	WorkflowID        string `json:"workflowid"`
}

func (p EventCloneClusterCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCloneClusterCompletedFormat) String() string {
	return fmt.Sprintf(
		"Event %s - (Clone cluster completed) sourceclustername %s targetclustername %s workflow %s",
		lvl.EventHeader, lvl.SourceClusterName, lvl.TargetClusterName, lvl.WorkflowID)
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
type EventFailoverClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Target      string `json:"target"`
}

func (p EventFailoverClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventFailoverClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (failover) - clustername %s - target %s", lvl.EventHeader, lvl.Clustername, lvl.Target)
	return msg
}

//----------------------------
type EventFailoverClusterCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Target      string `json:"target"`
}

func (p EventFailoverClusterCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventFailoverClusterCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (failover completed) - clustername %s - target %s", lvl.EventHeader, lvl.Clustername, lvl.Target)
	return msg
}

//----------------------------
type EventUpgradeClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventUpgradeClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpgradeClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (upgrade) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventUpgradeClusterCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventUpgradeClusterCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpgradeClusterCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (upgrade completed) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
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
type EventCreateUserFormat struct {
	EventHeader      `json:"eventheader"`
	Clustername      string `json:"clustername"`
	PostgresUsername string `json:"postgresusername"`
	PostgresPassword string `json:"postgrespassword"`
	Managed          bool   `json:"managed"`
}

func (p EventCreateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s (create user) - clustername %s - postgres user [%s]", lvl.EventHeader, lvl.Clustername, lvl.PostgresUsername)
	return msg
}

//----------------------------
type EventDeleteUserFormat struct {
	EventHeader      `json:"eventheader"`
	Clustername      string `json:"clustername"`
	PostgresUsername string `json:"postgresusername"`
	Managed          bool   `json:"managed"`
}

func (p EventDeleteUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeleteUserFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete user) - clustername %s - postgres user [%s]", lvl.EventHeader, lvl.Clustername, lvl.PostgresUsername)
	return msg
}

//----------------------------
type EventChangePasswordUserFormat struct {
	EventHeader      `json:"eventheader"`
	Clustername      string `json:"clustername"`
	PostgresUsername string `json:"postgresusername"`
	PostgresPassword string `json:"postgrespassword"`
}

func (p EventChangePasswordUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventChangePasswordUserFormat) String() string {
	msg := fmt.Sprintf("Event %s (change password user) - clustername %s - postgres user [%s]", lvl.EventHeader, lvl.Clustername, lvl.PostgresUsername)
	return msg
}

//----------------------------
type EventCreateLabelFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Label       string `json:"label"`
}

func (p EventCreateLabelFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateLabelFormat) String() string {
	msg := fmt.Sprintf("Event %s (create label) - clustername %s - label [%s]", lvl.EventHeader, lvl.Clustername, lvl.Label)
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
type EventLoadFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Loadconfig  string `json:"loadconfig"`
}

func (p EventLoadFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventLoadFormat) String() string {
	msg := fmt.Sprintf("Event %s (load) - clustername %s - load config [%s]", lvl.EventHeader, lvl.Clustername, lvl.Loadconfig)
	return msg
}

//----------------------------
type EventLoadCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Loadconfig  string `json:"loadconfig"`
}

func (p EventLoadCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventLoadCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (load completed) - clustername %s - load config [%s]", lvl.EventHeader, lvl.Clustername, lvl.Loadconfig)
	return msg
}

//----------------------------
type EventBenchmarkFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventBenchmarkFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventBenchmarkFormat) String() string {
	msg := fmt.Sprintf("Event %s (benchmark) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventBenchmarkCompletedFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (p EventBenchmarkCompletedFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventBenchmarkCompletedFormat) String() string {
	msg := fmt.Sprintf("Event %s (benchmark completed) - clustername %s", lvl.EventHeader, lvl.Clustername)
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
