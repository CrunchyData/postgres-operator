// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"context"
	"errors"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

// API is a combination of Group, Version, and Kind that can be used to check
// what is available in the Kubernetes API. There are four ways to populate it:
//  1. Group without Version nor Kind means any resource in that Group.
//  2. Group with Version but no Kind means any resource in that GV.
//  3. Group with Kind but no Version means that Kind in any Version of the Group.
//  4. Group with Version and Kind means that exact GVK.
type API = schema.GroupVersionKind

type APIs interface {
	Has(API) bool
	HasAll(...API) bool
	HasOne(...API) bool
}

// APISet implements [APIs] using empty struct for minimal memory consumption.
type APISet map[API]struct{}

func NewAPISet(api ...API) APISet {
	s := make(APISet)

	for i := range api {
		s[api[i]] = struct{}{}
		s[API{Group: api[i].Group}] = struct{}{}
		s[API{Group: api[i].Group, Version: api[i].Version}] = struct{}{}
		s[API{Group: api[i].Group, Kind: api[i].Kind}] = struct{}{}
	}

	return s
}

// Has returns true when api is available in s.
func (s APISet) Has(api API) bool { return s.HasOne(api) }

// HasAll returns true when every api is available in s.
func (s APISet) HasAll(api ...API) bool {
	for i := range api {
		if _, present := s[api[i]]; !present {
			return false
		}
	}
	return true
}

// HasOne returns true when at least one api is available in s.
func (s APISet) HasOne(api ...API) bool {
	for i := range api {
		if _, present := s[api[i]]; present {
			return true
		}
	}
	return false
}

type APIDiscoveryRunner struct {
	Client interface {
		ServerGroups() (*metav1.APIGroupList, error)
		ServerResourcesForGroupVersion(string) (*metav1.APIResourceList, error)
	}

	refresh time.Duration

	want []API
	have struct {
		sync.RWMutex
		APISet
	}
}

// NewAPIDiscoveryRunner creates an [APIDiscoveryRunner] that periodically reads
// what APIs are available in the Kubernetes at config.
func NewAPIDiscoveryRunner(config *rest.Config) (*APIDiscoveryRunner, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(config)

	runner := &APIDiscoveryRunner{
		Client:  dc,
		refresh: 10 * time.Minute,
		want: []API{
			{Group: "cert-manager.io", Kind: "Certificate"},
			{Group: "gateway.networking.k8s.io", Kind: "ReferenceGrant"},
			{Group: "security.openshift.io", Kind: "SecurityContextConstraints"},
			{Group: "snapshot.storage.k8s.io", Kind: "VolumeSnapshot"},
			{Group: "trust.cert-manager.io", Kind: "Bundle"},
		},
	}

	return runner, err
}

// NeedLeaderElection returns false so that r runs on any [manager.Manager],
// regardless of which is elected leader in the Kubernetes namespace.
func (r *APIDiscoveryRunner) NeedLeaderElection() bool { return false }

// Read fetches available APIs from Kubernetes.
func (r *APIDiscoveryRunner) Read() error {

	// Build an index of the APIs we want to know about.
	wantAPIs := make(map[string]map[string]sets.Set[string])
	for _, want := range r.want {
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

	// Build an index of the Groups, GVs, GKs, and GVKs available in Kuberentes
	// that we want to know about.
	haveWantedAPIs := make(map[API]struct{})
	for _, apiG := range groups.Groups {
		var haveG string = apiG.Name
		haveWantedAPIs[API{Group: haveG}] = struct{}{}

		for _, apiGV := range apiG.Versions {
			var haveV string = apiGV.Version
			haveWantedAPIs[API{Group: haveG, Version: haveV}] = struct{}{}

			// Only fetch Resources when there are Kinds we want to know about.
			if wantAPIs[haveG][""].Len() == 0 && wantAPIs[haveG][haveV].Len() == 0 {
				continue
			}

			resources, err := r.Client.ServerResourcesForGroupVersion(apiGV.GroupVersion)
			if err != nil {
				return err
			}

			for _, apiR := range resources.APIResources {
				var haveK string = apiR.Kind
				haveWantedAPIs[API{Group: haveG, Kind: haveK}] = struct{}{}
				haveWantedAPIs[API{Group: haveG, Kind: haveK, Version: haveV}] = struct{}{}
			}
		}
	}

	r.have.Lock()
	r.have.APISet = haveWantedAPIs
	r.have.Unlock()

	return nil
}

// Start periodically reads the Kuberentes API. It blocks until ctx is cancelled.
func (r *APIDiscoveryRunner) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.refresh)
	defer ticker.Stop()

	log := logging.FromContext(ctx).WithValues("controller", "kubernetes")

	for {
		select {
		case <-ticker.C:
			if err := r.Read(); err != nil {
				log.Error(err, "Unable to detect Kubernetes APIs")
			}
		case <-ctx.Done():
			// TODO(controller-runtime): Fixed in v0.19.0
			// https://github.com/kubernetes-sigs/controller-runtime/issues/1927
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		}
	}
}

// Has returns true when api is available in Kuberentes.
func (r *APIDiscoveryRunner) Has(api API) bool { return r.HasOne(api) }

// HasAll returns true when every api is available in Kubernetes.
func (r *APIDiscoveryRunner) HasAll(api ...API) bool {
	r.have.RLock()
	defer r.have.RUnlock()
	return r.have.HasAll(api...)
}

// HasOne returns true when at least one api is available in Kubernetes.
func (r *APIDiscoveryRunner) HasOne(api ...API) bool {
	r.have.RLock()
	defer r.have.RUnlock()
	return r.have.HasOne(api...)
}

type apiContextKey struct{}

// Kubernetes returns the APIs previously stored by [NewAPIContext].
// When nothing was stored, it returns an empty [APISet].
func Kubernetes(ctx context.Context) APIs {
	if apis, ok := ctx.Value(apiContextKey{}).(APIs); ok {
		return apis
	}
	return APISet{}
}

// NewAPIContext returns a copy of ctx containing apis. Retrieve it using [Kubernetes].
func NewAPIContext(ctx context.Context, apis APIs) context.Context {
	return context.WithValue(ctx, apiContextKey{}, apis)
}
