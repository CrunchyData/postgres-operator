// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CrunchyBridgeClusterSpec defines the desired state of CrunchyBridgeCluster
// to be managed by Crunchy Data Bridge
type CrunchyBridgeClusterSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Whether the cluster is high availability,
	// meaning that it has a secondary it can fail over to quickly
	// in case the primary becomes unavailable.
	// +kubebuilder:validation:Required
	IsHA bool `json:"isHa"`

	// Whether the cluster is protected. Protected clusters can't be destroyed until
	// their protected flag is removed
	// +optional
	IsProtected bool `json:"isProtected,omitempty"`

	// The name of the cluster
	// ---
	// According to Bridge API/GUI errors,
	// "Field name should be between 5 and 50 characters in length, containing only unicode characters, unicode numbers, hyphens, spaces, or underscores, and starting with a character", and ending with a character or number.
	// +kubebuilder:validation:MinLength=5
	// +kubebuilder:validation:MaxLength=50
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9\-_ ]*[A-Za-z0-9]$`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	ClusterName string `json:"clusterName"`

	// The ID of the cluster's plan. Determines instance, CPU, and memory.
	// +kubebuilder:validation:Required
	Plan string `json:"plan"`

	// The ID of the cluster's major Postgres version.
	// Currently Bridge offers 13-17
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=13
	// +kubebuilder:validation:Maximum=17
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	PostgresVersion int `json:"majorVersion"`

	// The cloud provider where the cluster is located.
	// Currently Bridge offers aws, azure, and gcp only
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum={aws,azure,gcp}
	// +kubebuilder:validation:XValidation:rule=`self == oldSelf`,message="immutable"
	Provider string `json:"provider"`

	// The provider region where the cluster is located.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule=`self == oldSelf`,message="immutable"
	Region string `json:"region"`

	// Roles for which to create Secrets that contain their credentials which
	// are retrieved from the Bridge API. An empty list creates no role secrets.
	// Removing a role from this list does NOT drop the role nor revoke their
	// access, but it will delete that role's secret from the kube cluster.
	// +listType=map
	// +listMapKey=name
	// +optional
	Roles []*CrunchyBridgeClusterRoleSpec `json:"roles,omitempty"`

	// The name of the secret containing the API key and team id
	// +kubebuilder:validation:Required
	Secret string `json:"secret,omitempty"`

	// The amount of storage available to the cluster in gigabytes.
	// The amount must be an integer, followed by Gi (gibibytes) or G (gigabytes) to match Kubernetes conventions.
	// If the amount is given in Gi, we round to the nearest G value.
	// The minimum value allowed by Bridge is 10 GB.
	// The maximum value allowed by Bridge is 65535 GB.
	// +kubebuilder:validation:Required
	Storage resource.Quantity `json:"storage"`
}

type CrunchyBridgeClusterRoleSpec struct {
	// Name of the role within Crunchy Bridge.
	// More info: https://docs.crunchybridge.com/concepts/users
	Name string `json:"name"`

	// The name of the Secret that will hold the role credentials.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Type=string
	SecretName string `json:"secretName"`
}

// CrunchyBridgeClusterStatus defines the observed state of CrunchyBridgeCluster
type CrunchyBridgeClusterStatus struct {
	// The name of the cluster in Bridge.
	// +optional
	ClusterName string `json:"name,omitempty"`

	// conditions represent the observations of postgres cluster's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// The Hostname of the postgres cluster in Bridge, provided by Bridge API and null until then.
	// +optional
	Host string `json:"host,omitempty"`

	// The ID of the postgres cluster in Bridge, provided by Bridge API and null until then.
	// +optional
	ID string `json:"id,omitempty"`

	// Whether the cluster is high availability, meaning that it has a secondary it can fail
	// over to quickly in case the primary becomes unavailable.
	// +optional
	IsHA *bool `json:"isHa"`

	// Whether the cluster is protected. Protected clusters can't be destroyed until
	// their protected flag is removed
	// +optional
	IsProtected *bool `json:"isProtected"`

	// The cluster's major Postgres version.
	// +optional
	MajorVersion int `json:"majorVersion"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// The cluster upgrade as represented by Bridge
	// +optional
	OngoingUpgrade []*UpgradeOperation `json:"ongoingUpgrade,omitempty"`

	// The ID of the cluster's plan. Determines instance, CPU, and memory.
	// +optional
	Plan string `json:"plan"`

	// Most recent, raw responses from Bridge API
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	Responses APIResponses `json:"responses"`

	// State of cluster in Bridge.
	// +optional
	State string `json:"state,omitempty"`

	// The amount of storage available to the cluster.
	// +optional
	Storage *resource.Quantity `json:"storage"`
}

type APIResponses struct {
	Cluster SchemalessObject `json:"cluster,omitempty"`
	Status  SchemalessObject `json:"status,omitempty"`
	Upgrade SchemalessObject `json:"upgrade,omitempty"`
}

type ClusterUpgrade struct {
	Operations []*UpgradeOperation `json:"operations,omitempty"`
}

type UpgradeOperation struct {
	Flavor       string `json:"flavor"`
	StartingFrom string `json:"starting_from"`
	State        string `json:"state"`
}

// TODO(crunchybridgecluster) Think through conditions
// CrunchyBridgeClusterStatus condition types.
const (
	ConditionUnknown   = ""
	ConditionUpgrading = "Upgrading"
	ConditionReady     = "Ready"
	ConditionDeleting  = "Deleting"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:csv:customresourcedefinitions:resources={{ConfigMap,v1},{Secret,v1},{Service,v1},{CronJob,v1beta1},{Deployment,v1},{Job,v1},{StatefulSet,v1},{PersistentVolumeClaim,v1}}

// CrunchyBridgeCluster is the Schema for the crunchybridgeclusters API
type CrunchyBridgeCluster struct {
	// ObjectMeta.Name is a DNS subdomain.
	// - https://docs.k8s.io/concepts/overview/working-with-objects/names/#dns-subdomain-names
	// - https://releases.k8s.io/v1.21.0/staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/validator.go#L60

	// In Bridge json, meta.name is "name"
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// NOTE(cbandy): Every CrunchyBridgeCluster needs a Spec, but it is optional here
	// so ObjectMeta can be managed independently.

	Spec   CrunchyBridgeClusterSpec   `json:"spec,omitempty"`
	Status CrunchyBridgeClusterStatus `json:"status,omitempty"`
}

// Default implements "sigs.k8s.io/controller-runtime/pkg/webhook.Defaulter" so
// a webhook can be registered for the type.
// - https://book.kubebuilder.io/reference/webhook-overview.html
func (c *CrunchyBridgeCluster) Default() {
	if len(c.APIVersion) == 0 {
		c.APIVersion = GroupVersion.String()
	}
	if len(c.Kind) == 0 {
		c.Kind = "CrunchyBridgeCluster"
	}
}

// +kubebuilder:object:root=true

// CrunchyBridgeClusterList contains a list of CrunchyBridgeCluster
type CrunchyBridgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CrunchyBridgeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CrunchyBridgeCluster{}, &CrunchyBridgeClusterList{})
}

func NewCrunchyBridgeCluster() *CrunchyBridgeCluster {
	cluster := &CrunchyBridgeCluster{}
	cluster.SetGroupVersionKind(GroupVersion.WithKind("CrunchyBridgeCluster"))
	return cluster
}
