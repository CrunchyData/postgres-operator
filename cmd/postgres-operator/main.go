// Copyright 2017 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.opentelemetry.io/otel"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/bridge/crunchybridgecluster"
	"github.com/crunchydata/postgres-operator/internal/controller/pgupgrade"
	"github.com/crunchydata/postgres-operator/internal/controller/postgrescluster"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/controller/standalone_pgadmin"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/internal/upgradecheck"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var versionString string

// assertNoError panics when err is not nil.
func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}

func initLogging() {
	// Configure a singleton that treats logging.Logger.V(1) as logrus.DebugLevel.
	var verbosity int
	if strings.EqualFold(os.Getenv("CRUNCHY_DEBUG"), "true") {
		verbosity = 1
	}
	logging.SetLogSink(logging.Logrus(os.Stdout, versionString, 1, verbosity))

	global := logging.FromContext(context.Background())
	runtime.SetLogger(global)
}

//+kubebuilder:rbac:groups="coordination.k8s.io",resources="leases",verbs={get,create,update}

func initManager() (runtime.Options, error) {
	log := logging.FromContext(context.Background())

	options := runtime.Options{}
	options.Cache.SyncPeriod = initialize.Pointer(time.Hour)

	options.HealthProbeBindAddress = ":8081"

	// Enable leader elections when configured with a valid Lease.coordination.k8s.io name.
	// - https://docs.k8s.io/concepts/architecture/leases
	// - https://releases.k8s.io/v1.30.0/pkg/apis/coordination/validation/validation.go#L26
	if lease := os.Getenv("PGO_CONTROLLER_LEASE_NAME"); len(lease) > 0 {
		if errs := validation.IsDNS1123Subdomain(lease); len(errs) > 0 {
			return options, fmt.Errorf("value for PGO_CONTROLLER_LEASE_NAME is invalid: %v", errs)
		}

		options.LeaderElection = true
		options.LeaderElectionID = lease
		options.LeaderElectionNamespace = os.Getenv("PGO_NAMESPACE")
	}

	// Check PGO_TARGET_NAMESPACE for backwards compatibility with
	// "singlenamespace" installations
	singlenamespace := strings.TrimSpace(os.Getenv("PGO_TARGET_NAMESPACE"))

	// Check PGO_TARGET_NAMESPACES for non-cluster-wide, multi-namespace
	// installations
	multinamespace := strings.TrimSpace(os.Getenv("PGO_TARGET_NAMESPACES"))

	// Initialize DefaultNamespaces if any target namespaces are set
	if len(singlenamespace) > 0 || len(multinamespace) > 0 {
		options.Cache.DefaultNamespaces = map[string]runtime.CacheConfig{}
	}

	if len(singlenamespace) > 0 {
		options.Cache.DefaultNamespaces[singlenamespace] = runtime.CacheConfig{}
	}

	if len(multinamespace) > 0 {
		for _, namespace := range strings.FieldsFunc(multinamespace, func(c rune) bool {
			return c != '-' && !unicode.IsLetter(c) && !unicode.IsNumber(c)
		}) {
			options.Cache.DefaultNamespaces[namespace] = runtime.CacheConfig{}
		}
	}

	options.Controller.GroupKindConcurrency = map[string]int{
		"PostgresCluster." + v1beta1.GroupVersion.Group: 2,
	}

	if s := os.Getenv("PGO_WORKERS"); s != "" {
		if i, err := strconv.Atoi(s); err == nil && i > 0 {
			options.Controller.GroupKindConcurrency["PostgresCluster."+v1beta1.GroupVersion.Group] = i
		} else {
			log.Error(err, "PGO_WORKERS must be a positive number")
		}
	}

	return options, nil
}

func main() {
	// This context is canceled by SIGINT, SIGTERM, or by calling shutdown.
	ctx, shutdown := context.WithCancel(runtime.SignalHandler())

	otelFlush, err := initOpenTelemetry()
	assertNoError(err)
	defer otelFlush()

	initLogging()

	log := logging.FromContext(ctx)
	log.V(1).Info("debug flag set to true")

	features := feature.NewGate()
	assertNoError(features.Set(os.Getenv("PGO_FEATURE_GATES")))
	log.Info("feature gates enabled", "PGO_FEATURE_GATES", features.String())

	cfg, err := runtime.GetConfig()
	assertNoError(err)

	cfg.Wrap(otelTransportWrapper())

	// Configure client-go to suppress warnings when warning headers are encountered. This prevents
	// warnings from being logged over and over again during reconciliation (e.g. this will suppress
	// deprecation warnings when using an older version of a resource for backwards compatibility).
	rest.SetDefaultWarningHandler(rest.NoWarnings{})

	options, err := initManager()
	assertNoError(err)

	// Add to the Context that Manager passes to Reconciler.Start, Runnable.Start,
	// and eventually Reconciler.Reconcile.
	options.BaseContext = func() context.Context {
		ctx := context.Background()
		ctx = feature.NewContext(ctx, features)
		return ctx
	}

	mgr, err := runtime.NewManager(cfg, options)
	assertNoError(err)

	openshift := isOpenshift(cfg)
	if openshift {
		log.Info("detected OpenShift environment")
	}

	registrar, err := registration.NewRunner(os.Getenv("RSA_KEY"), os.Getenv("TOKEN_PATH"), shutdown)
	assertNoError(err)
	assertNoError(mgr.Add(registrar))
	_ = registrar.CheckToken()

	// add all PostgreSQL Operator controllers to the runtime manager
	addControllersToManager(mgr, openshift, log, registrar)

	if features.Enabled(feature.BridgeIdentifiers) {
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

	// Enable health probes
	assertNoError(mgr.AddHealthzCheck("health", healthz.Ping))
	assertNoError(mgr.AddReadyzCheck("check", healthz.Ping))

	log.Info("starting controller runtime manager and will wait for signal to exit")

	assertNoError(mgr.Start(ctx))
	log.Info("signal received, exiting")
}

// addControllersToManager adds all PostgreSQL Operator controllers to the provided controller
// runtime manager.
func addControllersToManager(mgr runtime.Manager, openshift bool, log logging.Logger, reg registration.Registration) {
	pgReconciler := &postgrescluster.Reconciler{
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
