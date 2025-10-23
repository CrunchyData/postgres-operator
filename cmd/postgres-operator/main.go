// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/bridge/crunchybridgecluster"
	"github.com/crunchydata/postgres-operator/internal/controller/pgupgrade"
	"github.com/crunchydata/postgres-operator/internal/controller/postgrescluster"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/controller/standalone_pgadmin"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/kubernetes"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/tracing"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

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

//+kubebuilder:rbac:groups="coordination.k8s.io",resources="leases",verbs={get,create,update,watch}
//+kubebuilder:rbac:groups="authentication.k8s.io",resources="tokenreviews",verbs={create}
//+kubebuilder:rbac:groups="authorization.k8s.io",resources="subjectaccessreviews",verbs={create}

func initManager(ctx context.Context) (runtime.Options, error) {
	log := logging.FromContext(ctx)

	options := runtime.Options{}
	options.Cache.SyncPeriod = initialize.Pointer(time.Hour)

	// If we aren't using it, http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	options.Metrics.TLSOpts = append(options.Metrics.TLSOpts, func(c *tls.Config) {
		log.Info("enabling metrics via http/1.1")
		c.NextProtos = []string{"http/1.1"}
	})

	// Use https port
	options.Metrics.BindAddress = ":8443"
	options.Metrics.SecureServing = true

	// FilterProvider is used to protect the metrics endpoint with authn/authz.
	// These configurations ensure that only authorized users and service accounts
	// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
	// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.3/pkg/metrics/filters#WithAuthenticationAndAuthorization
	options.Metrics.FilterProvider = filters.WithAuthenticationAndAuthorization

	// Set health probe port
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
	running, stopRunning := context.WithCancel(context.Background())
	defer stopRunning()

	initVersion()
	initLogging()
	log := logging.FromContext(running)
	log.V(1).Info("debug flag set to true")

	// Start a goroutine that waits for SIGINT or SIGTERM.
	{
		signals := []os.Signal{os.Interrupt, syscall.SIGTERM}
		receive := make(chan os.Signal, len(signals))
		signal.Notify(receive, signals...)
		go func() {
			// Wait for a signal then immediately restore the default signal handlers.
			// After this, a SIGHUP, SIGINT, or SIGTERM causes the program to exit.
			// - https://pkg.go.dev/os/signal#hdr-Default_behavior_of_signals_in_Go_programs
			s := <-receive
			signal.Stop(receive)

			log.Info("received signal from OS", "signal", s.String())
			stopRunning()
		}()
	}

	features := feature.NewGate()
	assertNoError(features.Set(os.Getenv("PGO_FEATURE_GATES")))

	running = feature.NewContext(running, features)
	log.Info("feature gates",
		// These are set by the user
		"PGO_FEATURE_GATES", feature.ShowAssigned(running),
		// These are enabled, including features that are on by default
		"enabled", feature.ShowEnabled(running))

	// Initialize OpenTelemetry and flush data when there is a panic.
	otelFinish, err := initOpenTelemetry(running)
	assertNoError(err)
	defer func(ctx context.Context) { _ = otelFinish(ctx) }(running)

	tracing.SetDefaultTracer(tracing.New("github.com/CrunchyData/postgres-operator"))

	cfg, err := runtime.GetConfig()
	assertNoError(err)

	cfg.UserAgent = userAgent
	cfg.Wrap(otelTransportWrapper())

	// TODO(controller-runtime): Set config.WarningHandler instead after v0.19.0.
	// Configure client-go to suppress warnings when warning headers are encountered. This prevents
	// warnings from being logged over and over again during reconciliation (e.g. this will suppress
	// deprecation warnings when using an older version of a resource for backwards compatibility).
	rest.SetDefaultWarningHandler(rest.NoWarnings{})

	k8s, err := kubernetes.NewDiscoveryRunner(cfg)
	assertNoError(err)
	assertNoError(k8s.Read(running))

	log.Info("connected to Kubernetes", "api", k8s.Version().String(), "openshift", k8s.IsOpenShift())

	options, err := initManager(running)
	assertNoError(err)

	// Add to the Context that Manager passes to Reconciler.Start, Runnable.Start,
	// and eventually Reconciler.Reconcile.
	options.BaseContext = func() context.Context {
		ctx := context.Background()
		ctx = feature.NewContext(ctx, features)
		ctx = kubernetes.NewAPIContext(ctx, k8s)
		return ctx
	}

	mgr, err := runtime.NewManager(cfg, options)
	assertNoError(err)
	assertNoError(mgr.Add(k8s))

	// add all PostgreSQL Operator controllers to the runtime manager
	addControllersToManager(mgr, log)

	if features.Enabled(feature.BridgeIdentifiers) {
		constructor := func() *bridge.Client {
			client := bridge.NewClient(os.Getenv("PGO_BRIDGE_URL"), versionString)
			client.Transport = otelTransportWrapper()(http.DefaultTransport)
			return client
		}

		assertNoError(bridge.ManagedInstallationReconciler(mgr, constructor))
	}

	// Enable health probes
	assertNoError(mgr.AddHealthzCheck("health", healthz.Ping))
	assertNoError(mgr.AddReadyzCheck("check", healthz.Ping))

	// Start the manager and wait for its context to be canceled.
	stopped := make(chan error, 1)
	go func() { stopped <- mgr.Start(running) }()
	<-running.Done()

	// Set a deadline for graceful termination.
	log.Info("shutting down")
	stopping, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Wait for the manager to return or the deadline to pass.
	select {
	case err = <-stopped:
	case <-stopping.Done():
		err = stopping.Err()
	}

	// Flush any telemetry with the remaining time we have.
	if err = errors.Join(err, otelFinish(stopping)); err != nil {
		log.Error(err, "shutdown failed")
	} else {
		log.Info("shutdown complete")
	}
}

// addControllersToManager adds all PostgreSQL Operator controllers to the provided controller
// runtime manager.
func addControllersToManager(mgr runtime.Manager, log logging.Logger) {
	pgReconciler := &postgrescluster.Reconciler{
		Client:   mgr.GetClient(),
		Owner:    postgrescluster.ControllerName,
		Recorder: mgr.GetEventRecorderFor(postgrescluster.ControllerName),
	}

	if err := pgReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create PostgresCluster controller")
		os.Exit(1)
	}

	upgradeReconciler := &pgupgrade.PGUpgradeReconciler{
		Client:   mgr.GetClient(),
		Owner:    "pgupgrade-controller",
		Recorder: mgr.GetEventRecorderFor("pgupgrade-controller"),
	}

	if err := upgradeReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create PGUpgrade controller")
		os.Exit(1)
	}

	pgAdminReconciler := &standalone_pgadmin.PGAdminReconciler{
		Client:   mgr.GetClient(),
		Owner:    "pgadmin-controller",
		Recorder: mgr.GetEventRecorderFor(naming.ControllerPGAdmin),
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
