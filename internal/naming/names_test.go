// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestAsObjectKey(t *testing.T) {
	assert.Equal(t, AsObjectKey(
		metav1.ObjectMeta{Namespace: "ns1", Name: "thing"}),
		client.ObjectKey{Namespace: "ns1", Name: "thing"})
}

func TestContainerNamesUniqueAndValid(t *testing.T) {
	// Container names have to be unique within a Pod. The number of containers
	// we deploy should be few enough that we can name them uniquely across all
	// pods.
	// - https://docs.k8s.io/reference/kubernetes-api/workload-resources/pod-v1/

	names := sets.NewString()
	for _, name := range []string{
		ContainerDatabase,
		ContainerNSSWrapperInit,
		ContainerPGAdmin,
		ContainerPGAdminStartup,
		ContainerPGBackRestConfig,
		ContainerPGBackRestLogDirInit,
		ContainerPGBouncer,
		ContainerPGBouncerConfig,
		ContainerPostgresStartup,
		ContainerPGMonitorExporter,
	} {
		assert.Assert(t, !names.Has(name), "%q defined already", name)
		assert.Assert(t, nil == validation.IsDNS1123Label(name))
		names.Insert(name)
	}
}

func TestClusterNamesUniqueAndValid(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1", Name: "pg0",
		},
	}
	repoName := "hippo-repo"
	instanceSet := &v1beta1.PostgresInstanceSetSpec{
		Name: "set-1",
	}

	type test struct {
		name  string
		value metav1.ObjectMeta
	}

	testUniqueAndValid := func(t *testing.T, tests []test) sets.Set[string] {
		names := sets.Set[string]{}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.value.Namespace, cluster.Namespace)
				assert.Assert(t, tt.value.Name != cluster.Name, "may collide")
				assert.Assert(t, !names.Has(tt.value.Name), "%q defined already", tt.value.Name)
				assert.Assert(t, nil == validation.IsDNS1123Label(tt.value.Name))
				names.Insert(tt.value.Name)
			})
		}
		return names
	}

	t.Run("ConfigMaps", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterConfigMap", ClusterConfigMap(cluster)},
			{"ClusterPGAdmin", ClusterPGAdmin(cluster)},
			{"ClusterPGBouncer", ClusterPGBouncer(cluster)},
			{"PatroniDistributedConfiguration", PatroniDistributedConfiguration(cluster)},
			{"PatroniLeaderConfigMap", PatroniLeaderConfigMap(cluster)},
			{"PatroniTrigger", PatroniTrigger(cluster)},
			{"PGBackRestConfig", PGBackRestConfig(cluster)},
			{"PGBackRestSSHConfig", PGBackRestSSHConfig(cluster)},
		})
	})

	t.Run("CronJobs", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"PGBackRestCronJon", PGBackRestCronJob(cluster, "full", "repo1")},
			{"PGBackRestCronJon", PGBackRestCronJob(cluster, "incr", "repo2")},
			{"PGBackRestCronJon", PGBackRestCronJob(cluster, "diff", "repo3")},
			{"PGBackRestCronJon", PGBackRestCronJob(cluster, "full", "repo4")},
		})
	})

	t.Run("Deployments", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterPGBouncer", ClusterPGBouncer(cluster)},
		})
	})

	t.Run("Jobs", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"PGBackRestBackupJob", PGBackRestBackupJob(cluster)},
			{"PGBackRestRestoreJob", PGBackRestRestoreJob(cluster)},
		})
	})

	t.Run("PodDisruptionBudgets", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"InstanceSetPDB", InstanceSet(cluster, instanceSet)},
			{"PGBouncerPDB", ClusterPGBouncer(cluster)},
		})
	})

	t.Run("RoleBindings", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterInstanceRBAC", ClusterInstanceRBAC(cluster)},
			{"PGBackRestRBAC", PGBackRestRBAC(cluster)},
		})
	})

	t.Run("Roles", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterInstanceRBAC", ClusterInstanceRBAC(cluster)},
			{"PGBackRestRBAC", PGBackRestRBAC(cluster)},
		})
	})

	t.Run("Secrets", func(t *testing.T) {
		names := testUniqueAndValid(t, []test{
			{"ClusterPGBouncer", ClusterPGBouncer(cluster)},
			{"DeprecatedPostgresUserSecret", DeprecatedPostgresUserSecret(cluster)},
			{"PostgresTLSSecret", PostgresTLSSecret(cluster)},
			{"ReplicationClientCertSecret", ReplicationClientCertSecret(cluster)},
			{"PGBackRestSSHSecret", PGBackRestSSHSecret(cluster)},
			{"MonitoringUserSecret", MonitoringUserSecret(cluster)},
		})

		// NOTE: This does not fail when a conflict is introduced. When adding a
		// Secret, be sure to compare it to the function below.
		t.Run("OperatorConfiguration", func(t *testing.T) {
			other := OperatorConfigurationSecret().Name
			assert.Assert(t, !names.Has(other), "%q defined already", other)
		})

		t.Run("PostgresUserSecret", func(t *testing.T) {
			value := PostgresUserSecret(cluster, "some-user")

			assert.Equal(t, value.Namespace, cluster.Namespace)
			assert.Assert(t, nil == validation.IsDNS1123Label(value.Name))

			prefix := PostgresUserSecret(cluster, "").Name
			for _, name := range sets.List(names) {
				assert.Assert(t, !strings.HasPrefix(name, prefix), "%q may collide", name)
			}
		})
	})

	t.Run("ServiceAccounts", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterInstanceRBAC", ClusterInstanceRBAC(cluster)},
			{"PGBackRestRBAC", PGBackRestRBAC(cluster)},
		})
	})

	t.Run("Services", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterPGBouncer", ClusterPGBouncer(cluster)},
			{"ClusterPGAdmin", ClusterPGAdmin(cluster)},
			{"ClusterPodService", ClusterPodService(cluster)},
			{"ClusterPrimaryService", ClusterPrimaryService(cluster)},
			{"ClusterReplicaService", ClusterReplicaService(cluster)},
			// Patroni can use Endpoints which relate directly to a Service.
			{"PatroniDistributedConfiguration", PatroniDistributedConfiguration(cluster)},
			{"PatroniLeaderEndpoints", PatroniLeaderEndpoints(cluster)},
			{"PatroniTrigger", PatroniTrigger(cluster)},
		})
	})

	t.Run("StatefulSets", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterPGAdmin", ClusterPGAdmin(cluster)},
		})
	})

	t.Run("Volumes", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterPGAdmin", ClusterPGAdmin(cluster)},
			{"PGBackRestRepoVolume", PGBackRestRepoVolume(cluster, repoName)},
		})
	})

	t.Run("VolumeSnapshots", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterVolumeSnapshot", ClusterVolumeSnapshot(cluster)},
		})
	})
}

