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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type (
	CacheConfig = cache.Config
	Manager     = manager.Manager
	Options     = manager.Options
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

// GetConfig returns a Kubernetes client configuration from KUBECONFIG or the
// service account Kubernetes gives to pods.
func GetConfig() (*rest.Config, error) { return config.GetConfig() }

// NewManager returns a Manager that interacts with the Kubernetes API of config.
// When config is nil, it reads from KUBECONFIG or the local service account.
// When options.Scheme is nil, it uses the Scheme from this package.
func NewManager(config *rest.Config, options manager.Options) (manager.Manager, error) {
	var m manager.Manager
	var err error

	if config == nil {
		config, err = GetConfig()
	}

	if options.Scheme == nil {
		options.Scheme = Scheme
	}

	if err == nil {
		m, err = manager.New(config, options)
	}

	return m, err
}

// SetLogger assigns the default Logger used by [sigs.k8s.io/controller-runtime].
func SetLogger(logger logging.Logger) { log.SetLogger(logger) }

// SignalHandler returns a Context that is canceled on SIGINT or SIGTERM.
func SignalHandler() context.Context { return signals.SetupSignalHandler() }
