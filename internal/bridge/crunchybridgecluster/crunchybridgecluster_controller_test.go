// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package crunchybridgecluster

import (
	"context"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/bridge"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var testTeamId = "5678"
var testApiKey = "9012"

func TestReconcileBridgeConnectionSecret(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}

	ns := setupNamespace(t, tClient).Name
	cluster := testCluster()
	cluster.Namespace = ns

	t.Run("Failure", func(t *testing.T) {
		key, team, err := reconciler.reconcileBridgeConnectionSecret(ctx, cluster)
		assert.Equal(t, key, "")
		assert.Equal(t, team, "")
		assert.Check(t, err != nil)
		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, readyCondition.Reason, "SecretInvalid")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"The condition of the cluster is unknown because the secret is invalid:"))
		}
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, upgradingCondition.Reason, "SecretInvalid")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"The condition of the upgrade(s) is unknown because the secret is invalid:"))
		}
	})

	t.Run("ValidSecretFound", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "crunchy-bridge-api-key",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"key":  []byte(`asdf`),
				"team": []byte(`jkl;`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		key, team, err := reconciler.reconcileBridgeConnectionSecret(ctx, cluster)
		assert.Equal(t, key, "asdf")
		assert.Equal(t, team, "jkl;")
		assert.NilError(t, err)
	})
}

func TestHandleDuplicateClusterName(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	clusterInBridge := testClusterApiResource()
	clusterInBridge.ClusterName = "bridge-cluster-1" // originally "hippo-cluster"
	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey:   testApiKey,
			TeamId:   testTeamId,
			Clusters: []*bridge.ClusterApiResource{clusterInBridge},
		}
	}

	ns := setupNamespace(t, tClient).Name

	t.Run("FailureToListClusters", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns

		controllerResult, err := reconciler.handleDuplicateClusterName(ctx, "bad_api_key", testTeamId, cluster)
		assert.Check(t, err != nil)
		assert.Equal(t, *controllerResult, ctrl.Result{})
		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, readyCondition.Reason, "UnknownClusterState")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"Issue listing existing clusters in Bridge:"))
		}
	})

	t.Run("NoDuplicateFound", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns

		controllerResult, err := reconciler.handleDuplicateClusterName(ctx, testApiKey, testTeamId, cluster)
		assert.NilError(t, err)
		assert.Check(t, controllerResult == nil)
	})

	t.Run("DuplicateFoundAdoptionAnnotationNotPresent", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Spec.ClusterName = "bridge-cluster-1" // originally "hippo-cluster"

		controllerResult, err := reconciler.handleDuplicateClusterName(ctx, testApiKey, testTeamId, cluster)
		assert.NilError(t, err)
		assert.Equal(t, *controllerResult, ctrl.Result{})
		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, readyCondition.Reason, "DuplicateClusterName")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"A cluster with the same name already exists for this team (Team ID: "))
		}
	})

	t.Run("DuplicateFoundAdoptionAnnotationPresent", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Spec.ClusterName = "bridge-cluster-1" // originally "hippo-cluster"
		cluster.Annotations = map[string]string{}
		cluster.Annotations[naming.CrunchyBridgeClusterAdoptionAnnotation] = "1234"

		controllerResult, err := reconciler.handleDuplicateClusterName(ctx, testApiKey, testTeamId, cluster)
		assert.NilError(t, err)
		assert.Equal(t, *controllerResult, ctrl.Result{Requeue: true})
		assert.Equal(t, cluster.Status.ID, "1234")
	})
}

func TestHandleCreateCluster(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey:   testApiKey,
			TeamId:   testTeamId,
			Clusters: []*bridge.ClusterApiResource{},
		}
	}

	t.Run("SuccessfulCreate", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns

		controllerResult := reconciler.handleCreateCluster(ctx, testApiKey, testTeamId, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		assert.Equal(t, cluster.Status.ID, "0")

		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, readyCondition.Reason, "UnknownClusterState")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"The condition of the cluster is unknown."))
		}

		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, upgradingCondition.Reason, "UnknownUpgradeState")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"The condition of the upgrade(s) is unknown."))
		}
	})

	t.Run("UnsuccessfulCreate", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns

		controllerResult := reconciler.handleCreateCluster(ctx, "bad_api_key", testTeamId, cluster)
		assert.Equal(t, controllerResult, ctrl.Result{})
		assert.Equal(t, cluster.Status.ID, "")

		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, readyCondition.Reason, "ClusterInvalid")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"Cannot create from spec:"))
		}

		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		assert.Check(t, upgradingCondition == nil)
	})
}

