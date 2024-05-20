package main

/*
Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
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
	"net/http"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	cruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/autogrow"
	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/bridge/crunchybridgecluster"
	"github.com/crunchydata/postgres-operator/internal/controller/pgupgrade"
	"github.com/crunchydata/postgres-operator/internal/controller/postgrescluster"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/controller/standalone_pgadmin"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/internal/upgradecheck"
	"github.com/crunchydata/postgres-operator/internal/util"
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
	logging.SetLogSink(logging.Logrus(os.Stdout, versionString, 1, verbosity))
}

func main() {
	// Set any supplied feature gates; panic on any unrecognized feature gate
	err := util.AddAndSetFeatureGates(os.Getenv("PGO_FEATURE_GATES"))
	assertNoError(err)

	otelFlush, err := initOpenTelemetry()
	assertNoError(err)
	defer otelFlush()

	initLogging()

	// create a context that will be used to stop all controllers on a SIGTERM or SIGINT
	ctx := cruntime.SetupSignalHandler()
	ctx, shutdown := context.WithCancel(ctx)
	log := logging.FromContext(ctx)
	log.V(1).Info("debug flag set to true")

	log.Info("feature gates enabled",
		"PGO_FEATURE_GATES", os.Getenv("PGO_FEATURE_GATES"))

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

	openshift := isOpenshift(cfg)
	if openshift {
		log.Info("detected OpenShift environment")
	}

	registrar, err := registration.NewRunner(os.Getenv("RSA_KEY"), os.Getenv("TOKEN_PATH"), shutdown)
	assertNoError(err)
	assertNoError(mgr.Add(registrar))
	_ = registrar.CheckToken()

	autogrow, err := autogrow.NewRunner(mgr.GetConfig(), log)
	assertNoError(mgr.Add(autogrow))

	// add all PostgreSQL Operator controllers to the runtime manager
	addControllersToManager(mgr, openshift, log, registrar, autogrow)

	if util.DefaultMutableFeatureGate.Enabled(util.BridgeIdentifiers) {
		constructor := func() *bridge.Client {
			client := bridge.NewClient(os.Getenv("PGO_BRIDGE_URL"), versionString)
			client.Transport = otelTransportWrapper()(http.DefaultTransport)
			return client
		}

		assertNoError(bridge.ManagedInstallationReconciler(mgr, constructor))
	}

	// Enable upgrade checking
	upgradeCheckingDisabled := strings.EqualFold(os.Getenv("CHECK_FOR_UPGRADES"), "false")
	if !upgradeCheckingDisabled {
		log.Info("upgrade checking enabled")
		// get the URL for the check for upgrades endpoint if set in the env
		assertNoError(upgradecheck.ManagedScheduler(mgr,
			openshift, os.Getenv("CHECK_FOR_UPGRADES_URL"), versionString))
	} else {
		log.Info("upgrade checking disabled")
	}

	log.Info("starting controller runtime manager and will wait for signal to exit")

	assertNoError(mgr.Start(ctx))
	log.Info("signal received, exiting")
}

// addControllersToManager adds all PostgreSQL Operator controllers to the provided controller
// runtime manager.
func addControllersToManager(mgr manager.Manager, openshift bool, log logr.Logger, reg registration.Registration, autogrow autogrow.Autogrow) {
	pgReconciler := &postgrescluster.Reconciler{
		Autogrow:     autogrow,
		Client:       mgr.GetClient(),
		IsOpenShift:  openshift,
		Owner:        postgrescluster.ControllerName,
		Recorder:     mgr.GetEventRecorderFor(postgrescluster.ControllerName),
		Registration: reg,
		Tracer:       otel.Tracer(postgrescluster.ControllerName),
	}

	if err := pgReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create PostgresCluster controller")
		os.Exit(1)
	}

	upgradeReconciler := &pgupgrade.PGUpgradeReconciler{
		Client:       mgr.GetClient(),
		Owner:        "pgupgrade-controller",
		Recorder:     mgr.GetEventRecorderFor("pgupgrade-controller"),
		Registration: reg,
	}

	if err := upgradeReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create PGUpgrade controller")
		os.Exit(1)
	}

	pgAdminReconciler := &standalone_pgadmin.PGAdminReconciler{
		Client:      mgr.GetClient(),
		Owner:       "pgadmin-controller",
		Recorder:    mgr.GetEventRecorderFor(naming.ControllerPGAdmin),
		IsOpenShift: openshift,
	}

	if err := pgAdminReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create PGAdmin controller")
		os.Exit(1)
	}

	constructor := func() bridge.ClientInterface {
		client := bridge.NewClient(os.Getenv("PGO_BRIDGE_URL"), versionString)
		client.Transport = otelTransportWrapper()(http.DefaultTransport)
		return client
	}

	crunchyBridgeClusterReconciler := &crunchybridgecluster.CrunchyBridgeClusterReconciler{
		Client: mgr.GetClient(),
		Owner:  "crunchybridgecluster-controller",
		// TODO(crunchybridgecluster): recorder?
		// Recorder: mgr.GetEventRecorderFor(naming...),
		NewClient: constructor,
	}

	if err := crunchyBridgeClusterReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CrunchyBridgeCluster controller")
		os.Exit(1)
	}
}

func isOpenshift(cfg *rest.Config) bool {
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
					return true
				}
			}
		}
	}

	return false
}
