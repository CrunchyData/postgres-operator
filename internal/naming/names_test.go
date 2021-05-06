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

package naming

import (
	"fmt"
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
	// - https://docs.k8s.io/reference/kubernetes-api/workloads-resources/pod-v1/

	names := sets.NewString()
	for _, name := range []string{
		ContainerDatabase,
		ContainerDatabasePGDATAInit,
		ContainerNSSWrapperInit,
		ContainerPGBouncer,
		ContainerPGBouncerConfig,
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

	type test struct {
		name  string
		value metav1.ObjectMeta
	}

	testUniqueAndValid := func(t *testing.T, tests []test) {
		names := sets.NewString()
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.value.Namespace, cluster.Namespace)
				assert.Assert(t, tt.value.Name != cluster.Name, "may collide")
				assert.Assert(t, !names.Has(tt.value.Name), "%q defined already", tt.value.Name)
				assert.Assert(t, nil == validation.IsDNS1123Label(tt.value.Name))
				names.Insert(tt.value.Name)
			})
		}
	}

	t.Run("ConfigMaps", func(t *testing.T) {
		testUniqueAndValid(t, []test{
			{"ClusterConfigMap", ClusterConfigMap(cluster)},
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
			{"ClusterPGBouncer", PGBackRestBackupJob(cluster)},
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
		testUniqueAndValid(t, []test{
			{"ClusterPGBouncer", ClusterPGBouncer(cluster)},
			{"PostgresUserSecret", PostgresUserSecret(cluster)},
			{"PostgresTLSSecret", PostgresTLSSecret(cluster)},
			{"ReplicationClientCertSecret", ReplicationClientCertSecret(cluster)},
			{"PGBackRestSSHSecret", PGBackRestSSHSecret(cluster)},
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
			{"ClusterPodService", ClusterPodService(cluster)},
			{"ClusterPrimaryService", ClusterPrimaryService(cluster)},
			// Patroni can use Endpoints which relate directly to a Service.
			{"PatroniDistributedConfiguration", PatroniDistributedConfiguration(cluster)},
			{"PatroniLeaderEndpoints", PatroniLeaderEndpoints(cluster)},
			{"PatroniTrigger", PatroniTrigger(cluster)},
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
			{"InstancePGDataVolume", InstancePGDataVolume(instance)},
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

func TestGetPGDATADirectory(t *testing.T) {
	postgresVersion := 13

	cluster := &v1beta1.PostgresCluster{
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: postgresVersion,
		},
	}

	assert.Equal(t, GetPGDATADirectory(cluster), fmt.Sprintf("/pgdata/pg%d", postgresVersion))
}
