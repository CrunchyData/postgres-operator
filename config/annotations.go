package config

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

// annotations used by the operator
const (
	ANNOTATION_PGHA_BOOTSTRAP_REPLICA    = "pgo-pgha-bootstrap-replica"
	ANNOTATION_CLONE_BACKREST_PVC_SIZE   = "clone-backrest-pvc-size"
	ANNOTATION_CLONE_ENABLE_METRICS      = "clone-enable-metrics"
	ANNOTATION_CLONE_PVC_SIZE            = "clone-pvc-size"
	ANNOTATION_CLONE_SOURCE_CLUSTER_NAME = "clone-source-cluster-name"
	ANNOTATION_CLONE_TARGET_CLUSTER_NAME = "clone-target-cluster-name"
	ANNOTATION_PRIMARY_DEPLOYMENT        = "primary-deployment"
	// annotation to track the cluster's current primary
	ANNOTATION_CURRENT_PRIMARY = "current-primary"
	// annotation to indicate whether a cluster has been upgraded
	ANNOTATION_IS_UPGRADED = "is-upgraded"
	// annotation to store the Operator versions upgraded from and to
	ANNOTATION_UPGRADE_INFO = "upgrade-info"
	// annotation to store the string boolean, used when checking upgrade status
	ANNOTATIONS_FALSE = "false"
)