func TestHandleGetCluster(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name
	firstClusterInBridge := testClusterApiResource()
	secondClusterInBridge := testClusterApiResource()
	secondClusterInBridge.ID = "2345"                     // originally "1234"
	secondClusterInBridge.ClusterName = "hippo-cluster-2" // originally "hippo-cluster"

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey:   testApiKey,
			TeamId:   testTeamId,
			Clusters: []*bridge.ClusterApiResource{firstClusterInBridge, secondClusterInBridge},
		}
	}

	t.Run("SuccessfulGet", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"

		err := reconciler.handleGetCluster(ctx, testApiKey, cluster)
		assert.NilError(t, err)
		assert.Equal(t, cluster.Status.ClusterName, firstClusterInBridge.ClusterName)
		assert.Equal(t, cluster.Status.Host, firstClusterInBridge.Host)
		assert.Equal(t, cluster.Status.ID, firstClusterInBridge.ID)
		assert.Equal(t, cluster.Status.IsHA, firstClusterInBridge.IsHA)
		assert.Equal(t, cluster.Status.IsProtected, firstClusterInBridge.IsProtected)
		assert.Equal(t, cluster.Status.MajorVersion, firstClusterInBridge.MajorVersion)
		assert.Equal(t, cluster.Status.Plan, firstClusterInBridge.Plan)
		assert.Equal(t, *cluster.Status.Storage, *bridge.FromGibibytes(firstClusterInBridge.Storage))
	})

	t.Run("UnsuccessfulGet", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "bad_cluster_id"

		err := reconciler.handleGetCluster(ctx, testApiKey, cluster)
		assert.Check(t, err != nil)

		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, readyCondition.Reason, "UnknownClusterState")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"Issue getting cluster information from Bridge:"))
		}
	})
}

func TestHandleGetClusterStatus(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name
	readyClusterId := "1234"
	creatingClusterId := "7890"
	readyClusterStatusInBridge := testClusterStatusApiResource(readyClusterId)
	creatingClusterStatusInBridge := testClusterStatusApiResource(creatingClusterId)
	creatingClusterStatusInBridge.State = "creating" // originally "ready"

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey: testApiKey,
			TeamId: testTeamId,
			ClusterStatuses: map[string]*bridge.ClusterStatusApiResource{
				readyClusterId:    readyClusterStatusInBridge,
				creatingClusterId: creatingClusterStatusInBridge,
			},
		}
	}

	t.Run("SuccessReadyState", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = readyClusterId

		err := reconciler.handleGetClusterStatus(ctx, testApiKey, cluster)
		assert.NilError(t, err)
		assert.Equal(t, cluster.Status.State, "ready")
		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, readyCondition.Reason, "ready")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"Bridge cluster state is ready"))
		}
	})

	t.Run("SuccessNonReadyState", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = creatingClusterId

		err := reconciler.handleGetClusterStatus(ctx, testApiKey, cluster)
		assert.NilError(t, err)
		assert.Equal(t, cluster.Status.State, "creating")
		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, readyCondition.Reason, "creating")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"Bridge cluster state is creating"))
		}
	})

	t.Run("UnsuccessfulGet", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = creatingClusterId

		err := reconciler.handleGetClusterStatus(ctx, "bad_api_key", cluster)
		assert.Check(t, err != nil)
		assert.Equal(t, cluster.Status.State, "unknown")
		readyCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionReady)
		if assert.Check(t, readyCondition != nil) {
			assert.Equal(t, readyCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, readyCondition.Reason, "UnknownClusterState")
			assert.Check(t, cmp.Contains(readyCondition.Message,
				"Issue getting cluster status from Bridge:"))
		}
	})
}

func TestHandleGetClusterUpgrade(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name
	upgradingClusterId := "1234"
	notUpgradingClusterId := "7890"
	upgradingClusterUpgradeInBridge := testClusterUpgradeApiResource(upgradingClusterId)
	notUpgradingClusterUpgradeInBridge := testClusterUpgradeApiResource(notUpgradingClusterId)
	notUpgradingClusterUpgradeInBridge.Operations = []*v1beta1.UpgradeOperation{}

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey: testApiKey,
			TeamId: testTeamId,
			ClusterUpgrades: map[string]*bridge.ClusterUpgradeApiResource{
				upgradingClusterId:    upgradingClusterUpgradeInBridge,
				notUpgradingClusterId: notUpgradingClusterUpgradeInBridge,
			},
		}
	}

	t.Run("SuccessUpgrading", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = upgradingClusterId

		err := reconciler.handleGetClusterUpgrade(ctx, testApiKey, cluster)
		assert.NilError(t, err)
		assert.Equal(t, *cluster.Status.OngoingUpgrade[0], v1beta1.UpgradeOperation{
			Flavor:       "resize",
			StartingFrom: "",
			State:        "in_progress",
		})
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "resize")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Performing an upgrade of type resize with a state of in_progress."))
		}
	})

	t.Run("SuccessNotUpgrading", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = notUpgradingClusterId

		err := reconciler.handleGetClusterUpgrade(ctx, testApiKey, cluster)
		assert.NilError(t, err)
		assert.Equal(t, len(cluster.Status.OngoingUpgrade), 0)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, upgradingCondition.Reason, "NoUpgradesInProgress")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"No upgrades being performed"))
		}
	})

	t.Run("UnsuccessfulGet", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = notUpgradingClusterId

		err := reconciler.handleGetClusterUpgrade(ctx, "bad_api_key", cluster)
		assert.Check(t, err != nil)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionUnknown)
			assert.Equal(t, upgradingCondition.Reason, "UnknownUpgradeState")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Issue getting cluster upgrade from Bridge:"))
		}
	})
}

