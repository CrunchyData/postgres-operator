// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package upgradecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/kubernetes"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGenerateHeader(t *testing.T) {
	setupDeploymentID(t)
	ctx := context.Background()
	cfg, cc := require.Kubernetes2(t)

	discovery, err := kubernetes.NewDiscoveryRunner(cfg)
	assert.NilError(t, err)
	assert.NilError(t, discovery.Read(ctx))
	ctx = kubernetes.NewAPIContext(ctx, discovery)

	t.Setenv("PGO_INSTALLER", "test")
	t.Setenv("PGO_INSTALLER_ORIGIN", "test-origin")
	t.Setenv("PGO_NAMESPACE", require.Namespace(t, cc).Name)
	t.Setenv("BUILD_SOURCE", "developer")

	t.Run("error ensuring ID", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "patch error",
		}
		ctx, calls := setupLogCapture(ctx)

		res := generateHeader(ctx, fakeClientWithOptionalError, "1.2.3", "")
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: could not apply configmap`))
		assert.Equal(t, discovery.IsOpenShift(), res.IsOpenShift)
		assert.Equal(t, deploymentID, res.DeploymentID)
		pgoList := v1beta1.PostgresClusterList{}
		err := cc.List(ctx, &pgoList)
		assert.NilError(t, err)
		assert.Equal(t, len(pgoList.Items), res.PGOClustersTotal)
		bridgeList := v1beta1.CrunchyBridgeClusterList{}
		err = cc.List(ctx, &bridgeList)
		assert.NilError(t, err)
		assert.Equal(t, len(bridgeList.Items), res.BridgeClustersTotal)
		assert.Equal(t, "1.2.3", res.PGOVersion)
		assert.Equal(t, discovery.Version().String(), res.KubernetesEnv)
		assert.Equal(t, "test", res.PGOInstaller)
		assert.Equal(t, "test-origin", res.PGOInstallerOrigin)
		assert.Equal(t, "developer", res.BuildSource)
	})

	t.Run("error getting cluster count", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "list error",
		}
		ctx, calls := setupLogCapture(ctx)

		res := generateHeader(ctx, fakeClientWithOptionalError, "1.2.3", "")
		assert.Equal(t, len(*calls), 2)
		// Aggregating the logs since we cannot determine which call will be first
		callsAggregate := strings.Join(*calls, " ")
		assert.Assert(t, cmp.Contains(callsAggregate, `upgrade check issue: could not count postgres clusters`))
		assert.Assert(t, cmp.Contains(callsAggregate, `upgrade check issue: could not count bridge clusters`))
		assert.Equal(t, discovery.IsOpenShift(), res.IsOpenShift)
		assert.Equal(t, deploymentID, res.DeploymentID)
		assert.Equal(t, 0, res.PGOClustersTotal)
		assert.Equal(t, 0, res.BridgeClustersTotal)
		assert.Equal(t, "1.2.3", res.PGOVersion)
		assert.Equal(t, discovery.Version().String(), res.KubernetesEnv)
		assert.Equal(t, "test", res.PGOInstaller)
		assert.Equal(t, "test-origin", res.PGOInstallerOrigin)
		assert.Equal(t, "developer", res.BuildSource)
	})

	t.Run("success", func(t *testing.T) {
		ctx, calls := setupLogCapture(ctx)
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.TablespaceVolumes: true,
		}))
		ctx = feature.NewContext(ctx, gate)

		res := generateHeader(ctx, cc, "1.2.3", "")
		assert.Equal(t, len(*calls), 0)
		assert.Equal(t, discovery.IsOpenShift(), res.IsOpenShift)
		assert.Equal(t, deploymentID, res.DeploymentID)
		pgoList := v1beta1.PostgresClusterList{}
		err := cc.List(ctx, &pgoList)
		assert.NilError(t, err)
		assert.Equal(t, len(pgoList.Items), res.PGOClustersTotal)
		assert.Equal(t, "1.2.3", res.PGOVersion)
		assert.Equal(t, discovery.Version().String(), res.KubernetesEnv)
		assert.Check(t, strings.Contains(
			res.FeatureGatesEnabled,
			"TablespaceVolumes=true",
		))
		assert.Equal(t, "test", res.PGOInstaller)
		assert.Equal(t, "test-origin", res.PGOInstallerOrigin)
		assert.Equal(t, "developer", res.BuildSource)
	})
}

func TestEnsureID(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Setenv("PGO_NAMESPACE", require.Namespace(t, cc).Name)

	t.Run("success, no id set in mem or configmap", func(t *testing.T) {
		deploymentID = ""
		oldID := deploymentID
		ctx, calls := setupLogCapture(ctx)

		newID := ensureDeploymentID(ctx, cc)
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, newID != oldID)
		assert.Assert(t, newID == deploymentID)

		cm := &corev1.ConfigMap{}
		err := cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cm)
		assert.NilError(t, err)
		assert.Equal(t, newID, cm.Data["deployment_id"])
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("success, id set in mem, configmap created", func(t *testing.T) {
		oldID := setupDeploymentID(t)

		cm := &corev1.ConfigMap{}
		err := cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cm)
		assert.Error(t, err, `configmaps "pgo-upgrade-check" not found`)
		ctx, calls := setupLogCapture(ctx)

		newID := ensureDeploymentID(ctx, cc)
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, newID == oldID)
		assert.Assert(t, newID == deploymentID)

		err = cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cm)
		assert.NilError(t, err)
		assert.Assert(t, deploymentID == cm.Data["deployment_id"])

		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("success, id set in configmap, mem overwritten", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"deployment_id": string(uuid.NewUUID()),
			},
		}
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)

		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		oldID := setupDeploymentID(t)
		ctx, calls := setupLogCapture(ctx)
		newID := ensureDeploymentID(ctx, cc)
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, newID != oldID)
		assert.Assert(t, newID == deploymentID)
		assert.Assert(t, deploymentID == cmRetrieved.Data["deployment_id"])

		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("configmap failed, no namespace given", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"deployment_id": string(uuid.NewUUID()),
			},
		}
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)

		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		oldID := setupDeploymentID(t)
		ctx, calls := setupLogCapture(ctx)
		t.Setenv("PGO_NAMESPACE", "")

		newID := ensureDeploymentID(ctx, cc)
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: namespace not set`))
		assert.Assert(t, newID == oldID)
		assert.Assert(t, newID == deploymentID)
		assert.Assert(t, deploymentID != cmRetrieved.Data["deployment_id"])
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("configmap failed with not NotFound error, using preexisting ID", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "get error",
		}
		oldID := setupDeploymentID(t)
		ctx, calls := setupLogCapture(ctx)

		newID := ensureDeploymentID(ctx, fakeClientWithOptionalError)
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: error retrieving configmap`))
		assert.Assert(t, newID == oldID)
		assert.Assert(t, newID == deploymentID)

		cmRetrieved := &corev1.ConfigMap{}
		err := cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.Error(t, err, `configmaps "pgo-upgrade-check" not found`)
	})

	t.Run("configmap failed to create, using preexisting ID", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "patch error",
		}
		oldID := setupDeploymentID(t)

		ctx, calls := setupLogCapture(ctx)
		newID := ensureDeploymentID(ctx, fakeClientWithOptionalError)
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: could not apply configmap`))
		assert.Assert(t, newID == oldID)
		assert.Assert(t, newID == deploymentID)
	})
}

