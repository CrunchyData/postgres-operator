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
	"k8s.io/klog/v2"
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
	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/internal/tracing"
	"github.com/crunchydata/postgres-operator/internal/upgradecheck"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// must panics when err is not nil.
func must(err error) { need(0, err) }
func need[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}

func initClient() (*rest.Config, error) {
	config, err := runtime.GetConfig()

	if err == nil && userAgent == "" {
		err = errors.New("call initVersion first")
	}
	if err == nil {
		config.UserAgent = userAgent
		config.Wrap(otelTransportWrapper())

		// Log Kubernetes API warnings encountered by client-go at a high verbosity.
		// See [rest.WarningLogger].
		handler := runtime.WarningHandler(func(ctx context.Context, code int, _ string, message string) {
			if code == 299 && len(message) != 0 {
				logging.FromContext(ctx).V(2).WithName("client-go").Info(message)
			}
		})
		config.WarningHandler = handler
		config.WarningHandlerWithContext = handler
	}

	return config, err
}

func initLogging() {
	debug := strings.TrimSpace(os.Getenv("CRUNCHY_DEBUG"))

	var verbosity int
	if strings.EqualFold(debug, "true") {
		verbosity = 1
	} else if i, err := strconv.Atoi(debug); err == nil && i > 0 {
		verbosity = i
	}

	// Configure a singleton that treats logging.Logger.V(1) as logrus.DebugLevel.
	logging.SetLogSink(logging.Logrus(os.Stdout, versionString, 1, verbosity))

	global := logging.FromContext(context.Background())
	runtime.SetLogger(global)

	// [k8s.io/client-go/tools/leaderelection] logs to the global [klog] instance.
	// - https://github.com/kubernetes-sigs/controller-runtime/issues/2656
	klog.SetLoggerWithOptions(global, klog.ContextualLogger(true))
}

//+kubebuilder:rbac:groups="coordination.k8s.io",resources="leases",verbs={get,create,update,watch}
//+kubebuilder:rbac:groups="authentication.k8s.io",resources="tokenreviews",verbs={create}
//+kubebuilder:rbac:groups="authorization.k8s.io",resources="subjectaccessreviews",verbs={create}

func initManager(ctx context.Context) (runtime.Options, error) {
	log := logging.FromContext(ctx).WithName("manager")

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
	must(features.Set(os.Getenv("PGO_FEATURE_GATES")))

	running = feature.NewContext(running, features)
	log.Info("feature gates",
		// These are set by the user
		"PGO_FEATURE_GATES", feature.ShowAssigned(running),
		// These are enabled, including features that are on by default
		"enabled", feature.ShowEnabled(running))

	// Initialize OpenTelemetry and flush data when there is a panic.
	otelFinish := need(initOpenTelemetry(running))
	defer func(ctx context.Context) { _ = otelFinish(ctx) }(running)

	tracing.SetDefaultTracer(tracing.New("github.com/CrunchyData/postgres-operator"))

	// Load Kubernetes client configuration and ensure it works.
	config := need(initClient())
	k8s := need(kubernetes.NewDiscoveryRunner(config))
	must(k8s.Read(running))
	log.Info("connected to Kubernetes", "api", k8s.Version().String(), "openshift", k8s.IsOpenShift())

	options := need(initManager(running))

	// Add to the Context that Manager passes to Reconciler.Start, Runnable.Start,
	// and eventually Reconciler.Reconcile.
	options.BaseContext = func() context.Context {
		ctx := context.Background()
		ctx = feature.NewContext(ctx, features)
		ctx = kubernetes.NewAPIContext(ctx, k8s)
		return ctx
	}

	manager := need(runtime.NewManager(config, options))
	must(manager.Add(k8s))

	registrar := need(registration.NewRunner(os.Getenv("RSA_KEY"), os.Getenv("TOKEN_PATH"), stopRunning))
	must(manager.Add(registrar))
	token, _ := registrar.CheckToken()

	bridgeURL := os.Getenv("PGO_BRIDGE_URL")
	bridgeClient := func() *bridge.Client {
		client := bridge.NewClient(bridgeURL, versionString)
		client.Transport = otelTransportWrapper()(http.DefaultTransport)
		return client
	}

	// add all PostgreSQL Operator controllers to the runtime manager
	addControllersToManager(manager, log, registrar)
	must(pgupgrade.ManagedReconciler(manager, registrar))
	must(standalone_pgadmin.ManagedReconciler(manager))
	must(crunchybridgecluster.ManagedReconciler(manager, func() bridge.ClientInterface {
		return bridgeClient()
	}))

	if features.Enabled(feature.BridgeIdentifiers) {
		must(bridge.ManagedInstallationReconciler(manager, bridgeClient))
	}

	// Enable upgrade checking
	upgradeCheckingDisabled := strings.EqualFold(os.Getenv("CHECK_FOR_UPGRADES"), "false")
	if !upgradeCheckingDisabled {
		log.Info("upgrade checking enabled")
		url := os.Getenv("CHECK_FOR_UPGRADES_URL")
		must(upgradecheck.ManagedScheduler(manager, url, versionString, token))
	} else {
		log.Info("upgrade checking disabled")
	}

	// Enable health probes
	must(manager.AddHealthzCheck("health", healthz.Ping))
	must(manager.AddReadyzCheck("check", healthz.Ping))

	// Start the manager and wait for its context to be canceled.
	stopped := make(chan error, 1)
	go func() { stopped <- manager.Start(running) }()
	<-running.Done()

	// Set a deadline for graceful termination.
	log.Info("shutting down")
	stopping, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Wait for the manager to return or the deadline to pass.
	var err error
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
func addControllersToManager(mgr runtime.Manager, log logging.Logger, reg registration.Registration) {
	pgReconciler := &postgrescluster.Reconciler{
		Client:       mgr.GetClient(),
		Owner:        naming.ControllerPostgresCluster,
		Recorder:     mgr.GetEventRecorderFor(naming.ControllerPostgresCluster),
		Registration: reg,
	}

	if err := pgReconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create PostgresCluster controller")
		os.Exit(1)
	}
}
