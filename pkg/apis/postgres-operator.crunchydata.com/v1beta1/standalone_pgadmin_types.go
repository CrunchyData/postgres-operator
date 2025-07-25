// Copyright 2023 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PGAdminConfiguration represents pgAdmin configuration files.
type StandalonePGAdminConfiguration struct {
	// Files allows the user to mount projected volumes into the pgAdmin
	// container so that files can be referenced by pgAdmin as needed.
	// +optional
	Files []corev1.VolumeProjection `json:"files,omitempty"`

	// A Secret containing the value for the CONFIG_DATABASE_URI setting.
	// More info: https://www.pgadmin.org/docs/pgadmin4/latest/external_database.html
	// +optional
	ConfigDatabaseURI *OptionalSecretKeyRef `json:"configDatabaseURI,omitempty"`

	// Settings for the Gunicorn server.
	// More info: https://docs.gunicorn.org/en/latest/settings.html
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	Gunicorn SchemalessObject `json:"gunicorn,omitempty"`

	// A Secret containing the value for the LDAP_BIND_PASSWORD setting.
	// More info: https://www.pgadmin.org/docs/pgadmin4/latest/ldap.html
	// +optional
	LDAPBindPassword *OptionalSecretKeyRef `json:"ldapBindPassword,omitempty"`

	// Settings for the pgAdmin server process. Keys should be uppercase and
	// values must be constants.
	// More info: https://www.pgadmin.org/docs/pgadmin4/latest/config_py.html
	// ---
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	//
	// +mapType=granular
	// +optional
	Settings SchemalessObject `json:"settings,omitempty"`

	// Secrets for the `OAUTH2_CONFIG` setting. If there are `OAUTH2_CONFIG` values
	// in the settings field, they will be combined with the values loaded here.
	// More info: https://www.pgadmin.org/docs/pgadmin4/latest/oauth2.html
	// ---
	// The controller expects this number to be no more than two digits.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	//
	// +listType=map
	// +listMapKey=name
	// +optional
	OAuthConfigurations []PGAdminOAuthConfig `json:"oauthConfigurations,omitempty"`
}

// +structType=atomic
type PGAdminOAuthConfig struct {
	// The OAUTH2_NAME of this configuration.
	// ---
	// This goes into a filename, so let's keep it short and simple.
	// The Secret is allowed to contain OAUTH2_NAME and deviate from this.
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9]+$`
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=20
	// +required
	Name string `json:"name"`

	// A Secret containing the settings of one OAuth2 provider as a JSON object.
	// ---
	// +required
	Secret SecretKeyRef `json:"secret"`
}

// PGAdminSpec defines the desired state of PGAdmin
type PGAdminSpec struct {

	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Configuration settings for the pgAdmin process. Changes to any of these
	// values will be loaded without validation. Be careful, as
	// you may put pgAdmin into an unusable state.
	// +optional
	Config StandalonePGAdminConfiguration `json:"config,omitzero"`

	// Defines a PersistentVolumeClaim for pgAdmin data.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes
	// ---
	// +required
	DataVolumeClaimSpec VolumeClaimSpec `json:"dataVolumeClaimSpec"`

	// The image name to use for pgAdmin instance.
	// +optional
	Image *string `json:"image,omitempty"`

	// ImagePullPolicy is used to determine when Kubernetes will attempt to
	// pull (download) container images.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:Type=string
	//
	// +kubebuilder:validation:Enum={Always,Never,IfNotPresent}
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// The image pull secrets used to pull from a private registry.
	// Changing this value causes all running PGAdmin pods to restart.
	// https://k8s.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Configuration for the OpenTelemetry collector container used to collect
	// logs and metrics.
	// +optional
	Instrumentation *InstrumentationSpec `json:"instrumentation,omitempty"`

	// Resource requirements for the PGAdmin container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitzero"`

	// Scheduling constraints of the PGAdmin pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Priority class name for the PGAdmin pod. Changing this
	// value causes PGAdmin pod to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Tolerations of the PGAdmin pod.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// ServerGroups for importing PostgresClusters to pgAdmin.
	// To create a pgAdmin with no selectors, leave this field empty.
	// A pgAdmin created with no `ServerGroups` will not automatically
	// add any servers through discovery. PostgresClusters can still be
	// added manually.
	// +optional
	ServerGroups []ServerGroup `json:"serverGroups"`

	// pgAdmin users that are managed via the PGAdmin spec. Users can still
	// be added via the pgAdmin GUI, but those users will not show up here.
	// +listType=map
	// +listMapKey=username
	// +optional
	Users []PGAdminUser `json:"users,omitempty"`

	// ServiceName will be used as the name of a ClusterIP service pointing
	// to the pgAdmin pod and port. If the service already exists, PGO will
	// update the service. For more information about services reference
	// the Kubernetes and CrunchyData documentation.
	// https://kubernetes.io/docs/concepts/services-networking/service/
	// +optional
	ServiceName string `json:"serviceName,omitempty"`
}

