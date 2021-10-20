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
	"gotest.tools/assert"
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
	cluster.Spec.AutoPGTune = &v1beta1.PostgresClusterPGTune{
		ApplicationType: "mixed",
	}
	return cluster
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

	actual := GetPGTuneConfigParameters(cluster)

	expected := map[string]interface{}{
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
		"min_wal_size":                     "1024MB",
		"max_wal_size":                     "4096MB",
	}

	assert.DeepEqual(t, expected, actual)
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

	actual := GetPGTuneConfigParameters(cluster)

	expected := map[string]interface{}{
		"max_parallel_workers":             "2",
		"max_parallel_workers_per_gather":  "1",
		"max_worker_processes":             "2",
		"max_parallel_maintenance_workers": "1",
		"default_statistics_target":        "100",
		"min_wal_size":                     "1024MB",
		"max_wal_size":                     "4096MB",
	}

	assert.DeepEqual(t, expected, actual)

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

	actual := GetPGTuneConfigParameters(cluster)

	expected := map[string]interface{}{
		"shared_buffers":            "1024MB",
		"wal_buffers":               "16MB",
		"default_statistics_target": "100",
		"effective_cache_size":      "3072MB",
		"maintenance_work_mem":      "256MB",
		"min_wal_size":              "1024MB",
		"max_wal_size":              "4096MB",
		"work_mem":                  "2621kB",
	}

	assert.DeepEqual(t, expected, actual)
}

func TestStorageTypes(t *testing.T) {
	for _, tt := range []struct {
		hdtype   string
		expected map[string]interface{}
	}{
		{
			hdtype: HDTypeSSD,
			expected: map[string]interface{}{
				"effective_io_concurrency":  "200",
				"random_page_cost":          "1.1",
				"max_wal_size":              "4096MB",
				"min_wal_size":              "1024MB",
				"default_statistics_target": "100",
			},
		},
		{
			hdtype: HDTypeHDD,
			expected: map[string]interface{}{
				"effective_io_concurrency":  "2",
				"random_page_cost":          "4",
				"max_wal_size":              "4096MB",
				"min_wal_size":              "1024MB",
				"default_statistics_target": "100",
			},
		},
		{
			hdtype: HDTypeSAN,
			expected: map[string]interface{}{
				"effective_io_concurrency":  "300",
				"random_page_cost":          "1.1",
				"max_wal_size":              "4096MB",
				"min_wal_size":              "1024MB",
				"default_statistics_target": "100",
			},
		},
	} {
		cluster := DefaultCluster()
		for i := range cluster.Spec.InstanceSets {
			cluster.Spec.InstanceSets[i].Default(i)
		}
		cluster.Spec.AutoPGTune.HDType = &tt.hdtype
		actual := GetPGTuneConfigParameters(cluster)

		assert.DeepEqual(t, tt.expected, actual)
	}
}

func TestApplicationTypesDefaults(t *testing.T) {
	for _, tt := range []struct {
		apptype  string
		expected map[string]interface{}
	}{
		{
			apptype: AppTypeDW,
			expected: map[string]interface{}{
				"min_wal_size":              "4096MB",
				"max_wal_size":              "16384MB",
				"default_statistics_target": "500",
			},
		},
		{
			apptype: AppTypeDesktop,
			expected: map[string]interface{}{
				"min_wal_size":              "100MB",
				"max_wal_size":              "2048MB",
				"default_statistics_target": "100",
			},
		},
		{
			apptype: AppTypeOLTP,
			expected: map[string]interface{}{
				"min_wal_size":              "2048MB",
				"max_wal_size":              "8192MB",
				"default_statistics_target": "100",
			},
		},
		{
			apptype: AppTypeWeb,
			expected: map[string]interface{}{
				"min_wal_size":              "1024MB",
				"max_wal_size":              "4096MB",
				"default_statistics_target": "100",
			},
		},
		{
			apptype: AppTypeMixed,
			expected: map[string]interface{}{
				"min_wal_size":              "1024MB",
				"max_wal_size":              "4096MB",
				"default_statistics_target": "100",
			},
		},
	} {
		cluster := DefaultCluster()
		for i := range cluster.Spec.InstanceSets {
			cluster.Spec.InstanceSets[i].Default(i)
		}
		cluster.Spec.AutoPGTune.ApplicationType = tt.apptype
		actual := GetPGTuneConfigParameters(cluster)

		assert.DeepEqual(t, tt.expected, actual)
	}
}

