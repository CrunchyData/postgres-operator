package v1

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgclusterResourcePlural ..
const PgclusterResourcePlural = "pgclusters"

// Pgcluster is the CRD that defines a Crunchy PG Cluster
//
// swagger:ignore Pgcluster
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Pgcluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PgclusterSpec   `json:"spec"`
	Status            PgclusterStatus `json:"status,omitempty"`
}

// PgclusterSpec is the CRD that defines a Crunchy PG Cluster Spec
// swagger:ignore
type PgclusterSpec struct {
	Namespace       string        `json:"namespace"`
	Name            string        `json:"name"`
	ClusterName     string        `json:"clustername"`
	Policies        string        `json:"policies"`
	CCPImage        string        `json:"ccpimage"`
	CCPImageTag     string        `json:"ccpimagetag"`
	CCPImagePrefix  string        `json:"ccpimageprefix"`
	PGOImagePrefix  string        `json:"pgoimageprefix"`
	Port            string        `json:"port"`
	PGBadgerPort    string        `json:"pgbadgerport"`
	ExporterPort    string        `json:"exporterport"`
	PrimaryStorage  PgStorageSpec `json:primarystorage`
	ArchiveStorage  PgStorageSpec `json:archivestorage`
	ReplicaStorage  PgStorageSpec `json:replicastorage`
	BackrestStorage PgStorageSpec `json:backreststorage`
	// Resources behaves just like the "Requests" section of a Kubernetes
	// container definition. You can set individual items such as "cpu" and
	// "memory", e.g. "{ cpu: "0.5", memory: "2Gi" }"
	//
	// For memory requests, we only set the "Request" portion of the Container
	// resource definition, but if we do allow for the "Limit" portion to be set,
	// we would keep it unified to get a "Guaranteed" QoS.
	//
	// We don't set the Limit you say? Yes: we want to avoid the OOM killer coming
	// for the PostgreSQL process or any of their backends per lots of guidance
	// from the PostgreSQL documentation. Based on Kubernetes' behavior with
	// limits, the best thing is to not set them. However, if they ever do set,
	// we'll ensure that we get the Guaranteed QoS to help avoid OOM risks.
	//
	// Guaranteed QoS prevents a backend from being first in line to be killed if
	// the *Node* has memory pressure, but if there is, say
	// a runaway client backend that causes the *Pod* to exceed its memory
	// limit, a backend can still be killed by the OOM killer, which is not
	// great.
	//
	// As such, given the choice, the preference is for the Pod to be evicted
	// and have a failover event, vs. having an individual client backend killed
	// and causing potential "bad things."
	//
	// For more info on PostgreSQL and Kubernetes memory management, see:
	//
	// https://www.postgresql.org/docs/current/kernel-resources.html#LINUX-MEMORY-OVERCOMMIT
	// https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#how-pods-with-resource-limits-are-run
	//
	// Now, for CPU, we set both the Request and the Limit, based on how
	// Kubernetes interacts with these parameters
	Resources v1.ResourceList `json:"resources"`
	// BackrestResources, if specified, contains the container request resources
	// for the pgBackRest Deployment for this PostgreSQL cluster
	BackrestResources v1.ResourceList `json:"backrestResources"`
	// PgBouncer contains all of the settings to properly maintain a pgBouncer
	// implementation
	PgBouncer          PgBouncerSpec            `json:"pgBouncer"`
	PrimaryHost        string                   `json:"primaryhost"`
	User               string                   `json:"user"`
	Database           string                   `json:"database"`
	Replicas           string                   `json:"replicas"`
	SecretFrom         string                   `json:"secretfrom"`
	UserSecretName     string                   `json:"usersecretname"`
	RootSecretName     string                   `json:"rootsecretname"`
	PrimarySecretName  string                   `json:"primarysecretname"`
	CollectSecretName  string                   `json:"collectSecretName"`
	Status             string                   `json:"status"`
	CustomConfig       string                   `json:"customconfig"`
	UserLabels         map[string]string        `json:"userlabels"`
	PodAntiAffinity    PodAntiAffinitySpec      `json:"podPodAntiAffinity"`
	SyncReplication    *bool                    `json:"syncReplication"`
	BackrestS3Bucket   string                   `json:"backrestS3Bucket"`
	BackrestS3Region   string                   `json:"backrestS3Region"`
	BackrestS3Endpoint string                   `json:"backrestS3Endpoint"`
	BackrestRepoPath   string                   `json:"backrestRepoPath"`
	TablespaceMounts   map[string]PgStorageSpec `json:"tablespaceMounts"`
	TLS                TLSSpec                  `json:"tls"`
	TLSOnly            bool                     `json:"tlsOnly"`
	Standby            bool                     `json:"standby"`
	Shutdown           bool                     `json:"shutdown"`
}

// PgclusterList is the CRD that defines a Crunchy PG Cluster List
// swagger:ignore
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgclusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgcluster `json:"items"`
}

// PgclusterStatus is the CRD that defines PG Cluster Status
// swagger:ignore
type PgclusterStatus struct {
	State   PgclusterState `json:"state,omitempty"`
	Message string         `json:"message,omitempty"`
}

