package pgtune

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

/* This package calculates recommended configuration settings according to pgTune (https://pgtune.leopard.in.ua/), assuming DBType is mixed
The settings are updated using Patroni Dynamic Conifguration which is already implemented, so this essentialy enables Patroni
This works only if PostgresCluster.Spec.autoPgTune property is set to true (default is false)
If both PostgresCluster.Spec.autoPgTune and PostgresCluster.Spec.Patroni.DynamicConfiguration are enabled, DynamicConfiguration will override this.
Full credit goes to https://github.com/le0pard/pgtune */

import (
	"fmt"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	SizeKB        = "KB"
	SizeMB        = "MB"
	SizeGB        = "GB"
	SharedBuffers = "shared_buffers"
)

func GetPGTuneConfigParameters(cluster *v1beta1.PostgresCluster) map[string]interface{} {
	parameters := make(map[string]interface{})
	// postgresql := make(map[string]interface{})
	// cfg := make(map[string]interface{})

	TuneSharedBuffers(cluster, parameters)

	// postgresql["parameters"] = parameters
	// cfg["postgresql"] = postgresql
	return parameters
}

func initPatroniForPGTune(cluster *v1beta1.PostgresCluster) {
	if cluster.Spec.Patroni == nil {
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{DynamicConfiguration: runtime.RawExtension{}}
	}
	if cluster.Spec.Patroni.DynamicConfiguration.Raw == nil {
		cluster.Spec.Patroni.DynamicConfiguration.Raw = []byte{}
	}
}

func TotalMemoryInKB(cluster *v1beta1.PostgresCluster) int64 {
	totalMemInBytes := cluster.Spec.InstanceSets[0].Resources.Requests.Memory().Value()
	totalMemInKB := totalMemInBytes / 1024
	return totalMemInKB
}

func EditPatroniSpec(cluster *v1beta1.PostgresCluster) error {
	return nil
}

func InitConfigString(cfg map[string]interface{}) {

}

// TuneSharedBuffers calculates work_mem recommended configuration property
func TuneWorkMem(cluster *v1beta1.PostgresCluster) error {
	return nil
}

// TuneSharedBuffers calculates shared_buffers recommended configuration property
func TuneSharedBuffers(cluster *v1beta1.PostgresCluster, params map[string]interface{}) error {
	totalMemInKB := TotalMemoryInKB(cluster)
	if totalMemInKB > 0 {
		params[SharedBuffers] = fmt.Sprintf("%dGB", GB(totalMemInKB/4, SizeKB))
	}
	return nil
}

func GB(t int64, convertFrom string) int64 {
	switch convertFrom {
	case SizeMB:
		return t / 1024
	case SizeKB:
		return t / (1024 * 1024)
	}
	return t
}

func MB(t int64, convertFrom string) int64 {
	switch convertFrom {
	case SizeGB:
		return t * 1024
	case SizeKB:
		return t / 1024
	}
	return t
}

func KB(t int64, convertFrom string) int64 {
	switch convertFrom {
	case SizeMB:
		return t * 1024
	case SizeGB:
		return t * 1024 * 1024
	}
	return t
}