func TestManageUpgradeCheckConfigMap(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Setenv("PGO_NAMESPACE", require.Namespace(t, cc).Name)

	t.Run("no namespace given", func(t *testing.T) {
		ctx, calls := setupLogCapture(ctx)
		t.Setenv("PGO_NAMESPACE", "")

		returnedCM := manageUpgradeCheckConfigMap(ctx, cc, "current-id")
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: namespace not set`))
		assert.Assert(t, returnedCM.Data["deployment_id"] == "current-id")
	})

	t.Run("configmap not found, created", func(t *testing.T) {
		cmRetrieved := &corev1.ConfigMap{}
		err := cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.Error(t, err, `configmaps "pgo-upgrade-check" not found`)

		ctx, calls := setupLogCapture(ctx)
		returnedCM := manageUpgradeCheckConfigMap(ctx, cc, "current-id")

		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, returnedCM.Data["deployment_id"] == "current-id")
		err = cc.Delete(ctx, returnedCM)
		assert.NilError(t, err)
	})

	t.Run("configmap failed with not NotFound error", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "get error",
		}
		ctx, calls := setupLogCapture(ctx)

		returnedCM := manageUpgradeCheckConfigMap(ctx, fakeClientWithOptionalError,
			"current-id")
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: error retrieving configmap`))
		assert.Assert(t, returnedCM.Data["deployment_id"] == "current-id")
	})

	t.Run("no deployment id in configmap", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"wrong_field": string(uuid.NewUUID()),
			},
		}
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)

		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		ctx, calls := setupLogCapture(ctx)
		returnedCM := manageUpgradeCheckConfigMap(ctx, cc, "current-id")
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, returnedCM.Data["deployment_id"] == "current-id")
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("mangled deployment id", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"deploymentid": string(uuid.NewUUID())[1:],
			},
		}
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)

		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		ctx, calls := setupLogCapture(ctx)
		returnedCM := manageUpgradeCheckConfigMap(ctx, cc, "current-id")
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, returnedCM.Data["deployment_id"] == "current-id")
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("good configmap with good id", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"deployment_id": string(uuid.NewUUID()),
			},
		}
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)

		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(
			naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		ctx, calls := setupLogCapture(ctx)
		returnedCM := manageUpgradeCheckConfigMap(ctx, cc, "current-id")
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, returnedCM.Data["deployment-id"] != "current-id")
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("configmap failed to create", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "patch error",
		}

		ctx, calls := setupLogCapture(ctx)
		returnedCM := manageUpgradeCheckConfigMap(ctx, fakeClientWithOptionalError,
			"current-id")
		assert.Equal(t, len(*calls), 1)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: could not apply configmap`))
		assert.Assert(t, returnedCM.Data["deployment_id"] == "current-id")
	})
}

func TestApplyConfigMap(t *testing.T) {
	ctx := context.Background()
	cc := require.Kubernetes(t)
	t.Setenv("PGO_NAMESPACE", require.Namespace(t, cc).Name)

	t.Run("successful create", func(t *testing.T) {
		cmRetrieved := &corev1.ConfigMap{}
		err := cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.Error(t, err, `configmaps "pgo-upgrade-check" not found`)

		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"new_field": "new_value",
			},
		}
		cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		err = applyConfigMap(ctx, cc, cm, "test")
		assert.NilError(t, err)
		cmRetrieved = &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)
		assert.Equal(t, cm.Data["new_value"], cmRetrieved.Data["new_value"])
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("successful update", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"new_field": "old_value",
			},
		}
		cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)
		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		cm2 := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"new_field": "new_value",
			},
		}
		cm2.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		err = applyConfigMap(ctx, cc, cm2, "test")
		assert.NilError(t, err)
		cmRetrieved = &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)
		assert.Equal(t, cm.Data["new_value"], cmRetrieved.Data["new_value"])
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("successful nothing changed", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"new_field": "new_value",
			},
		}
		cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		err := cc.Create(ctx, cm)
		assert.NilError(t, err)
		cmRetrieved := &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)

		cm2 := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"new_field": "new_value",
			},
		}
		cm2.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		err = applyConfigMap(ctx, cc, cm2, "test")
		assert.NilError(t, err)
		cmRetrieved = &corev1.ConfigMap{}
		err = cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.NilError(t, err)
		assert.Equal(t, cm.Data["new_value"], cmRetrieved.Data["new_value"])
		err = cc.Delete(ctx, cm)
		assert.NilError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		cmRetrieved := &corev1.ConfigMap{}
		err := cc.Get(ctx, naming.AsObjectKey(naming.UpgradeCheckConfigMap()), cmRetrieved)
		assert.Error(t, err, `configmaps "pgo-upgrade-check" not found`)

		cm := &corev1.ConfigMap{
			ObjectMeta: naming.UpgradeCheckConfigMap(),
			Data: map[string]string{
				"new_field": "new_value",
			},
		}
		cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		fakeClientWithOptionalError := &fakeClientWithError{
			cc, "patch error",
		}

		err = applyConfigMap(ctx, fakeClientWithOptionalError, cm, "test")
		assert.Error(t, err, "patch error")
	})
}

func TestGetManagedClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		fakeClient := setupFakeClientWithPGOScheme(t, true)
		ctx, calls := setupLogCapture(ctx)
		count := getManagedClusters(ctx, fakeClient)
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, count == 2)
	})

	t.Run("list throw error", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			setupFakeClientWithPGOScheme(t, true), "list error",
		}
		ctx, calls := setupLogCapture(ctx)
		count := getManagedClusters(ctx, fakeClientWithOptionalError)
		assert.Assert(t, len(*calls) > 0)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: could not count postgres clusters`))
		assert.Assert(t, count == 0)
	})
}

func TestGetBridgeClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		fakeClient := setupFakeClientWithPGOScheme(t, true)
		ctx, calls := setupLogCapture(ctx)
		count := getBridgeClusters(ctx, fakeClient)
		assert.Equal(t, len(*calls), 0)
		assert.Assert(t, count == 2)
	})

	t.Run("list throw error", func(t *testing.T) {
		fakeClientWithOptionalError := &fakeClientWithError{
			setupFakeClientWithPGOScheme(t, true), "list error",
		}
		ctx, calls := setupLogCapture(ctx)
		count := getBridgeClusters(ctx, fakeClientWithOptionalError)
		assert.Assert(t, len(*calls) > 0)
		assert.Assert(t, cmp.Contains((*calls)[0], `upgrade check issue: could not count bridge clusters`))
		assert.Assert(t, count == 0)
	})
}

func TestAddHeader(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		req := &http.Request{
			Header: http.Header{},
		}
		versionString := "1.2.3"
		upgradeInfo := &clientUpgradeData{
			PGOVersion: versionString,
		}

		result := addHeader(req, upgradeInfo)
		header := result.Header[clientHeader]

		passedThroughData := &clientUpgradeData{}
		err := json.Unmarshal([]byte(header[0]), passedThroughData)
		assert.NilError(t, err)

		assert.Equal(t, passedThroughData.PGOVersion, "1.2.3")
		// Failure to list clusters results in 0 returned
		assert.Equal(t, passedThroughData.PGOClustersTotal, 0)
	})
}
