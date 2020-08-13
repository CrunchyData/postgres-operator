package config

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// DefaultPgBouncerReplicas is the total number of Pods to place in a pgBouncer
// Deployment
const DefaultPgBouncerReplicas = 1

// Default resource values for deploying a PostgreSQL cluster. These values are
// utilized if the user has not provided these values either through
// configuration or from one-off API/CLI calls.
//
// These values were determined by either program defaults (e.g. the PostgreSQL
// one) and/or loose to vigorous experimentation and profiling
var (
	// DefaultBackrestRepoResourceMemory is the default value of the resource
	// request for memory for a pgBackRest repository
	DefaultBackrestResourceMemory = resource.MustParse("48Mi")
	// DefaultInstanceResourceMemory is the default value of the resource request
	// for memory for a PostgreSQL instance in a cluster
	DefaultInstanceResourceMemory = resource.MustParse("512Mi")
	// DefaultPgBouncerResourceMemory is the default value of the resource request
	// for memory of a pgBouncer instance
	DefaultPgBouncerResourceMemory = resource.MustParse("24Mi")
	// DefaultExporterResourceMemory is the default value of the resource request
	// for memory of a Crunchy Postgres Exporter instance
	DefaultExporterResourceMemory = resource.MustParse("24Mi")
)

// The following constants define the default refresh intervals for any informers created
// by that require a refresh interval
const (
	// ControllerGroupRefreshInterval is the default informer refresh interval in seconds
	// for the controllers created by the Controller Manager that require a refresh interval
	DefaultControllerGroupRefreshInterval = 60
	// NamespaceRefreshInterval is the default informer refresh interval in seconds
	// for the Operator's namespace controller
	DefaultNamespaceRefreshInterval = 60
)

// The following constants define the default number of workers created for the worker queues
// created within the various controller created by the Operator
const (
	// DefaultConfigMapWorkerCount defines the default number or workers for the worker queue
	// in the ConfigMap controller
	DefaultConfigMapWorkerCount = 2
	// DefaultNamespaceWorkerCount defines the default number or workers for the worker queue
	// in the Namespace controller
	DefaultNamespaceWorkerCount = 3
	// DefaultPGClusterWorkerCount defines the default number or workers for the worker queue
	// in the PGCluster controller
	DefaultPGClusterWorkerCount = 1
	// DefaultPGReplicaWorkerCount defines the default number or workers for the worker queue
	// in the PGReplica controller
	DefaultPGReplicaWorkerCount = 1
	// DefaultPGTaskWorkerCount defines the default number or workers for the worker queue
	// in the PGTask controller
	DefaultPGTaskWorkerCount = 1
)