func TestHandleUpgrade(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name
	clusterInBridge := testClusterApiResource()

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey:   testApiKey,
			TeamId:   testTeamId,
			Clusters: []*bridge.ClusterApiResource{clusterInBridge},
		}
	}

	t.Run("UpgradePlan", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.Plan = "standard-16" // originally "standard-8"

		controllerResult := reconciler.handleUpgrade(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "maintenance")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Performing an upgrade of type maintenance with a state of in_progress."))
			assert.Equal(t, *cluster.Status.OngoingUpgrade[0], v1beta1.UpgradeOperation{
				Flavor:       "maintenance",
				StartingFrom: "",
				State:        "in_progress",
			})
		}
	})

	t.Run("UpgradePostgres", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.PostgresVersion = 16 // originally "15"

		controllerResult := reconciler.handleUpgrade(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "major_version_upgrade")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Performing an upgrade of type major_version_upgrade with a state of in_progress."))
			assert.Equal(t, *cluster.Status.OngoingUpgrade[0], v1beta1.UpgradeOperation{
				Flavor:       "major_version_upgrade",
				StartingFrom: "",
				State:        "in_progress",
			})
		}
	})

	t.Run("UpgradeStorage", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.Storage = resource.MustParse("15Gi") // originally "10Gi"

		controllerResult := reconciler.handleUpgrade(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "resize")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Performing an upgrade of type resize with a state of in_progress."))
			assert.Equal(t, *cluster.Status.OngoingUpgrade[0], v1beta1.UpgradeOperation{
				Flavor:       "resize",
				StartingFrom: "",
				State:        "in_progress",
			})
		}
	})

	t.Run("UpgradeFailure", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.Storage = resource.MustParse("15Gi") // originally "10Gi"

		controllerResult := reconciler.handleUpgrade(ctx, "bad_api_key", cluster)
		assert.Equal(t, controllerResult, ctrl.Result{})
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, upgradingCondition.Reason, "UpgradeError")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Error performing an upgrade: boom"))
		}
	})
}

func TestHandleUpgradeHA(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name
	clusterInBridgeWithHaDisabled := testClusterApiResource()
	clusterInBridgeWithHaEnabled := testClusterApiResource()
	clusterInBridgeWithHaEnabled.ID = "2345"                  // originally "1234"
	clusterInBridgeWithHaEnabled.IsHA = initialize.Bool(true) // originally "false"

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey: testApiKey,
			TeamId: testTeamId,
			Clusters: []*bridge.ClusterApiResource{clusterInBridgeWithHaDisabled,
				clusterInBridgeWithHaEnabled},
		}
	}

	t.Run("EnableHA", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.IsHA = true // originally "false"

		controllerResult := reconciler.handleUpgradeHA(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "ha_change")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Performing an upgrade of type ha_change with a state of enabling_ha."))
			assert.Equal(t, *cluster.Status.OngoingUpgrade[0], v1beta1.UpgradeOperation{
				Flavor:       "ha_change",
				StartingFrom: "",
				State:        "enabling_ha",
			})
		}
	})

	t.Run("DisableHA", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "2345"

		controllerResult := reconciler.handleUpgradeHA(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "ha_change")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Performing an upgrade of type ha_change with a state of disabling_ha."))
			assert.Equal(t, *cluster.Status.OngoingUpgrade[0], v1beta1.UpgradeOperation{
				Flavor:       "ha_change",
				StartingFrom: "",
				State:        "disabling_ha",
			})
		}
	})

	t.Run("UpgradeFailure", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"

		controllerResult := reconciler.handleUpgradeHA(ctx, "bad_api_key", cluster)
		assert.Equal(t, controllerResult, ctrl.Result{})
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, upgradingCondition.Reason, "UpgradeError")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"Error performing an HA upgrade: boom"))
		}
	})
}