func TestDifferentParameters(t *testing.T) {
	for _, tt := range []struct {
		name     string
		apptype  string
		hdtype   string
		cpu      string
		memory   string
		expected map[string]interface{}
	}{
		{
			name:    "1-cpu-not-tuned",
			apptype: "web",
			hdtype:  "ssd",
			cpu:     "1000m",
			memory:  "2Gi",
			expected: map[string]interface{}{
				"shared_buffers":            "512MB",
				"effective_cache_size":      "1536MB",
				"maintenance_work_mem":      "128MB",
				"wal_buffers":               "16MB",
				"default_statistics_target": "100",
				"random_page_cost":          "1.1",
				"effective_io_concurrency":  "200",
				"work_mem":                  "1310kB",
				"min_wal_size":              "1024MB",
				"max_wal_size":              "4096MB",
			},
		},
		{
			name:    "no-app-type-assume-mixed",
			apptype: "",
			hdtype:  "hdd",
			cpu:     "4000m",
			memory:  "8Gi",
			expected: map[string]interface{}{
				"shared_buffers":                   "2048MB",
				"effective_cache_size":             "6144MB",
				"maintenance_work_mem":             "512MB",
				"wal_buffers":                      "16MB",
				"default_statistics_target":        "100",
				"random_page_cost":                 "4",
				"effective_io_concurrency":         "2",
				"work_mem":                         "5242kB",
				"min_wal_size":                     "1024MB",
				"max_wal_size":                     "4096MB",
				"max_parallel_maintenance_workers": "2",
				"max_parallel_workers":             "4",
				"max_parallel_workers_per_gather":  "2",
				"max_worker_processes":             "4",
			},
		},
		{
			name:    "nothing-is-cool",
			apptype: "",
			hdtype:  "",
			cpu:     "0",
			memory:  "0",
			expected: map[string]interface{}{
				"default_statistics_target": "100",
				"min_wal_size":              "1024MB",
				"max_wal_size":              "4096MB",
			},
		},
		{
			name:    "all-but-memory",
			apptype: "oltp",
			hdtype:  "san",
			cpu:     "7000m",
			memory:  "0",
			expected: map[string]interface{}{
				"default_statistics_target":        "100",
				"random_page_cost":                 "1.1",
				"effective_io_concurrency":         "300",
				"min_wal_size":                     "2048MB",
				"max_wal_size":                     "8192MB",
				"max_parallel_maintenance_workers": "4",
				"max_parallel_workers":             "7",
				"max_parallel_workers_per_gather":  "4",
				"max_worker_processes":             "7",
			},
		},
	} {
		cluster := DefaultCluster()
		for i := range cluster.Spec.InstanceSets {
			cluster.Spec.InstanceSets[i].Default(i)
		}
		if tt.apptype != "" {
			cluster.Spec.AutoPGTune.ApplicationType = tt.apptype
		}
		if tt.hdtype != "" {
			cluster.Spec.AutoPGTune.HDType = &tt.hdtype
		}
		for i := range cluster.Spec.InstanceSets {
			cluster.Spec.InstanceSets[i].Resources = v1.ResourceRequirements{}
			cluster.Spec.InstanceSets[i].Resources = v1.ResourceRequirements{Requests: v1.ResourceList{
				"memory": resource.MustParse(tt.memory),
				"cpu":    resource.MustParse(tt.cpu),
			},
			}

			actual := GetPGTuneConfigParameters(cluster)
			assert.DeepEqual(t, tt.expected, actual)
		}
	}
}