// +kubebuilder:validation:XValidation:rule=`[has(self.postgresClusterName),has(self.postgresClusterSelector)].exists_one(x,x)`,message=`exactly one of "postgresClusterName" or "postgresClusterSelector" is required`
type ServerGroup struct {
	// The name for the ServerGroup in pgAdmin.
	// Must be unique in the pgAdmin's ServerGroups since it becomes the ServerGroup name in pgAdmin.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// PostgresClusterName selects one cluster to add to pgAdmin by name.
	// +kubebuilder:validation:Optional
	PostgresClusterName string `json:"postgresClusterName,omitempty"`

	// PostgresClusterSelector selects clusters to dynamically add to pgAdmin by matching labels.
	// An empty selector like `{}` will select ALL clusters in the namespace.
	// +kubebuilder:validation:Optional
	PostgresClusterSelector metav1.LabelSelector `json:"postgresClusterSelector,omitzero"`
}

type PGAdminUser struct {
	// A reference to the secret that holds the user's password.
	// +kubebuilder:validation:Required
	PasswordRef *corev1.SecretKeySelector `json:"passwordRef"`

	// Role determines whether the user has admin privileges or not.
	// Defaults to User. Valid options are Administrator and User.
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	//
	// +kubebuilder:validation:Enum={Administrator,User}
	// +optional
	Role string `json:"role,omitempty"`

	// The username for User in pgAdmin.
	// Must be unique in the pgAdmin's users list.
	// +kubebuilder:validation:Required
	Username string `json:"username"`
}

// PGAdminStatus defines the observed state of PGAdmin
type PGAdminStatus struct {

	// conditions represent the observations of pgAdmin's current state.
	// Known .status.conditions.type is: "PersistentVolumeResizing"
	// +optional
	// +listType=map
	// +listMapKey=type
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ImageSHA represents the image SHA for the container running pgAdmin.
	// +optional
	ImageSHA string `json:"imageSHA,omitempty"`

	// MajorVersion represents the major version of the running pgAdmin.
	// +optional
	MajorVersion int `json:"majorVersion,omitempty"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+versionName=v1beta1

// PGAdmin is the Schema for the PGAdmin API
type PGAdmin struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +optional
	Spec PGAdminSpec `json:"spec,omitzero"`
	// +optional
	Status PGAdminStatus `json:"status,omitzero"`
}

// Default implements "sigs.k8s.io/controller-runtime/pkg/webhook.Defaulter" so
// a webhook can be registered for the type.
// - https://book.kubebuilder.io/reference/webhook-overview.html
func (p *PGAdmin) Default() {
	if len(p.APIVersion) == 0 {
		p.APIVersion = GroupVersion.String()
	}
	if len(p.Kind) == 0 {
		p.Kind = "PGAdmin"
	}
}

func NewPGAdmin() *PGAdmin {
	p := &PGAdmin{}
	p.Default()
	return p
}

//+kubebuilder:object:root=true

// PGAdminList contains a list of PGAdmin
type PGAdminList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PGAdmin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PGAdmin{}, &PGAdminList{})
}
