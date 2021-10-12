//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupJobs) DeepCopyInto(out *BackupJobs) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
	if in.PriorityClassName != nil {
		in, out := &in.PriorityClassName, &out.PriorityClassName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupJobs.
func (in *BackupJobs) DeepCopy() *BackupJobs {
	if in == nil {
		return nil
	}
	out := new(BackupJobs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Backups) DeepCopyInto(out *Backups) {
	*out = *in
	in.PGBackRest.DeepCopyInto(&out.PGBackRest)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Backups.
func (in *Backups) DeepCopy() *Backups {
	if in == nil {
		return nil
	}
	out := new(Backups)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataSource) DeepCopyInto(out *DataSource) {
	*out = *in
	if in.PostgresCluster != nil {
		in, out := &in.PostgresCluster, &out.PostgresCluster
		*out = new(PostgresClusterDataSource)
		(*in).DeepCopyInto(*out)
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = new(DataSourceVolumes)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataSource.
func (in *DataSource) DeepCopy() *DataSource {
	if in == nil {
		return nil
	}
	out := new(DataSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataSourceVolume) DeepCopyInto(out *DataSourceVolume) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataSourceVolume.
func (in *DataSourceVolume) DeepCopy() *DataSourceVolume {
	if in == nil {
		return nil
	}
	out := new(DataSourceVolume)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataSourceVolumes) DeepCopyInto(out *DataSourceVolumes) {
	*out = *in
	if in.PGDataVolume != nil {
		in, out := &in.PGDataVolume, &out.PGDataVolume
		*out = new(DataSourceVolume)
		**out = **in
	}
	if in.PGWALVolume != nil {
		in, out := &in.PGWALVolume, &out.PGWALVolume
		*out = new(DataSourceVolume)
		**out = **in
	}
	if in.PGBackRestVolume != nil {
		in, out := &in.PGBackRestVolume, &out.PGBackRestVolume
		*out = new(DataSourceVolume)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataSourceVolumes.
func (in *DataSourceVolumes) DeepCopy() *DataSourceVolumes {
	if in == nil {
		return nil
	}
	out := new(DataSourceVolumes)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseInitSQL) DeepCopyInto(out *DatabaseInitSQL) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseInitSQL.
func (in *DatabaseInitSQL) DeepCopy() *DatabaseInitSQL {
	if in == nil {
		return nil
	}
	out := new(DatabaseInitSQL)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExporterSpec) DeepCopyInto(out *ExporterSpec) {
	*out = *in
	if in.Configuration != nil {
		in, out := &in.Configuration, &out.Configuration
		*out = make([]v1.VolumeProjection, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExporterSpec.
func (in *ExporterSpec) DeepCopy() *ExporterSpec {
	if in == nil {
		return nil
	}
	out := new(ExporterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceSidecars) DeepCopyInto(out *InstanceSidecars) {
	*out = *in
	if in.ReplicaCertCopy != nil {
		in, out := &in.ReplicaCertCopy, &out.ReplicaCertCopy
		*out = new(Sidecar)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceSidecars.
func (in *InstanceSidecars) DeepCopy() *InstanceSidecars {
	if in == nil {
		return nil
	}
	out := new(InstanceSidecars)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metadata) DeepCopyInto(out *Metadata) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metadata.
func (in *Metadata) DeepCopy() *Metadata {
	if in == nil {
		return nil
	}
	out := new(Metadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MonitoringSpec) DeepCopyInto(out *MonitoringSpec) {
	*out = *in
	if in.PGMonitor != nil {
		in, out := &in.PGMonitor, &out.PGMonitor
		*out = new(PGMonitorSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MonitoringSpec.
func (in *MonitoringSpec) DeepCopy() *MonitoringSpec {
	if in == nil {
		return nil
	}
	out := new(MonitoringSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MonitoringStatus) DeepCopyInto(out *MonitoringStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MonitoringStatus.
func (in *MonitoringStatus) DeepCopy() *MonitoringStatus {
	if in == nil {
		return nil
	}
	out := new(MonitoringStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestArchive) DeepCopyInto(out *PGBackRestArchive) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(Metadata)
		(*in).DeepCopyInto(*out)
	}
	if in.Configuration != nil {
		in, out := &in.Configuration, &out.Configuration
		*out = make([]v1.VolumeProjection, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Global != nil {
		in, out := &in.Global, &out.Global
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Jobs != nil {
		in, out := &in.Jobs, &out.Jobs
		*out = new(BackupJobs)
		(*in).DeepCopyInto(*out)
	}
	if in.Repos != nil {
		in, out := &in.Repos, &out.Repos
		*out = make([]PGBackRestRepo, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RepoHost != nil {
		in, out := &in.RepoHost, &out.RepoHost
		*out = new(PGBackRestRepoHost)
		(*in).DeepCopyInto(*out)
	}
	if in.Manual != nil {
		in, out := &in.Manual, &out.Manual
		*out = new(PGBackRestManualBackup)
		(*in).DeepCopyInto(*out)
	}
	if in.Restore != nil {
		in, out := &in.Restore, &out.Restore
		*out = new(PGBackRestRestore)
		(*in).DeepCopyInto(*out)
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = new(PGBackRestSidecars)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestArchive.
func (in *PGBackRestArchive) DeepCopy() *PGBackRestArchive {
	if in == nil {
		return nil
	}
	out := new(PGBackRestArchive)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestBackupSchedules) DeepCopyInto(out *PGBackRestBackupSchedules) {
	*out = *in
	if in.Full != nil {
		in, out := &in.Full, &out.Full
		*out = new(string)
		**out = **in
	}
	if in.Differential != nil {
		in, out := &in.Differential, &out.Differential
		*out = new(string)
		**out = **in
	}
	if in.Incremental != nil {
		in, out := &in.Incremental, &out.Incremental
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestBackupSchedules.
func (in *PGBackRestBackupSchedules) DeepCopy() *PGBackRestBackupSchedules {
	if in == nil {
		return nil
	}
	out := new(PGBackRestBackupSchedules)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestJobStatus) DeepCopyInto(out *PGBackRestJobStatus) {
	*out = *in
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		*out = (*in).DeepCopy()
	}
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestJobStatus.
func (in *PGBackRestJobStatus) DeepCopy() *PGBackRestJobStatus {
	if in == nil {
		return nil
	}
	out := new(PGBackRestJobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestManualBackup) DeepCopyInto(out *PGBackRestManualBackup) {
	*out = *in
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestManualBackup.
func (in *PGBackRestManualBackup) DeepCopy() *PGBackRestManualBackup {
	if in == nil {
		return nil
	}
	out := new(PGBackRestManualBackup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestRepo) DeepCopyInto(out *PGBackRestRepo) {
	*out = *in
	if in.BackupSchedules != nil {
		in, out := &in.BackupSchedules, &out.BackupSchedules
		*out = new(PGBackRestBackupSchedules)
		(*in).DeepCopyInto(*out)
	}
	if in.Azure != nil {
		in, out := &in.Azure, &out.Azure
		*out = new(RepoAzure)
		**out = **in
	}
	if in.GCS != nil {
		in, out := &in.GCS, &out.GCS
		*out = new(RepoGCS)
		**out = **in
	}
	if in.S3 != nil {
		in, out := &in.S3, &out.S3
		*out = new(RepoS3)
		**out = **in
	}
	if in.Volume != nil {
		in, out := &in.Volume, &out.Volume
		*out = new(RepoPVC)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestRepo.
func (in *PGBackRestRepo) DeepCopy() *PGBackRestRepo {
	if in == nil {
		return nil
	}
	out := new(PGBackRestRepo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestRepoHost) DeepCopyInto(out *PGBackRestRepoHost) {
	*out = *in
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PriorityClassName != nil {
		in, out := &in.PriorityClassName, &out.PriorityClassName
		*out = new(string)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]v1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SSHConfiguration != nil {
		in, out := &in.SSHConfiguration, &out.SSHConfiguration
		*out = new(v1.ConfigMapProjection)
		(*in).DeepCopyInto(*out)
	}
	if in.SSHSecret != nil {
		in, out := &in.SSHSecret, &out.SSHSecret
		*out = new(v1.SecretProjection)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestRepoHost.
func (in *PGBackRestRepoHost) DeepCopy() *PGBackRestRepoHost {
	if in == nil {
		return nil
	}
	out := new(PGBackRestRepoHost)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestRestore) DeepCopyInto(out *PGBackRestRestore) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.PostgresClusterDataSource != nil {
		in, out := &in.PostgresClusterDataSource, &out.PostgresClusterDataSource
		*out = new(PostgresClusterDataSource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestRestore.
func (in *PGBackRestRestore) DeepCopy() *PGBackRestRestore {
	if in == nil {
		return nil
	}
	out := new(PGBackRestRestore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestScheduledBackupStatus) DeepCopyInto(out *PGBackRestScheduledBackupStatus) {
	*out = *in
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		*out = (*in).DeepCopy()
	}
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestScheduledBackupStatus.
func (in *PGBackRestScheduledBackupStatus) DeepCopy() *PGBackRestScheduledBackupStatus {
	if in == nil {
		return nil
	}
	out := new(PGBackRestScheduledBackupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestSidecars) DeepCopyInto(out *PGBackRestSidecars) {
	*out = *in
	if in.PGBackRest != nil {
		in, out := &in.PGBackRest, &out.PGBackRest
		*out = new(Sidecar)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestSidecars.
func (in *PGBackRestSidecars) DeepCopy() *PGBackRestSidecars {
	if in == nil {
		return nil
	}
	out := new(PGBackRestSidecars)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBackRestStatus) DeepCopyInto(out *PGBackRestStatus) {
	*out = *in
	if in.ManualBackup != nil {
		in, out := &in.ManualBackup, &out.ManualBackup
		*out = new(PGBackRestJobStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.ScheduledBackups != nil {
		in, out := &in.ScheduledBackups, &out.ScheduledBackups
		*out = make([]PGBackRestScheduledBackupStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RepoHost != nil {
		in, out := &in.RepoHost, &out.RepoHost
		*out = new(RepoHostStatus)
		**out = **in
	}
	if in.Repos != nil {
		in, out := &in.Repos, &out.Repos
		*out = make([]RepoStatus, len(*in))
		copy(*out, *in)
	}
	if in.Restore != nil {
		in, out := &in.Restore, &out.Restore
		*out = new(PGBackRestJobStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBackRestStatus.
func (in *PGBackRestStatus) DeepCopy() *PGBackRestStatus {
	if in == nil {
		return nil
	}
	out := new(PGBackRestStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBouncerConfiguration) DeepCopyInto(out *PGBouncerConfiguration) {
	*out = *in
	if in.Files != nil {
		in, out := &in.Files, &out.Files
		*out = make([]v1.VolumeProjection, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Global != nil {
		in, out := &in.Global, &out.Global
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Databases != nil {
		in, out := &in.Databases, &out.Databases
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Users != nil {
		in, out := &in.Users, &out.Users
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBouncerConfiguration.
func (in *PGBouncerConfiguration) DeepCopy() *PGBouncerConfiguration {
	if in == nil {
		return nil
	}
	out := new(PGBouncerConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBouncerPodSpec) DeepCopyInto(out *PGBouncerPodSpec) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(Metadata)
		(*in).DeepCopyInto(*out)
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	in.Config.DeepCopyInto(&out.Config)
	if in.CustomTLSSecret != nil {
		in, out := &in.CustomTLSSecret, &out.CustomTLSSecret
		*out = new(v1.SecretProjection)
		(*in).DeepCopyInto(*out)
	}
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int32)
		**out = **in
	}
	if in.PriorityClassName != nil {
		in, out := &in.PriorityClassName, &out.PriorityClassName
		*out = new(string)
		**out = **in
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceSpec)
		**out = **in
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = new(PGBouncerSidecars)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]v1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBouncerPodSpec.
func (in *PGBouncerPodSpec) DeepCopy() *PGBouncerPodSpec {
	if in == nil {
		return nil
	}
	out := new(PGBouncerPodSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBouncerPodStatus) DeepCopyInto(out *PGBouncerPodStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBouncerPodStatus.
func (in *PGBouncerPodStatus) DeepCopy() *PGBouncerPodStatus {
	if in == nil {
		return nil
	}
	out := new(PGBouncerPodStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGBouncerSidecars) DeepCopyInto(out *PGBouncerSidecars) {
	*out = *in
	if in.PGBouncerConfig != nil {
		in, out := &in.PGBouncerConfig, &out.PGBouncerConfig
		*out = new(Sidecar)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGBouncerSidecars.
func (in *PGBouncerSidecars) DeepCopy() *PGBouncerSidecars {
	if in == nil {
		return nil
	}
	out := new(PGBouncerSidecars)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PGMonitorSpec) DeepCopyInto(out *PGMonitorSpec) {
	*out = *in
	if in.Exporter != nil {
		in, out := &in.Exporter, &out.Exporter
		*out = new(ExporterSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PGMonitorSpec.
func (in *PGMonitorSpec) DeepCopy() *PGMonitorSpec {
	if in == nil {
		return nil
	}
	out := new(PGMonitorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PatroniSpec) DeepCopyInto(out *PatroniSpec) {
	*out = *in
	in.DynamicConfiguration.DeepCopyInto(&out.DynamicConfiguration)
	if in.LeaderLeaseDurationSeconds != nil {
		in, out := &in.LeaderLeaseDurationSeconds, &out.LeaderLeaseDurationSeconds
		*out = new(int32)
		**out = **in
	}
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int32)
		**out = **in
	}
	if in.SyncPeriodSeconds != nil {
		in, out := &in.SyncPeriodSeconds, &out.SyncPeriodSeconds
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PatroniSpec.
func (in *PatroniSpec) DeepCopy() *PatroniSpec {
	if in == nil {
		return nil
	}
	out := new(PatroniSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PatroniStatus) DeepCopyInto(out *PatroniStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PatroniStatus.
func (in *PatroniStatus) DeepCopy() *PatroniStatus {
	if in == nil {
		return nil
	}
	out := new(PatroniStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresCluster) DeepCopyInto(out *PostgresCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresCluster.
func (in *PostgresCluster) DeepCopy() *PostgresCluster {
	if in == nil {
		return nil
	}
	out := new(PostgresCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresClusterDataSource) DeepCopyInto(out *PostgresClusterDataSource) {
	*out = *in
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PriorityClassName != nil {
		in, out := &in.PriorityClassName, &out.PriorityClassName
		*out = new(string)
		**out = **in
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresClusterDataSource.
func (in *PostgresClusterDataSource) DeepCopy() *PostgresClusterDataSource {
	if in == nil {
		return nil
	}
	out := new(PostgresClusterDataSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresClusterList) DeepCopyInto(out *PostgresClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgresCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresClusterList.
func (in *PostgresClusterList) DeepCopy() *PostgresClusterList {
	if in == nil {
		return nil
	}
	out := new(PostgresClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresClusterSpec) DeepCopyInto(out *PostgresClusterSpec) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(Metadata)
		(*in).DeepCopyInto(*out)
	}
	if in.DataSource != nil {
		in, out := &in.DataSource, &out.DataSource
		*out = new(DataSource)
		(*in).DeepCopyInto(*out)
	}
	in.Backups.DeepCopyInto(&out.Backups)
	if in.CustomTLSSecret != nil {
		in, out := &in.CustomTLSSecret, &out.CustomTLSSecret
		*out = new(v1.SecretProjection)
		(*in).DeepCopyInto(*out)
	}
	if in.CustomReplicationClientTLSSecret != nil {
		in, out := &in.CustomReplicationClientTLSSecret, &out.CustomReplicationClientTLSSecret
		*out = new(v1.SecretProjection)
		(*in).DeepCopyInto(*out)
	}
	if in.DatabaseInitSQL != nil {
		in, out := &in.DatabaseInitSQL, &out.DatabaseInitSQL
		*out = new(DatabaseInitSQL)
		**out = **in
	}
	if in.DisableDefaultPodScheduling != nil {
		in, out := &in.DisableDefaultPodScheduling, &out.DisableDefaultPodScheduling
		*out = new(bool)
		**out = **in
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.InstanceSets != nil {
		in, out := &in.InstanceSets, &out.InstanceSets
		*out = make([]PostgresInstanceSetSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.OpenShift != nil {
		in, out := &in.OpenShift, &out.OpenShift
		*out = new(bool)
		**out = **in
	}
	if in.Patroni != nil {
		in, out := &in.Patroni, &out.Patroni
		*out = new(PatroniSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int32)
		**out = **in
	}
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(PostgresProxySpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Monitoring != nil {
		in, out := &in.Monitoring, &out.Monitoring
		*out = new(MonitoringSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceSpec)
		**out = **in
	}
	if in.Shutdown != nil {
		in, out := &in.Shutdown, &out.Shutdown
		*out = new(bool)
		**out = **in
	}
	if in.Standby != nil {
		in, out := &in.Standby, &out.Standby
		*out = new(PostgresStandbySpec)
		**out = **in
	}
	if in.SupplementalGroups != nil {
		in, out := &in.SupplementalGroups, &out.SupplementalGroups
		*out = make([]int64, len(*in))
		copy(*out, *in)
	}
	if in.Users != nil {
		in, out := &in.Users, &out.Users
		*out = make([]PostgresUserSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresClusterSpec.
func (in *PostgresClusterSpec) DeepCopy() *PostgresClusterSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresClusterStatus) DeepCopyInto(out *PostgresClusterStatus) {
	*out = *in
	if in.InstanceSets != nil {
		in, out := &in.InstanceSets, &out.InstanceSets
		*out = make([]PostgresInstanceSetStatus, len(*in))
		copy(*out, *in)
	}
	if in.Patroni != nil {
		in, out := &in.Patroni, &out.Patroni
		*out = new(PatroniStatus)
		**out = **in
	}
	if in.PGBackRest != nil {
		in, out := &in.PGBackRest, &out.PGBackRest
		*out = new(PGBackRestStatus)
		(*in).DeepCopyInto(*out)
	}
	out.Proxy = in.Proxy
	out.Monitoring = in.Monitoring
	if in.DatabaseInitSQL != nil {
		in, out := &in.DatabaseInitSQL, &out.DatabaseInitSQL
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresClusterStatus.
func (in *PostgresClusterStatus) DeepCopy() *PostgresClusterStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresInstanceSetSpec) DeepCopyInto(out *PostgresInstanceSetSpec) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(Metadata)
		(*in).DeepCopyInto(*out)
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	in.DataVolumeClaimSpec.DeepCopyInto(&out.DataVolumeClaimSpec)
	if in.PriorityClassName != nil {
		in, out := &in.PriorityClassName, &out.PriorityClassName
		*out = new(string)
		**out = **in
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = new(InstanceSidecars)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]v1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.WALVolumeClaimSpec != nil {
		in, out := &in.WALVolumeClaimSpec, &out.WALVolumeClaimSpec
		*out = new(v1.PersistentVolumeClaimSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresInstanceSetSpec.
func (in *PostgresInstanceSetSpec) DeepCopy() *PostgresInstanceSetSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresInstanceSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresInstanceSetStatus) DeepCopyInto(out *PostgresInstanceSetStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresInstanceSetStatus.
func (in *PostgresInstanceSetStatus) DeepCopy() *PostgresInstanceSetStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresInstanceSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresProxySpec) DeepCopyInto(out *PostgresProxySpec) {
	*out = *in
	if in.PGBouncer != nil {
		in, out := &in.PGBouncer, &out.PGBouncer
		*out = new(PGBouncerPodSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresProxySpec.
func (in *PostgresProxySpec) DeepCopy() *PostgresProxySpec {
	if in == nil {
		return nil
	}
	out := new(PostgresProxySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresProxyStatus) DeepCopyInto(out *PostgresProxyStatus) {
	*out = *in
	out.PGBouncer = in.PGBouncer
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresProxyStatus.
func (in *PostgresProxyStatus) DeepCopy() *PostgresProxyStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresProxyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresStandbySpec) DeepCopyInto(out *PostgresStandbySpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresStandbySpec.
func (in *PostgresStandbySpec) DeepCopy() *PostgresStandbySpec {
	if in == nil {
		return nil
	}
	out := new(PostgresStandbySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresUserSpec) DeepCopyInto(out *PostgresUserSpec) {
	*out = *in
	if in.Databases != nil {
		in, out := &in.Databases, &out.Databases
		*out = make([]PostgresIdentifier, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresUserSpec.
func (in *PostgresUserSpec) DeepCopy() *PostgresUserSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresUserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoAzure) DeepCopyInto(out *RepoAzure) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoAzure.
func (in *RepoAzure) DeepCopy() *RepoAzure {
	if in == nil {
		return nil
	}
	out := new(RepoAzure)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoGCS) DeepCopyInto(out *RepoGCS) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoGCS.
func (in *RepoGCS) DeepCopy() *RepoGCS {
	if in == nil {
		return nil
	}
	out := new(RepoGCS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoHostStatus) DeepCopyInto(out *RepoHostStatus) {
	*out = *in
	out.TypeMeta = in.TypeMeta
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoHostStatus.
func (in *RepoHostStatus) DeepCopy() *RepoHostStatus {
	if in == nil {
		return nil
	}
	out := new(RepoHostStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoPVC) DeepCopyInto(out *RepoPVC) {
	*out = *in
	in.VolumeClaimSpec.DeepCopyInto(&out.VolumeClaimSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoPVC.
func (in *RepoPVC) DeepCopy() *RepoPVC {
	if in == nil {
		return nil
	}
	out := new(RepoPVC)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoS3) DeepCopyInto(out *RepoS3) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoS3.
func (in *RepoS3) DeepCopy() *RepoS3 {
	if in == nil {
		return nil
	}
	out := new(RepoS3)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoStatus) DeepCopyInto(out *RepoStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoStatus.
func (in *RepoStatus) DeepCopy() *RepoStatus {
	if in == nil {
		return nil
	}
	out := new(RepoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceSpec) DeepCopyInto(out *ServiceSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceSpec.
func (in *ServiceSpec) DeepCopy() *ServiceSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Sidecar) DeepCopyInto(out *Sidecar) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Sidecar.
func (in *Sidecar) DeepCopy() *Sidecar {
	if in == nil {
		return nil
	}
	out := new(Sidecar)
	in.DeepCopyInto(out)
	return out
}
