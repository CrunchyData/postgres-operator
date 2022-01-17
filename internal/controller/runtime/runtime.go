package runtime

/*
Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// default refresh interval in minutes
var refreshInterval = 60 * time.Minute

// CreateRuntimeManager creates a new controller runtime manager for the PostgreSQL Operator.  The
// manager returned is configured specifically for the PostgreSQL Operator, and includes any
// controllers that will be responsible for managing PostgreSQL clusters using the
// 'postgrescluster' custom resource.  Additionally, the manager will only watch for resources in
// the namespace specified, with an empty string resulting in the manager watching all namespaces.
func CreateRuntimeManager(namespace string, config *rest.Config,
	disableMetrics bool) (manager.Manager, error) {

	pgoScheme, err := CreatePostgresOperatorScheme()
	if err != nil {
		return nil, err
	}

	options := manager.Options{
		Namespace:  namespace, // if empty then watching all namespaces
		SyncPeriod: &refreshInterval,
		Scheme:     pgoScheme,
	}
	if disableMetrics {
		options.HealthProbeBindAddress = "0"
		options.MetricsBindAddress = "0"
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

// CreatePostgresOperatorScheme creates a scheme containing the resource types required by the
// PostgreSQL Operator.  This includes any custom resource types specific to the PostgreSQL
// Operator, as well as any standard Kubernetes resource types.
func CreatePostgresOperatorScheme() (*runtime.Scheme, error) {

	// create a new scheme specifically for this manager
	pgoScheme := runtime.NewScheme()

	// add standard resource types to the scheme
	if err := scheme.AddToScheme(pgoScheme); err != nil {
		return nil, err
	}

	// add custom resource types to the default scheme
	if err := v1beta1.AddToScheme(pgoScheme); err != nil {
		return nil, err
	}

	return pgoScheme, nil
}
