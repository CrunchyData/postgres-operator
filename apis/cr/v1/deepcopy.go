package v1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Pgbackup) DeepCopyInto(out *Pgbackup) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = PgbackupSpec{
		Namespace:        in.Spec.Namespace,
		Name:             in.Spec.Name,
		StorageSpec:      in.Spec.StorageSpec,
		CCPImageTag:      in.Spec.CCPImageTag,
		BackupHost:       in.Spec.BackupHost,
		BackupUserSecret: in.Spec.BackupUserSecret,
		BackupPort:       in.Spec.BackupPort,
		BackupOpts:       in.Spec.BackupOpts,
		BackupStatus:     in.Spec.BackupStatus,
		BackupPVC:        in.Spec.BackupPVC,
	}
	out.Status = in.Status
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Pgbackup) DeepCopyObject() runtime.Object {
	out := Pgbackup{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PgbackupList) DeepCopyObject() runtime.Object {
	out := PgbackupList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Pgbackup, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Pgupgrade) DeepCopyInto(out *Pgupgrade) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Status = in.Status

	out.Spec = PgupgradeSpec{
		Namespace:       in.Spec.Namespace,
		Name:            in.Spec.Name,
		ResourceType:    in.Spec.ResourceType,
		UpgradeType:     in.Spec.UpgradeType,
		UpgradeStatus:   in.Spec.UpgradeStatus,
		StorageSpec:     in.Spec.StorageSpec,
		CCPImage:        in.Spec.CCPImage,
		CCPImageTag:     in.Spec.CCPImageTag,
		OldDatabaseName: in.Spec.OldDatabaseName,
		NewDatabaseName: in.Spec.NewDatabaseName,
		OldVersion:      in.Spec.OldVersion,
		NewVersion:      in.Spec.NewVersion,
		OldPVCName:      in.Spec.OldPVCName,
		NewPVCName:      in.Spec.NewPVCName,
		BackupPVCName:   in.Spec.BackupPVCName,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Pgupgrade) DeepCopyObject() runtime.Object {
	out := Pgupgrade{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PgupgradeList) DeepCopyObject() runtime.Object {
	out := PgupgradeList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Pgupgrade, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Pgreplica) DeepCopyInto(out *Pgreplica) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Status = in.Status
	out.Spec = PgreplicaSpec{
		Namespace:          in.Spec.Namespace,
		Name:               in.Spec.Name,
		ClusterName:        in.Spec.ClusterName,
		ReplicaStorage:     in.Spec.ReplicaStorage,
		ContainerResources: in.Spec.ContainerResources,
		Status:             in.Spec.Status,
		UserLabels:         in.Spec.UserLabels,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Pgreplica) DeepCopyObject() runtime.Object {
	out := Pgreplica{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PgreplicaList) DeepCopyObject() runtime.Object {
	out := PgreplicaList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Pgreplica, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Pgcluster) DeepCopyInto(out *Pgcluster) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Status = in.Status
	out.Spec = PgclusterSpec{
		Namespace:          in.Spec.Namespace,
		Name:               in.Spec.Name,
		ClusterName:        in.Spec.ClusterName,
		Policies:           in.Spec.Policies,
		CCPImage:           in.Spec.CCPImage,
		CCPImageTag:        in.Spec.CCPImageTag,
		Port:               in.Spec.Port,
		NodeName:           in.Spec.NodeName,
		PrimaryStorage:     in.Spec.PrimaryStorage,
		ReplicaStorage:     in.Spec.ReplicaStorage,
		BackrestStorage:    in.Spec.BackrestStorage,
		ContainerResources: in.Spec.ContainerResources,
		PrimaryHost:        in.Spec.PrimaryHost,
		User:               in.Spec.User,
		Database:           in.Spec.Database,
		Replicas:           in.Spec.Replicas,
		Strategy:           in.Spec.Strategy,
		SecretFrom:         in.Spec.SecretFrom,
		BackupPVCName:      in.Spec.BackupPVCName,
		BackupPath:         in.Spec.BackupPath,
		UserSecretName:     in.Spec.UserSecretName,
		RootSecretName:     in.Spec.RootSecretName,
		PrimarySecretName:  in.Spec.PrimarySecretName,
		Status:             in.Spec.Status,
		PswLastUpdate:      in.Spec.PswLastUpdate,
		CustomConfig:       in.Spec.CustomConfig,
		UserLabels:         in.Spec.UserLabels,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Pgcluster) DeepCopyObject() runtime.Object {
	out := Pgcluster{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PgclusterList) DeepCopyObject() runtime.Object {
	out := PgclusterList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Pgcluster, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Pgpolicy) DeepCopyInto(out *Pgpolicy) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = PgpolicySpec{
		Namespace: in.Spec.Namespace,
		Name:      in.Spec.Name,
		URL:       in.Spec.URL,
		SQL:       in.Spec.SQL,
		Status:    in.Spec.Status,
	}
	out.Status = in.Status
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Pgpolicy) DeepCopyObject() runtime.Object {
	out := Pgpolicy{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PgpolicyList) DeepCopyObject() runtime.Object {
	out := PgpolicyList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Pgpolicy, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Pgtask) DeepCopyInto(out *Pgtask) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = PgtaskSpec{
		Namespace:   in.Spec.Namespace,
		Name:        in.Spec.Name,
		StorageSpec: in.Spec.StorageSpec,
		TaskType:    in.Spec.TaskType,
		Status:      in.Spec.Status,
		Parameters:  in.Spec.Parameters,
	}
	out.Status = in.Status
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Pgtask) DeepCopyObject() runtime.Object {
	out := Pgtask{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *PgtaskList) DeepCopyObject() runtime.Object {
	out := PgtaskList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Pgtask, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
