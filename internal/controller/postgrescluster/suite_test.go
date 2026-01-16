// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
)

var suite struct {
	Client client.Client
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	suite.Client = require.Kubernetes(GinkgoT())

	logging.SetLogSink(logging.Logrus(GinkgoWriter, "test", 1, 1))
	log.SetLogger(logging.FromContext(context.Background()))
})

var _ = AfterSuite(func() {

})
