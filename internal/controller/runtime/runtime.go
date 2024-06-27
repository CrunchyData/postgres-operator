/*
Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// Scheme associates standard Kubernetes API objects and PGO API objects with Go structs.
var Scheme *runtime.Scheme = runtime.NewScheme()

func init() {
	if err := scheme.AddToScheme(Scheme); err != nil {
		panic(err)
	}
	if err := v1beta1.AddToScheme(Scheme); err != nil {
		panic(err)
	}
}

// default refresh interval in minutes
var refreshInterval = 60 * time.Minute

// CreateRuntimeManager creates a new controller runtime manager for the PostgreSQL Operator.  The
// manager returned is configured specifically for the PostgreSQL Operator, and includes any
// controllers that will be responsible for managing PostgreSQL clusters using the
// 'postgrescluster' custom resource.  Additionally, the manager will only watch for resources in
// the namespace specified, with an empty string resulting in the manager watching all namespaces.

// +kubebuilder:rbac:groups="coordination.k8s.io",resources="leases",verbs={get,create,update}

func CreateRuntimeManager(ctx context.Context, namespace string, config *rest.Config,
	disableMetrics bool) (manager.Manager, error) {
	log := log.FromContext(ctx)

	// Watch all namespaces by default
	options := manager.Options{
		Cache: cache.Options{
			SyncPeriod: &refreshInterval,
		},

		Scheme: Scheme,
	}
	// If namespace is not empty then add namespace to DefaultNamespaces
	if len(namespace) > 0 {
		options.Cache.DefaultNamespaces = map[string]cache.Config{
			namespace: {},
		}
	}
	if disableMetrics {
		options.HealthProbeBindAddress = "0"
		options.Metrics.BindAddress = "0"
	}

	// Add leader election options
	options, err := addLeaderElectionOptions(options)
	if err != nil {
		return nil, err
	} else {
		log.Info("Leader election enabled.")
	}

	// create controller runtime manager
	mgr, err := manager.New(config, options)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

// GetConfig creates a *rest.Config for talking to a Kubernetes API server.
func GetConfig() (*rest.Config, error) { return config.GetConfig() }

// addLeaderElectionOptions takes the manager.Options as an argument and will
// add leader election options if PGO_CONTROLLER_LEASE_NAME is set and valid.
// If PGO_CONTROLLER_LEASE_NAME is not valid, the function will return the
// original options and an error. If PGO_CONTROLLER_LEASE_NAME is not set at all,
// the function will return the original options.
func addLeaderElectionOptions(opts manager.Options) (manager.Options, error) {
	errs := []error{}

	leaderLeaseName := os.Getenv("PGO_CONTROLLER_LEASE_NAME")
	if len(leaderLeaseName) > 0 {
		// If no errors are returned by IsDNS1123Subdomain(), turn on leader election,
		// otherwise, return the errors
		dnsSubdomainErrors := validation.IsDNS1123Subdomain(leaderLeaseName)
		if len(dnsSubdomainErrors) == 0 {
			opts.LeaderElection = true
			opts.LeaderElectionNamespace = os.Getenv("PGO_NAMESPACE")
			opts.LeaderElectionID = leaderLeaseName
		} else {
			for _, errString := range dnsSubdomainErrors {
				err := errors.New(errString)
				errs = append(errs, err)
			}

			return opts, fmt.Errorf("value for PGO_CONTROLLER_LEASE_NAME is invalid: %v", errs)
		}
	}

	return opts, nil
}

// SetLogger assigns the default Logger used by [sigs.k8s.io/controller-runtime].
func SetLogger(logger logging.Logger) { log.SetLogger(logger) }
