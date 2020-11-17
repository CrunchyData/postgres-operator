package runtime

/*
Copyright 2020 Crunchy Data
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
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/controller/postgrescluster"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

// the name of the event recorder created for the postgrescluster controller
const recorderName = "postgrescluster-controller"

// default refresh interval in minutes
var refreshInterval = 60 * time.Minute

// CreateRuntimeManager creates a new controller runtime manager for the PostgreSQL Operator.  The
// manager returned is configured specifically for the PostgreSQL Operator, and includes any
// controllers that will be responsible for managing PostgreSQL clusters using the
// 'postgrescluster' custom resource.  Additionally, the manager will only watch for resources in
// the namespace specified, with an empty string resulting in the manager watching all namespaces.
func CreateRuntimeManager(namespace string) (manager.Manager, error) {

	pgoScheme, err := createPostgresOperatorScheme()
	if err != nil {
		return nil, err
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// create controller runtime manager
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:  namespace, // if empty then watching all namespaces
		SyncPeriod: &refreshInterval,
		Scheme:     pgoScheme,
	})
	if err != nil {
		return nil, err
	}

	// add all PostgreSQL Operator controllers to the runtime manager
	if err := addControllersToManager(mgr); err != nil {
		return nil, err
	}

	return mgr, nil
}

// addControllersToManager adds all PostgreSQL Operator controllers to the provided controller
// runtime manager.
func addControllersToManager(mgr manager.Manager) error {
	r := &postgrescluster.Reconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor(recorderName)}
	if err := r.SetupWithManager(mgr); err != nil {
		return err
	}
	return nil
}

// createPostgresOperatorScheme creates a scheme containing the resource types required by the
// PostgreSQL Operator.  This includes any custom resource types specific to the PostgreSQL
// Operator, as well as any standard Kubernetes resource types.
func createPostgresOperatorScheme() (*runtime.Scheme, error) {

	// create a new scheme specifically for this manager
	pgoScheme := runtime.NewScheme()

	// add standard resource types to the scheme
	if err := scheme.AddToScheme(pgoScheme); err != nil {
		return nil, err
	}

	// add custom resource types to the default scheme
	if err := v1alpha1.AddToScheme(pgoScheme); err != nil {
		return nil, err
	}

	return pgoScheme, nil
}