func TestHandleUpdate(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient).Name
	clusterInBridge := testClusterApiResource()

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}
	reconciler.NewClient = func() bridge.ClientInterface {
		return &TestBridgeClient{
			ApiKey:   testApiKey,
			TeamId:   testTeamId,
			Clusters: []*bridge.ClusterApiResource{clusterInBridge},
		}
	}

	t.Run("UpdateName", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.ClusterName = "new-cluster-name" // originally "hippo-cluster"

		controllerResult := reconciler.handleUpdate(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "ClusterUpgrade")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"An upgrade is occurring, the clusters name is new-cluster-name and the cluster is protected is false."))
		}
		assert.Equal(t, cluster.Status.ClusterName, "new-cluster-name")
	})

	t.Run("UpdateIsProtected", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.IsProtected = true // originally "false"

		controllerResult := reconciler.handleUpdate(ctx, testApiKey, cluster)
		assert.Equal(t, controllerResult.RequeueAfter, 3*time.Minute)
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionTrue)
			assert.Equal(t, upgradingCondition.Reason, "ClusterUpgrade")
			assert.Check(t, cmp.Contains(upgradingCondition.Message,
				"An upgrade is occurring, the clusters name is hippo-cluster and the cluster is protected is true."))
		}
		assert.Equal(t, *cluster.Status.IsProtected, true)
	})

	t.Run("UpgradeFailure", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns
		cluster.Status.ID = "1234"
		cluster.Spec.IsProtected = true // originally "false"

		controllerResult := reconciler.handleUpdate(ctx, "bad_api_key", cluster)
		assert.Equal(t, controllerResult, ctrl.Result{})
		upgradingCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1beta1.ConditionUpgrading)
		if assert.Check(t, upgradingCondition != nil) {
			assert.Equal(t, upgradingCondition.Status, metav1.ConditionFalse)
			assert.Equal(t, upgradingCondition.Reason, "UpgradeError")
			assert.Check(t, cmp.Contains(upgradingCondition.Message, "Error performing an upgrade: boom"))
		}
	})
}

func TestGetSecretKeys(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}

	ns := setupNamespace(t, tClient).Name
	cluster := testCluster()
	cluster.Namespace = ns

	t.Run("NoSecret", func(t *testing.T) {
		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Equal(t, apiKey, "")
		assert.Equal(t, team, "")
		assert.ErrorContains(t, err, "secrets \"crunchy-bridge-api-key\" not found")
	})

	t.Run("SecretMissingApiKey", func(t *testing.T) {
		cluster.Spec.Secret = "secret-missing-api-key" // originally "crunchy-bridge-api-key"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-missing-api-key",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"team": []byte(`jkl;`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Equal(t, apiKey, "")
		assert.Equal(t, team, "")
		assert.ErrorContains(t, err, "error handling secret; expected to find a key and a team: found key false, found team true")

		assert.NilError(t, tClient.Delete(ctx, secret))
	})

	t.Run("SecretMissingTeamId", func(t *testing.T) {
		cluster.Spec.Secret = "secret-missing-team-id"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-missing-team-id",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"key": []byte(`asdf`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Equal(t, apiKey, "")
		assert.Equal(t, team, "")
		assert.ErrorContains(t, err, "error handling secret; expected to find a key and a team: found key true, found team false")
	})

	t.Run("GoodSecret", func(t *testing.T) {
		cluster.Spec.Secret = "crunchy-bridge-api-key"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "crunchy-bridge-api-key",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"key":  []byte(`asdf`),
				"team": []byte(`jkl;`),
			},
		}
		assert.NilError(t, tClient.Create(ctx, secret))

		apiKey, team, err := reconciler.GetSecretKeys(ctx, cluster)
		assert.Equal(t, apiKey, "asdf")
		assert.Equal(t, team, "jkl;")
		assert.NilError(t, err)
	})
}

func TestDeleteControlled(t *testing.T) {
	ctx := context.Background()
	tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, tClient)
	reconciler := &CrunchyBridgeClusterReconciler{
		Client: tClient,
		Owner:  "crunchybridgecluster-controller",
	}

	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.Name = strings.ToLower(t.Name()) // originally "hippo-cr"
	assert.NilError(t, tClient.Create(ctx, cluster))

	t.Run("NotControlled", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "solo"

		assert.NilError(t, tClient.Create(ctx, secret))

		// No-op when there's no ownership
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))
		assert.NilError(t, tClient.Get(ctx, client.ObjectKeyFromObject(secret), secret))
	})

	t.Run("Controlled", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "controlled"

		assert.NilError(t, reconciler.setControllerReference(cluster, secret))
		assert.NilError(t, tClient.Create(ctx, secret))

		// Deletes when controlled by cluster.
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))

		err := tClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %#v", err)
	})
}
