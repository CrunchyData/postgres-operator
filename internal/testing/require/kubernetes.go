// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
)

// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest#pkg-constants
var envtestVarsSet = os.Getenv("KUBEBUILDER_ASSETS") != "" ||
	strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true")

// EnvTest returns an unstarted Environment with crds. It calls t.Skip when
// the "KUBEBUILDER_ASSETS" and "USE_EXISTING_CLUSTER" environment variables
// are unset.
func EnvTest(t testing.TB, crds envtest.CRDInstallOptions) *envtest.Environment {
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
func Kubernetes(t testing.TB) client.Client {
	t.Helper()
	_, cc := kubernetes3(t)
	return cc
}

// Kubernetes2 is the same as [Kubernetes] but also returns a copy of the client
// configuration.
func Kubernetes2(t testing.TB) (*rest.Config, client.Client) {
	t.Helper()
	env, cc := kubernetes3(t)
	return rest.CopyConfig(env.Config), cc
}

func kubernetes3(t testing.TB) (*envtest.Environment, client.Client) {
	t.Helper()

	if !envtestVarsSet {
		t.SkipNow()
	}

	frames := func() *goruntime.Frames {
		var pcs [5]uintptr
		n := goruntime.Callers(2, pcs[:])
		return goruntime.CallersFrames(pcs[0:n])
	}()

	// Calculate the project directory as reported by [goruntime.CallersFrames].
	frame, ok := frames.Next()
	self := frame.File
	root := strings.TrimSuffix(self,
		filepath.Join("internal", "testing", "require", "kubernetes.go"))

	// Find the first caller that is not in this file.
	for ok && frame.File == self {
		frame, ok = frames.Next()
	}
	caller := frame.File

	// Calculate the project directory path relative to the caller.
	base, err := filepath.Rel(filepath.Dir(caller), root)
	assert.NilError(t, err)

	kubernetes.Lock()
	defer kubernetes.Unlock()

	if kubernetes.env == nil {
		env := EnvTest(t, envtest.CRDInstallOptions{
			ErrorIfPathMissing: true,
			Paths: []string{
				filepath.Join(base, "config", "crd", "bases"),
				filepath.Join(base, "hack", "tools", "external-snapshotter", "client", "config", "crd"),
			},
			Scheme: runtime.Scheme,
		})

		_, err := env.Start()
		assert.NilError(t, err)

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
func Namespace(t testing.TB, cc client.Client) *corev1.Namespace {
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
