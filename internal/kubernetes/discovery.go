// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"errors"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

type Version = version.Info

// DiscoveryRunner implements [APIs] by reading from a Kubernetes API client.
// Its methods are safe to call concurrently.
type DiscoveryRunner struct {
	// NOTE(tracing): The methods of [discovery.DiscoveryClient] do not take
	// a Context so their API calls won't have a parent span.
	// - https://issue.k8s.io/126379
	Client interface {
		ServerGroups() (*metav1.APIGroupList, error)
		ServerResourcesForGroupVersion(string) (*metav1.APIResourceList, error)
		ServerVersion() (*version.Info, error)
	}

	refresh time.Duration

	// relevant is the list of APIs to examine during Read.
	// Has, HasAll, and HasAny return false when this is empty.
	relevant []API

	have struct {
		sync.RWMutex
		APISet
		Version
	}
}

// NewDiscoveryRunner creates a [DiscoveryRunner] that periodically reads from
// the Kubernetes at config.
func NewDiscoveryRunner(config *rest.Config) (*DiscoveryRunner, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(config)

	runner := &DiscoveryRunner{
		Client:  dc,
		refresh: 10 * time.Minute,
		relevant: []API{
			// https://cert-manager.io/docs/usage/certificate
			// https://cert-manager.io/docs/trust/trust-manager
			{Group: "cert-manager.io", Kind: "Certificate"},
			{Group: "trust.cert-manager.io", Kind: "Bundle"},

			// https://gateway-api.sigs.k8s.io/api-types/referencegrant
			// https://kep.k8s.io/3766
			{Group: "gateway.networking.k8s.io", Kind: "ReferenceGrant"},

			// https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html
			{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},

			// https://docs.k8s.io/concepts/storage/volume-snapshots
			{Group: "snapshot.storage.k8s.io", Kind: "VolumeSnapshot"},
		},
	}

	return runner, err
}

// Has returns true when api is available in Kuberentes.
func (r *DiscoveryRunner) Has(api API) bool { return r.HasAny(api) }

// HasAll returns true when every api is available in Kubernetes.
func (r *DiscoveryRunner) HasAll(api ...API) bool {
	r.have.RLock()
	defer r.have.RUnlock()
	return r.have.HasAll(api...)
}

// HasAny returns true when at least one api is available in Kubernetes.
func (r *DiscoveryRunner) HasAny(api ...API) bool {
	r.have.RLock()
	defer r.have.RUnlock()
	return r.have.HasAny(api...)
}

// IsOpenShift returns true if this Kubernetes might be OpenShift. The result
// may not be accurate.
func (r *DiscoveryRunner) IsOpenShift() bool {
	return r.Has(API{Group: "security.openshift.io", Kind: "SecurityContextConstraints"})
}

// NeedLeaderElection returns false so that r runs on any [manager.Manager],
// regardless of which is elected leader in the Kubernetes namespace.
func (r *DiscoveryRunner) NeedLeaderElection() bool { return false }

// Read fetches available APIs from Kubernetes.
func (r *DiscoveryRunner) Read(ctx context.Context) error {
	return errors.Join(r.readAPIs(ctx), r.readVersion())
}

func (r *DiscoveryRunner) readAPIs(ctx context.Context) error {
	// Build an index of the APIs we want to know about.
	wantAPIs := make(map[string]map[string]sets.Set[string])
	for _, want := range r.relevant {
		if wantAPIs[want.Group] == nil {
			wantAPIs[want.Group] = make(map[string]sets.Set[string])
		}
		if wantAPIs[want.Group][want.Version] == nil {
			wantAPIs[want.Group][want.Version] = sets.New[string]()
		}
		if want.Kind != "" {
			wantAPIs[want.Group][want.Version].Insert(want.Kind)
		}
	}

	// Fetch Groups and Versions from Kubernetes.
	groups, err := r.Client.ServerGroups()
	if err != nil {
		return err
	}

	// Build an index of the Groups and GVs available in Kubernetes;
	// add GK and GVK for resources that we want to know about.
	haveAPIs := make(APISet)
	for _, apiG := range groups.Groups {
		haveG := apiG.Name
		haveAPIs.Insert(API{Group: haveG})

		for _, apiGV := range apiG.Versions {
			haveV := apiGV.Version
			haveAPIs.Insert(API{Group: haveG, Version: haveV})

			// Only fetch Resources when there are Kinds we want to know about.
			if wantAPIs[haveG][""].Len() == 0 && wantAPIs[haveG][haveV].Len() == 0 {
				continue
			}

			resources, err := r.Client.ServerResourcesForGroupVersion(apiGV.GroupVersion)
			if err != nil {
				return err
			}

			for _, apiR := range resources.APIResources {
				haveK := apiR.Kind
				haveAPIs.Insert(
					API{Group: haveG, Kind: haveK},
					API{Group: haveG, Kind: haveK, Version: haveV},
				)
			}
		}
	}

	r.have.Lock()
	r.have.APISet = haveAPIs
	r.have.Unlock()

	r.have.RLock()
	defer r.have.RUnlock()
	logging.FromContext(ctx).V(1).Info("Found APIs", "index_size", r.have.Len())

	return nil
}

func (r *DiscoveryRunner) readVersion() error {
	info, err := r.Client.ServerVersion()

	if info != nil && err == nil {
		r.have.Lock()
		r.have.Version = *info
		r.have.Unlock()
	}

	return err
}

// Start periodically reads the Kuberentes API. It blocks until ctx is cancelled.
func (r *DiscoveryRunner) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.refresh)
	defer ticker.Stop()

	log := logging.FromContext(ctx).WithValues("controller", "kubernetes")
	ctx = logging.NewContext(ctx, log)

	for {
		select {
		case <-ticker.C:
			if err := r.Read(ctx); err != nil {
				log.Error(err, "Unable to detect Kubernetes APIs")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Version returns the detected version of Kubernetes.
func (r *DiscoveryRunner) Version() Version {
	r.have.RLock()
	defer r.have.RUnlock()
	return r.have.Version
}

// IsOpenShift returns true if the detected Kubernetes might be OpenShift.
// The result may not be accurate. When possible, use another technique to
// detect specific behavior. Use [Has] to check for specific APIs.
func IsOpenShift(ctx context.Context) bool {
	if i, ok := ctx.Value(apiContextKey{}).(interface{ IsOpenShift() bool }); ok {
		return i.IsOpenShift()
	}
	return false
}

// VersionString returns a textual representation of the detected Kubernetes
// version, if any.
func VersionString(ctx context.Context) string {
	if i, ok := ctx.Value(apiContextKey{}).(interface{ Version() Version }); ok {
		return i.Version().String()
	}
	return ""
}
