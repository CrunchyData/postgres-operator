// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package require

import (
	"context"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
)

type TestingT interface {
	assert.TestingT
	Cleanup(func())
	Helper()
	Name() string
	SkipNow()
}

// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#pkg-constants
var envtestVarsSet = os.Getenv("KUBEBUILDER_ASSETS") != "" ||
	strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true")

// EnvTest returns an unstarted Environment with crds. It calls t.Skip when
// the "KUBEBUILDER_ASSETS" and "USE_EXISTING_CLUSTER" environment variables
// are unset.
func EnvTest(t TestingT, crds envtest.CRDInstallOptions) *envtest.Environment {
	t.Helper()

	if !envtestVarsSet {
		t.SkipNow()
	}

	return &envtest.Environment{
		CRDInstallOptions: crds,
		Scheme:            crds.Scheme,
	}
}

var kubernetes struct {
	sync.Mutex

	// Count references to the started Environment.
	count int
	env   *envtest.Environment
	err   error
}

// Kubernetes starts or connects to a Kubernetes API and returns a client that uses it.
// When starting a local API, the client is a member of the "system:masters" group.
//
// It calls t.Fatal when something fails. It stops the local API using t.Cleanup.
// It calls t.Skip when the "KUBEBUILDER_ASSETS" and "USE_EXISTING_CLUSTER" environment
// variables are unset.
//
// Tests that call t.Parallel might share the same local API. Call t.Parallel after this
// function to ensure they share.
func Kubernetes(t TestingT) client.Client {
	t.Helper()
	_, cc := kubernetes3(t)
	return cc
}

// KubernetesAtLeast is the same as [Kubernetes] but also calls t.Skip when
// the connected Kubernetes API is earlier than minVersion, like "1.28" or "1.27.7".
func KubernetesAtLeast(t TestingT, minVersion string) client.Client {
	t.Helper()

	expectedVersion, err := version.ParseGeneric(minVersion)
	assert.NilError(t, err)

	// Start or connect to Kubernetes
	env, cc := kubernetes3(t)

	dc, err := discovery.NewDiscoveryClientForConfig(env.Config)
	assert.NilError(t, err)

	serverInfo, err := dc.ServerVersion()
	assert.NilError(t, err)

	serverVersion, err := version.ParseGeneric(serverInfo.GitVersion)
	assert.NilError(t, err)

	if serverVersion.LessThan(expectedVersion) {
		t.Log("Kubernetes version", serverVersion, "is before", expectedVersion)
		t.SkipNow()
	}

	return cc
}

// Kubernetes2 is the same as [Kubernetes] but also returns a copy of the client
// configuration.
func Kubernetes2(t TestingT) (*rest.Config, client.Client) {
	t.Helper()
	env, cc := kubernetes3(t)
	return rest.CopyConfig(env.Config), cc
}

func kubernetes3(t TestingT) (*envtest.Environment, client.Client) {
	t.Helper()

	if !envtestVarsSet {
		t.SkipNow()
	}

	kubernetes.Lock()
	defer kubernetes.Unlock()

	// Skip any remaining tests after the environment fails to start once.
	// The test that tried to start the environment has reported the error.
	if kubernetes.err != nil {
		t.SkipNow()
	}

	if kubernetes.env == nil {
		// Get the current call stack, minus the closure below.
		frames := func() *goruntime.Frames {
			var pcs [5]uintptr
			n := goruntime.Callers(2, pcs[:])
			return goruntime.CallersFrames(pcs[0:n])
		}()

		// Calculate the project directory as reported by [goruntime.CallersFrames].
		frame, ok := frames.Next()
		self := frame.File
		root := Value(filepath.EvalSymlinks(strings.TrimSuffix(self,
			filepath.Join("internal", "testing", "require", "kubernetes.go"))))

		// Find the first caller that is not in this file.
		for ok && frame.File == self {
			frame, ok = frames.Next()
		}
		caller := Value(filepath.EvalSymlinks(frame.File))

		// Calculate the project directory path relative to the caller.
		base := Value(filepath.Rel(filepath.Dir(caller), root))

		// Calculate the snapshotter module directory path relative to the project directory.
		// Ignore any "vendor" directory by explicitly passing "-mod=readonly" https://go.dev/ref/mod#build-commands
		var snapshotter string
		if pkgs, err := packages.Load(
			&packages.Config{BuildFlags: []string{"-mod=readonly"}, Mode: packages.NeedModule},
			"github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1",
		); assert.Check(t,
			err == nil && len(pkgs) > 0 && pkgs[0].Module != nil, "unable to load package: %v\n%#v", err, pkgs,
		) {
			mod := pkgs[0].Module
			assert.Assert(t, mod.Dir != "" && mod.Error == nil, "expected module in cache\n%#v", mod)

			snapshotter, err = filepath.Rel(root, mod.Dir)
			assert.NilError(t, err, "module directory: %q", mod.Dir)
		}

		env := EnvTest(t, envtest.CRDInstallOptions{
			ErrorIfPathMissing: true,
			Paths: []string{
				filepath.Join(base, "config", "crd", "bases"),
				filepath.Join(base, snapshotter, "config", "crd"),
			},
			Scheme: runtime.Scheme,
		})

		// There are multiple components in an environment; stop them all when any fail to start.
		// Keep the error so other tests know not to try again.
		_, kubernetes.err = env.Start()
		if kubernetes.err != nil {
			assert.Check(t, env.Stop())
			assert.NilError(t, kubernetes.err)
		}

		kubernetes.env = env
	}

	kubernetes.count++

	t.Cleanup(func() {
		kubernetes.Lock()
		defer kubernetes.Unlock()

		kubernetes.count--

		if kubernetes.count == 0 {
			assert.Check(t, kubernetes.env.Stop())
			kubernetes.env = nil
			kubernetes.err = nil
		}
	})

	cc, err := client.New(kubernetes.env.Config, client.Options{
		Scheme: kubernetes.env.Scheme,
	})
	assert.NilError(t, err)

	return kubernetes.env, cc
}

// Namespace creates a random namespace that is deleted by t.Cleanup. It calls
// t.Fatal when creation fails. The caller may delete the namespace at any time.
func Namespace(t TestingT, cc client.Client) *corev1.Namespace {
	t.Helper()

	// Remove / that shows up when running a sub-test
	// TestSomeThing/test_some_specific_thing
	name, _, _ := strings.Cut(t.Name(), "/")

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = map[string]string{"postgres-operator-test": name}

	ctx := context.Background()
	assert.NilError(t, cc.Create(ctx, ns))

	t.Cleanup(func() {
		assert.Check(t, client.IgnoreNotFound(cc.Delete(ctx, ns)))
	})

	return ns
}
