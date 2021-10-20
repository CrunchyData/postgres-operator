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

/* This package calculates recommended configuration settings according to pgTune (https://pgtune.leopard.in.ua/)
The settings are updated using Patroni Dynamic Conifguration which is already implemented, so this essentialy enables Patroni
This works only if PostgresCluster.Spec.autoPgTune property is set to true (default is false)
If both PostgresCluster.Spec.autoPgTune and PostgresCluster.Spec.Patroni.DynamicConfiguration are enabled, DynamicConfiguration will override this.
Memory and CPU values are taken from resources.requests propety,
Storage and Application Type values are CR fields.
Default Application Type is mixed, everything else will be ignored if not set.
Full credit goes to https://github.com/le0pard/pgtune */

import (
	"fmt"
	"math"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	SizeKB = "KB"
	SizeMB = "MB"
	SizeGB = "GB"

	HDTypeSSD = "ssd"
	HDTypeHDD = "hdd"
	HDTypeSAN = "san"

	AppTypeWeb     = "web"
	AppTypeOLTP    = "oltp"
	AppTypeDW      = "dw"
	AppTypeDesktop = "desktop"
	AppTypeMixed   = "mixed"

	NameSharedBuffers                 = "shared_buffers"
	NameWorkMem                       = "work_mem"
	NameEffectiveCacheSize            = "effective_cache_size"
	NameMaintenanceWorkMem            = "maintenance_work_mem"
	NameWalBuffers                    = "wal_buffers"
	NameDefaultStatisticsTarget       = "default_statistics_target"
	NameMinWalSize                    = "min_wal_size"
	NameMaxWalSize                    = "max_wal_size"
	NameMaxWorkerProcesses            = "max_worker_processes"
	NameMaxParallelWorkersPerGather   = "max_parallel_workers_per_gather"
	NameMaxParallelWorkers            = "max_parallel_workers"
	NameMaxParallelMaintenanceWorkers = "max_parallel_maintenance_workers"
	NameRandomPageCost                = "random_page_cost"
	NameEffectiveIOConcurrency        = "effective_io_concurrency"

	//Do not assign more then 10GB shared buffers
	MaxSharedBuffers = 10 * 1024
	//Do not assign more then 2GB RAM for maintenance
	MaxMaintenanceWorkMem             = 2048
	MaxWalBuffersKB                   = 16 * 1024 //16MB at most, represented in KB
	MinWalBuffersKB                   = 32        //32KB at least
	MaxWorkersPerGather               = 4
	MinWorkMem                        = 64 //64kb at least
	DefaultWorkersPerGatherForWorkMem = 2

	MaxConnections = 100 //This is already set up by Patroni
)

