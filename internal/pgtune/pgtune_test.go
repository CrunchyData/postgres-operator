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

package pgtune

import (
	"testing"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func DefaultCluster() *v1beta1.PostgresCluster {
	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()
	cluster.Namespace = "some-namespace"
	cluster.Name = "cluster-name"
	cluster.Spec.PostgresVersion = 13
	cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{Name: "test"}}
	for i := range cluster.Spec.InstanceSets {
		cluster.Spec.InstanceSets[i].Default(i)
	}
	return cluster
}

func AssertConfigVal(data map[string]interface{}, key string, expectedVal string) bool {
	val, ok := data[key]
	return ok && val == expectedVal
}

func TestGetPGTuneConfigParameters(t *testing.T) {
	t.Parallel()
	cluster := DefaultCluster()
	for i := range cluster.Spec.InstanceSets {
		cluster.Spec.InstanceSets[i].Default(i)
		cluster.Spec.InstanceSets[i].Resources = v1.ResourceRequirements{Requests: v1.ResourceList{
			"memory": resource.MustParse("4Gi"),
			"cpu":    resource.MustParse("2000m"),
		},
		}
	}

	data := GetPGTuneConfigParameters(cluster)

	expectedData := map[string]string{
		"shared_buffers":                   "1024MB",
		"max_parallel_workers":             "2",
		"max_parallel_workers_per_gather":  "1",
		"max_worker_processes":             "2",
		"max_parallel_maintenance_workers": "1",
		"wal_buffers":                      "16MB",
		"work_mem":                         "5242kB",
		"default_statistics_target":        "100",
		"effective_cache_size":             "3072MB",
		"maintenance_work_mem":             "256MB",
		"min_wal_size":                     "1GB",
		"max_wal_size":                     "4GB",
	}

	for k, v := range expectedData {
		if !AssertConfigVal(data, k, v) {
			t.Fail()
		}
	}
}

func TestNoMemoryRequests(t *testing.T) {
	cluster := DefaultCluster()
	for i := range cluster.Spec.InstanceSets {
		cluster.Spec.InstanceSets[i].Default(i)
		cluster.Spec.InstanceSets[i].Resources = v1.ResourceRequirements{Requests: v1.ResourceList{
			"cpu": resource.MustParse("2000m"),
		},
		}
	}

	data := GetPGTuneConfigParameters(cluster)

	expectedData := map[string]string{
		"max_parallel_workers":             "2",
		"max_parallel_workers_per_gather":  "1",
		"max_worker_processes":             "2",
		"max_parallel_maintenance_workers": "1",
		"default_statistics_target":        "100",
		"min_wal_size":                     "1GB",
		"max_wal_size":                     "4GB",
	}

	for k, v := range expectedData {
		if !AssertConfigVal(data, k, v) {
			t.Fail()
		}
	}
}

func TestNoCPURequests(t *testing.T) {
	cluster := DefaultCluster()
	for i := range cluster.Spec.InstanceSets {
		cluster.Spec.InstanceSets[i].Default(i)
		cluster.Spec.InstanceSets[i].Resources = v1.ResourceRequirements{Requests: v1.ResourceList{
			"memory": resource.MustParse("4Gi"),
		},
		}
	}

	data := GetPGTuneConfigParameters(cluster)

	expectedData := map[string]string{
		"shared_buffers":            "1024MB",
		"wal_buffers":               "16MB",
		"default_statistics_target": "100",
		"effective_cache_size":      "3072MB",
		"maintenance_work_mem":      "256MB",
		"min_wal_size":              "1GB",
		"max_wal_size":              "4GB",
	}

	for k, v := range expectedData {
		if !AssertConfigVal(data, k, v) {
			t.Fail()
		}
	}
}