func TestInstanceNamesUniqueAndValid(t *testing.T) {
	instance := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns", Name: "some-such",
		},
	}

	type test struct {
		name  string
		value metav1.ObjectMeta
	}

	t.Run("ConfigMaps", func(t *testing.T) {
		names := sets.NewString()
		for _, tt := range []test{
			{"InstanceConfigMap", InstanceConfigMap(instance)},
		} {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.value.Namespace, instance.Namespace)
				assert.Assert(t, tt.value.Name != instance.Name, "may collide")
				assert.Assert(t, !names.Has(tt.value.Name), "%q defined already", tt.value.Name)
				assert.Assert(t, nil == validation.IsDNS1123Label(tt.value.Name))
				names.Insert(tt.value.Name)
			})
		}
	})

	t.Run("PVCs", func(t *testing.T) {
		names := sets.NewString()
		for _, tt := range []test{
			{"InstancePostgresDataVolume", InstancePostgresDataVolume(instance)},
			{"InstancePostgresWALVolume", InstancePostgresWALVolume(instance)},
		} {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.value.Namespace, instance.Namespace)
				assert.Assert(t, tt.value.Name != instance.Name, "may collide")
				assert.Assert(t, !names.Has(tt.value.Name), "%q defined already", tt.value.Name)
				assert.Assert(t, nil == validation.IsDNS1123Label(tt.value.Name))
				names.Insert(tt.value.Name)
			})
		}
	})

	t.Run("Secrets", func(t *testing.T) {
		names := sets.NewString()
		for _, tt := range []test{
			{"InstanceCertificates", InstanceCertificates(instance)},
		} {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.value.Namespace, instance.Namespace)
				assert.Assert(t, tt.value.Name != instance.Name, "may collide")
				assert.Assert(t, !names.Has(tt.value.Name), "%q defined already", tt.value.Name)
				assert.Assert(t, nil == validation.IsDNS1123Label(tt.value.Name))
				names.Insert(tt.value.Name)
			})
		}
	})
}

func TestGenerateInstance(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1", Name: "pg0",
		},
	}
	set := &v1beta1.PostgresInstanceSetSpec{Name: "hippos"}

	instance := GenerateInstance(cluster, set)

	assert.Equal(t, cluster.Namespace, instance.Namespace)
	assert.Assert(t, strings.HasPrefix(instance.Name, cluster.Name+"-"+set.Name+"-"))
}

// TestGenerateStartupInstance ensures that a consistent ObjectMeta will be
// provided assuming the same cluster name and instance set name is passed
// into GenerateStartupInstance
func TestGenerateStartupInstance(t *testing.T) {
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1", Name: "pg0",
		},
	}
	set := &v1beta1.PostgresInstanceSetSpec{Name: "hippos"}

	instanceOne := GenerateStartupInstance(cluster, set)

	assert.Equal(t, cluster.Namespace, instanceOne.Namespace)
	assert.Assert(t, strings.HasPrefix(instanceOne.Name, cluster.Name+"-"+set.Name+"-"))

	instanceTwo := GenerateStartupInstance(cluster, set)
	assert.DeepEqual(t, instanceOne, instanceTwo)

}

func TestOperatorConfigurationSecret(t *testing.T) {
	t.Setenv("PGO_NAMESPACE", "cheese")

	value := OperatorConfigurationSecret()
	assert.Equal(t, value.Namespace, "cheese")
	assert.Assert(t, nil == validation.IsDNS1123Label(value.Name))
}

func TestPortNamesUniqueAndValid(t *testing.T) {
	// Port names have to be unique within a Pod. The number of ports we employ
	// should be few enough that we can name them uniquely across all pods.
	// - https://docs.k8s.io/reference/kubernetes-api/workload-resources/pod-v1/#ports

	names := sets.NewString()
	for _, name := range []string{
		PortExporter,
		PortPGAdmin,
		PortPGBouncer,
		PortPostgreSQL,
	} {
		assert.Assert(t, !names.Has(name), "%q defined already", name)
		assert.Assert(t, nil == validation.IsValidPortName(name))
		names.Insert(name)
	}
}
