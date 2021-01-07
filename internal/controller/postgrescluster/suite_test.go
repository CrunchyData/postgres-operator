// +build envtest

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

var suite struct {
	Client client.Client
	Config *rest.Config
	Scheme *runtime.Scheme

	Environment *envtest.Environment
	Manager     manager.Manager
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logging.SetLogFunc(1, logging.Logrus(GinkgoWriter, "test", 1))
	log.SetLogger(logging.FromContext(context.Background()))

	By("bootstrapping test environment")
	suite.Environment = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
	}

	suite.Scheme = runtime.NewScheme()
	Expect(scheme.AddToScheme(suite.Scheme)).To(Succeed())
	Expect(v1alpha1.AddToScheme(suite.Scheme)).To(Succeed())

	_, err := suite.Environment.Start()
	Expect(err).ToNot(HaveOccurred())

	suite.Config = suite.Environment.Config
	suite.Client, err = client.New(suite.Config, client.Options{Scheme: suite.Scheme})
	Expect(err).ToNot(HaveOccurred())
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	Expect(suite.Environment.Stop()).To(Succeed())
})