func GetPGTuneConfigParameters(cluster *v1beta1.PostgresCluster) map[string]interface{} {
	parameters := make(map[string]interface{})
	totalMemKB := TotalMemoryInKB(cluster)

	sharedBuffersVal := TuneSharedBuffers(cluster, parameters, totalMemKB)
	TuneDefaultStatisticsTarget(cluster, parameters)
	TuneEffectiveCacheSize(cluster, parameters, totalMemKB)
	TuneMaintenanceWorkMem(cluster, parameters, totalMemKB)
	TuneWalBuffers(cluster, parameters, sharedBuffersVal)
	TuneMinWalSize(cluster, parameters)
	TuneMaxWalSize(cluster, parameters)
	TuneRandomPageCost(cluster, parameters)
	TuneEffectiveIOConcurrency(cluster, parameters)
	parallelWorkersPerGather := TuneParallelSettings(cluster, parameters)
	TuneWorkMem(cluster, parameters, totalMemKB, sharedBuffersVal, parallelWorkersPerGather)

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

// TotalMemoryInKB returns memory request value in KB. 0 if no memory request has been made.
func TotalMemoryInKB(cluster *v1beta1.PostgresCluster) int64 {
	totalMemInBytes := cluster.Spec.InstanceSets[0].Resources.Requests.Memory().Value()
	totalMemInKB := totalMemInBytes / 1024
	return totalMemInKB
}

//TotalCPUCores returns the number of CPU cores. This is validated by Kubernetes so nov validation is needed.
func TotalCPUCores(cluster *v1beta1.PostgresCluster) int64 {
	totalCPU := cluster.Spec.InstanceSets[0].Resources.Requests.Cpu().Value()
	return totalCPU
}

/*
	Each of the following functions calculate a unique property following PGTune recommendations.
*/

// TuneWorkMem calculates work_mem recommended configuration property
// This is the most complicated one and it will be tuned only if both cpu and memory
// and memory have been requested. cpu must be greather than 2 as it depends on parallel settings
func TuneWorkMem(cluster *v1beta1.PostgresCluster, params map[string]interface{}, totalMemKB int64, sharedBuffers int, parallelWorkersPerGather int64) {
	// This will be tuned only if both memory and cpu has been requested.
	if parallelWorkersPerGather == 0 {
		parallelWorkersPerGather = DefaultWorkersPerGatherForWorkMem
	}
	if totalMemKB > 0 { //totalMemKB > 0 implies sharedBuffers > 0
		factor := 2 //mixed,web and dw
		fmt.Println(cluster.Spec.AutoPGTune.ApplicationType)
		switch cluster.Spec.AutoPGTune.ApplicationType {
		case AppTypeOLTP:
			factor = 1
		case AppTypeDesktop:
			factor = 6
		}
		workMem := int64((totalMemKB - KB(int64(sharedBuffers), SizeMB))) / (MaxConnections * 3) / parallelWorkersPerGather / int64(factor)
		if workMem < MinWorkMem {
			workMem = MinWorkMem
		}
		params[NameWorkMem] = fmt.Sprintf("%dkB", workMem)
	}
}

// TuneSharedBuffers calculates shared_buffers recommended configuration property which is Memory/4.
// returns sharedBuffers value or 0 if memory request is not set.
func TuneSharedBuffers(cluster *v1beta1.PostgresCluster, params map[string]interface{}, totalMemKB int64) int {
	// if totalMemKB == 0, then memory request has not been set. Do not assign sharedBuffers in that case.
	if totalMemKB > 0 {
		factor := 1
		if cluster.Spec.AutoPGTune.ApplicationType == AppTypeDesktop {
			factor = 4
		}
		// divide by 16 for desktop, 4 for everything else
		sharedBuffersVal := int(math.Min(float64(MB(totalMemKB/int64(4*factor), SizeKB)), MaxSharedBuffers))
		params[NameSharedBuffers] = fmt.Sprintf("%dMB", sharedBuffersVal)
		return sharedBuffersVal
	}
	return 0
}

func TuneEffectiveCacheSize(cluster *v1beta1.PostgresCluster, params map[string]interface{}, totalMemKB int64) {
	// if totalMemKB == 0, then memory request has not been set. Do not assign EffectiveCacheSize in that case.
	if totalMemKB > 0 {
		factor := 3
		if cluster.Spec.AutoPGTune.ApplicationType == AppTypeDesktop {
			factor = 1
		}
		// factor by 1/4 for desktop, 3/4 for anything else
		params[NameEffectiveCacheSize] = fmt.Sprintf("%dMB", MB(totalMemKB*int64(factor)/4, SizeKB))
	}
}

func TuneMaintenanceWorkMem(cluster *v1beta1.PostgresCluster, params map[string]interface{}, totalMemKB int64) {
	// if totalMemKB == 0, then memory request has not been set. Do not assign MaintenanceWorkMem in that case.
	if totalMemKB > 0 {
		factor := 2
		if cluster.Spec.AutoPGTune.ApplicationType == AppTypeDW {
			factor = 1
		}
		// divide by 8 for DW, 16 for anything else
		params[NameMaintenanceWorkMem] = fmt.Sprintf("%dMB", int(math.Min(float64(MB(totalMemKB/int64(8*factor), SizeKB)), MaxMaintenanceWorkMem))) //cap at 2GB
	}
}

func TuneWalBuffers(cluster *v1beta1.PostgresCluster, params map[string]interface{}, SharedBuffersMB int) {
	//SharedBuffersMB == 0 if and only if requests.memory is not set
	if SharedBuffersMB > 0 {
		walBuffersValue := math.Ceil(3 * float64(KB(int64(SharedBuffersMB), SizeMB)) / 100) //3% of SharedBuffers value
		walBuffersValue = math.Min(walBuffersValue, MaxWalBuffersKB)                        //at most MaxWalBuffers
		walBuffersValue = math.Max(walBuffersValue, MinWalBuffersKB)                        //at least MinWalBuffers
		if walBuffersValue >= 1024 {                                                        //format to MB
			params[NameWalBuffers] = fmt.Sprintf("%dMB", MB(int64(walBuffersValue), SizeKB))
		} else {
			params[NameWalBuffers] = fmt.Sprintf("%.fKB", walBuffersValue)
		}
	}
}

func TuneDefaultStatisticsTarget(cluster *v1beta1.PostgresCluster, params map[string]interface{}) {
	dst := 100
	if cluster.Spec.AutoPGTune.ApplicationType == AppTypeDW {
		dst = 500
	}
	params[NameDefaultStatisticsTarget] = fmt.Sprintf("%d", dst)
}

func TuneMinWalSize(cluster *v1beta1.PostgresCluster, params map[string]interface{}) {
	walSize := 1024 //mixed and web
	switch cluster.Spec.AutoPGTune.ApplicationType {
	case AppTypeDW:
		walSize = 4096
	case AppTypeOLTP:
		walSize = 2048
	case AppTypeDesktop:
		walSize = 100
	}
	params[NameMinWalSize] = fmt.Sprintf("%dMB", walSize)
}

func TuneMaxWalSize(cluster *v1beta1.PostgresCluster, params map[string]interface{}) {
	walSize := 4096 //mixed and web
	switch cluster.Spec.AutoPGTune.ApplicationType {
	case AppTypeDW:
		walSize = 16384
	case AppTypeOLTP:
		walSize = 8192
	case AppTypeDesktop:
		walSize = 2048
	}
	params[NameMaxWalSize] = fmt.Sprintf("%dMB", walSize)
}

/*
	TuneParallelSettings calculates all properties related to parallel execution
	They will be tuned only if cpu request is defined and greather than/equal to 2 cores.
	Returns the value of max_workers_per_gather property to be used later.
*/
func TuneParallelSettings(cluster *v1beta1.PostgresCluster, params map[string]interface{}) int64 {
	totalCPUCores := TotalCPUCores(cluster)
	if totalCPUCores >= 2 { //Do not tune for less than 2 CPUS. 0 means no CPU has been requested
		params[NameMaxWorkerProcesses] = fmt.Sprintf("%d", totalCPUCores)
		params[NameMaxParallelWorkers] = fmt.Sprintf("%d", totalCPUCores)

		WorkersPerGather := int64(math.Ceil(float64(totalCPUCores) / 2))
		CappedWorkersPerGather := int64(math.Min(float64(WorkersPerGather), MaxWorkersPerGather))
		if cluster.Spec.AutoPGTune.ApplicationType != AppTypeDW {
			// do not cap DW type application.
			WorkersPerGather = CappedWorkersPerGather
		}
		params[NameMaxParallelWorkersPerGather] = fmt.Sprintf("%d", WorkersPerGather)

		if cluster.Spec.PostgresVersion >= 11 {
			// cap any application type
			params[NameMaxParallelMaintenanceWorkers] = fmt.Sprintf("%d", CappedWorkersPerGather)
		}
		return WorkersPerGather
	}
	return 0
}

func TuneRandomPageCost(cluster *v1beta1.PostgresCluster, params map[string]interface{}) {
	if hdType := cluster.Spec.AutoPGTune.HDType; hdType != nil {
		rpc := 1.1
		if *hdType == HDTypeHDD {
			rpc = 4
		}
		params[NameRandomPageCost] = fmt.Sprintf("%g", rpc)
	}
}

func TuneEffectiveIOConcurrency(cluster *v1beta1.PostgresCluster, params map[string]interface{}) {
	if hdType := cluster.Spec.AutoPGTune.HDType; hdType != nil {
		iocon := 0
		switch *hdType {
		case HDTypeHDD:
			iocon = 2
		case HDTypeSSD:
			iocon = 200
		case HDTypeSAN:
			iocon = 300
		}
		if iocon > 0 {
			params[NameEffectiveIOConcurrency] = fmt.Sprintf("%d", iocon)
		}
	}
}

func GB(t int64, convertFrom string) int64 {
	switch convertFrom {
	case SizeMB:
		return int64(math.Ceil(float64(t) / 1024))
	case SizeKB:
		return int64(math.Ceil(float64(t) / (1024 * 1024)))
	}
	return t
}

func MB(t int64, convertFrom string) int64 {
	switch convertFrom {
	case SizeGB:
		return t * 1024
	case SizeKB:
		return int64(math.Ceil(float64(t) / 1024))
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
