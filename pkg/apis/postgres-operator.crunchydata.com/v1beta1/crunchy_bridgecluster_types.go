/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	IsHA bool `json:"is_ha"`

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
	Plan string `json:"plan_id"`

	// The ID of the cluster's major Postgres version.
	// Currently Bridge offers 13-16
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=13
	// +kubebuilder:validation:Maximum=16
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	PostgresVersion int `json:"postgres_version_id"`

	// The cloud provider where the cluster is located.
	// Currently Bridge offers aws, azure, and gcp only
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum={aws,azure,gcp}
	Provider string `json:"provider_id"`

	// The provider region where the cluster is located.
	// +kubebuilder:validation:Required
	Region string `json:"region_id"`

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

// CrunchyBridgeClusterStatus defines the observed state of CrunchyBridgeCluster
type CrunchyBridgeClusterStatus struct {
	// The ID of the postgrescluster in Bridge, provided by Bridge API and null until then.
	// +optional
	ID string `json:"id,omitempty"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions represent the observations of postgrescluster's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// The cluster as represented by Bridge
	// +optional
	Cluster *ClusterDetails `json:"clusterResponse,omitempty"`

	// The cluster upgrade as represented by Bridge
	// +optional
	ClusterUpgrade *ClusterUpgrade `json:"clusterUpgradeResponse,omitempty"`
}

// Right now used for cluster create requests and cluster get responses
type ClusterDetails struct {
	ID              string             `json:"id,omitempty"`
	IsHA            bool               `json:"is_ha,omitempty"`
	Name            string             `json:"name,omitempty"`
	Plan            string             `json:"plan_id,omitempty"`
	MajorVersion    int                `json:"major_version,omitempty"`
	PostgresVersion intstr.IntOrString `json:"postgres_version_id,omitempty"`
	Provider        string             `json:"provider_id,omitempty"`
	Region          string             `json:"region_id,omitempty"`
	Storage         int64              `json:"storage,omitempty"`
	Team            string             `json:"team_id,omitempty"`
	State           string             `json:"state,omitempty"`
	// TODO(crunchybridgecluster): add other fields, DiskUsage, Host, IsProtected, IsSuspended, CPU, Memory, etc.
}

type ClusterUpgrade struct {
	Operations []*Operation `json:"operations,omitempty"`
}

type Operation struct {
	Flavor       string `json:"flavor"`
	StartingFrom string `json:"starting_from"`
	State        string `json:"state"`
}

// TODO(crunchybridgecluster) Think through conditions
// CrunchyBridgeClusterStatus condition types.
const (
	ConditionUnknown  = ""
	ConditionPending  = "Pending"
	ConditionCreating = "Creating"
	ConditionUpdating = "Updating"
	ConditionReady    = "Ready"
	ConditionDeleting = "Deleting"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:csv:customresourcedefinitions:resources={{ConfigMap,v1},{Secret,v1},{Service,v1},{CronJob,v1beta1},{Deployment,v1},{Job,v1},{StatefulSet,v1},{PersistentVolumeClaim,v1}}

// CrunchyBridgeCluster is the Schema for the crunchybridgeclusters API
// This Custom Resource requires enabling CrunchyBridgeClusters feature gate
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