// PgclusterState is the crd that defines PG Cluster Stage
// swagger:ignore
type PgclusterState string

// PodAntiAffinityDeployment distinguishes between the different types of
// Deployments that can leverage PodAntiAffinity
type PodAntiAffinityDeployment int

// PodAntiAffinityType defines the different types of type of anti-affinity rules applied to pg
// clusters when utilizing the default pod anti-affinity rules provided by the PostgreSQL Operator,
// which are enabled for a new pg cluster by default.  Valid Values include "required" for
// requiredDuringSchedulingIgnoredDuringExecution anti-affinity, "preferred" for
// preferredDuringSchedulingIgnoredDuringExecution anti-affinity, and "disabled" to disable the
// default pod anti-affinity rules for the pg cluster all together.
type PodAntiAffinityType string

// PodAntiAffinitySpec provides multiple configurations for how pod
// anti-affinity can be set.
// - "Default" is the default rule that applies to all Pods that are a part of
//		the PostgreSQL cluster
// - "PgBackrest" applies just to the pgBackRest repository Pods in said
//		Deployment
// - "PgBouncer" applies to just pgBouncer Pods in said Deployment
// swaggier:ignore
type PodAntiAffinitySpec struct {
	Default    PodAntiAffinityType `json:"default"`
	PgBackRest PodAntiAffinityType `json:"pgBackRest"`
	PgBouncer  PodAntiAffinityType `json:"pgBouncer"`
}

// PgBouncerSpec is a struct that is used within the Cluster specification that
// provides the attributes for managing a PgBouncer implementation, including:
// - is it enabled?
// - what resources it should consume
// - TODO: how many replicas it has?
type PgBouncerSpec struct {
	// Enabled is true if a pgBouncer Deployment should be deployed with the
	// PostgreSQL cluster
	Enabled bool `json:"enabled"`
	// Resources, if specified, contains the container request resources
	// for any pgBouncer Deployments that are part of a PostgreSQL cluster
	Resources v1.ResourceList `json:"resources"`
}

// TLSSpec contains the information to set up a TLS-enabled PostgreSQL cluster
type TLSSpec struct {
	// CASecret contains the name of the secret to use as the trusted CA for the
	// TLSSecret
	// This is our own format and should contain at least one key: "ca.crt"
	// It can also contain a key "ca.crl" which is the certificate revocation list
	CASecret string `json:"caSecret"`
	// TLSSecret contains the name of the secret to use that contains the TLS
	// keypair for the PostgreSQL server
	// This follows the Kubernetes secret format ("kubernetes.io/tls") which has
	// two keys: tls.crt and tls.key
	TLSSecret string `json:"tlsSecret"`
}

// IsTLSEnabled returns true if the cluster is TLS enabled, i.e. both the TLS
// secret name and the CA secret name are available
func (t TLSSpec) IsTLSEnabled() bool {
	return (t.TLSSecret != "" && t.CASecret != "")
}

const (
	// PgclusterStateCreated ...
	PgclusterStateCreated PgclusterState = "pgcluster Created"
	// PgclusterStateProcessed ...
	PgclusterStateProcessed PgclusterState = "pgcluster Processed"
	// PgclusterStateInitialized ...
	PgclusterStateInitialized PgclusterState = "pgcluster Initialized"
	// PgclusterStateRestore ...
	PgclusterStateRestore PgclusterState = "pgcluster Restoring"
	// PgclusterStateShutdown indicates that the cluster has been shut down (i.e. the primary)
	// deployment has been scaled to 0
	PgclusterStateShutdown PgclusterState = "pgcluster Shutdown"

	// PodAntiAffinityRequired results in requiredDuringSchedulingIgnoredDuringExecution for any
	// default pod anti-affinity rules applied to pg custers
	PodAntiAffinityRequired PodAntiAffinityType = "required"

	// PodAntiAffinityPreffered results in preferredDuringSchedulingIgnoredDuringExecution for any
	// default pod anti-affinity rules applied to pg custers
	PodAntiAffinityPreffered PodAntiAffinityType = "preferred"

	// PodAntiAffinityDisabled disables any default pod anti-affinity rules applied to pg custers
	PodAntiAffinityDisabled PodAntiAffinityType = "disabled"
)

// The list of different types of PodAntiAffinityDeployments
const (
	PodAntiAffinityDeploymentDefault PodAntiAffinityDeployment = iota
	PodAntiAffinityDeploymentPgBackRest
	PodAntiAffinityDeploymentPgBouncer
)

// ValidatePodAntiAffinityType is responsible for validating whether or not the type of pod
// anti-affinity specified is valid
func (p PodAntiAffinityType) Validate() error {
	switch p {
	case
		PodAntiAffinityRequired,
		PodAntiAffinityPreffered,
		PodAntiAffinityDisabled,
		"":
		return nil
	}
	return fmt.Errorf("Invalid pod anti-affinity type.  Valid values are '%s', '%s' or '%s'",
		PodAntiAffinityRequired, PodAntiAffinityPreffered, PodAntiAffinityDisabled)
}
