// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"

	// Google Kubernetes Engine / Google Cloud Platform authentication provider
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/logging"
)

var suite struct {
	Client client.Client
	Config *rest.Config

	Environment   *envtest.Environment
	ServerVersion *version.Version

	Manager manager.Manager
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" && !strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		Skip("skipping")
	}

	logging.SetLogSink(logging.Logrus(GinkgoWriter, "test", 1, 1))
	log.SetLogger(logging.FromContext(context.Background()))

	By("bootstrapping test environment")
	suite.Environment = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "..", "hack", "tools", "external-snapshotter", "client", "config", "crd"),
		},
	}

	_, err := suite.Environment.Start()
	Expect(err).ToNot(HaveOccurred())

	DeferCleanup(suite.Environment.Stop)

	suite.Config = suite.Environment.Config
	suite.Client, err = client.New(suite.Config, client.Options{Scheme: runtime.Scheme})
	Expect(err).ToNot(HaveOccurred())

	dc, err := discovery.NewDiscoveryClientForConfig(suite.Config)
	Expect(err).ToNot(HaveOccurred())

	server, err := dc.ServerVersion()
	Expect(err).ToNot(HaveOccurred())

	suite.ServerVersion, err = version.ParseGeneric(server.GitVersion)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {

})
