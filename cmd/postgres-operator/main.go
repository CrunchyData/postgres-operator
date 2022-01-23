package main

/*
Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	cruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/controller/postgrescluster"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/upgradecheck"
)

var versionString string

// assertNoError panics when err is not nil.
func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}

func initLogging() {
	// Configure a singleton that treats logr.Logger.V(1) as logrus.DebugLevel.
	var verbosity int
	if strings.EqualFold(os.Getenv("CRUNCHY_DEBUG"), "true") {
		verbosity = 1
	}
	logging.SetLogFunc(verbosity, logging.Logrus(os.Stdout, versionString, 1))
}

func main() {
	otelFlush, err := initOpenTelemetry()
	assertNoError(err)
	defer otelFlush()

	initLogging()

	// create a context that will be used to stop all controllers on a SIGTERM or SIGINT
	ctx := cruntime.SetupSignalHandler()
	log := logging.FromContext(ctx)
	log.V(1).Info("debug flag set to true")

	cruntime.SetLogger(log)

	cfg, err := runtime.GetConfig()
	assertNoError(err)

	cfg.Wrap(otelTransportWrapper())

	// Configure client-go to suppress warnings when warning headers are encountered. This prevents
	// warnings from being logged over and over again during reconciliation (e.g. this will suppress
	// deprecation warnings when using an older version of a resource for backwards compatibility).
	rest.SetDefaultWarningHandler(rest.NoWarnings{})

	mgr, err := runtime.CreateRuntimeManager(os.Getenv("PGO_TARGET_NAMESPACE"), cfg, false)
	assertNoError(err)

	// add all PostgreSQL Operator controllers to the runtime manager
	err = addControllersToManager(ctx, mgr)
	assertNoError(err)

	log.Info("starting controller runtime manager and will wait for signal to exit")

	// Enable upgrade checking
	upgradeCheckingDisabled := strings.EqualFold(os.Getenv("CHECK_FOR_UPGRADES"), "false")
	if !upgradeCheckingDisabled {
		log.Info("upgrade checking enabled")
		// get the URL for the check for upgrades endpoint if set in the env
		upgradeCheckURL := os.Getenv("CHECK_FOR_UPGRADES_URL")
		go upgradecheck.CheckForUpgradesScheduler(ctx, versionString, upgradeCheckURL,
			mgr.GetClient(), mgr.GetConfig(), isOpenshift(ctx, mgr.GetConfig()),
			mgr.GetCache(),
		)
	} else {
		log.Info("upgrade checking disabled")
	}

	assertNoError(mgr.Start(ctx))
	log.Info("signal received, exiting")
}

// addControllersToManager adds all PostgreSQL Operator controllers to the provided controller
// runtime manager.
func addControllersToManager(ctx context.Context, mgr manager.Manager) error {
	r := &postgrescluster.Reconciler{
		Client:      mgr.GetClient(),
		Owner:       postgrescluster.ControllerName,
		Recorder:    mgr.GetEventRecorderFor(postgrescluster.ControllerName),
		Tracer:      otel.Tracer(postgrescluster.ControllerName),
		IsOpenShift: isOpenshift(ctx, mgr.GetConfig()),
	}
	return r.SetupWithManager(mgr)
}

func isOpenshift(ctx context.Context, cfg *rest.Config) bool {
	log := logging.FromContext(ctx)

	const sccGroupName, sccKind = "security.openshift.io", "SecurityContextConstraints"

	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	assertNoError(err)

	groups, err := client.ServerGroups()
	if err != nil {
		assertNoError(err)
	}
	for _, g := range groups.Groups {
		if g.Name != sccGroupName {
			continue
		}
		for _, v := range g.Versions {
			resourceList, err := client.ServerResourcesForGroupVersion(v.GroupVersion)
			if err != nil {
				assertNoError(err)
			}
			for _, r := range resourceList.APIResources {
				if r.Kind == sccKind {
					log.Info("detected OpenShift environment")
					return true
				}
			}
		}
	}

	return false
}
