package config

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	// ANNOTATION_BACKREST_RESTORE is used to annotate pgclusters that are restoring
	ANNOTATION_BACKREST_RESTORE       = "pgo-backrest-restore"
	ANNOTATION_PGHA_BOOTSTRAP_REPLICA = "pgo-pgha-bootstrap-replica"
	ANNOTATION_PRIMARY_DEPLOYMENT     = "primary-deployment"
	// ANNOTATION_CLUSTER_DO_NOT_RESIZE indicates on a custom resource update that
	// a specific instance should not be resized
	ANNOTATION_CLUSTER_DO_NOT_RESIZE = "do-not-resize"
	// ANNOTATION_CLUSTER_KEEP_BACKUPS indicates that if a custom resource is
	// deleted, ensure the backups are kept
	ANNOTATION_CLUSTER_KEEP_BACKUPS = "keep-backups"
	// ANNOTATION_CLUSTER_KEEP_DATA indicates that if a custom resource is
	// deleted, ensure the data directory is kept
	ANNOTATION_CLUSTER_KEEP_DATA = "keep-data"
	// annotation to track the cluster's current primary
	ANNOTATION_CURRENT_PRIMARY = "current-primary"
	// annotation to indicate whether a cluster has been upgraded
	ANNOTATION_IS_UPGRADED = "is-upgraded"
	// annotation to indicate an upgrade is in progress. this has the effect
	// of causeing the rmdata job in pgcluster to not run
	ANNOTATION_UPGRADE_IN_PROGRESS = "upgrade-in-progress"
	// annotation to store the Operator versions upgraded from and to
	ANNOTATION_UPGRADE_INFO = "upgrade-info"
	// annotation to store the string boolean, used when checking upgrade status
	ANNOTATIONS_FALSE = "false"
	// ANNOTATION_REPO_PATH is for storing the repository path for the pgBackRest repo in a cluster
	ANNOTATION_REPO_PATH = "repo-path"
	// ANNOTATION_PG_PORT is for storing the PostgreSQL port for a cluster
	ANNOTATION_PG_PORT = "pg-port"
	// ANNOTATION_GCS_BUCKET is for storing the name of the GCS bucket used by
	// pgBackRest in a cluster
	ANNOTATION_GCS_BUCKET = "gcs-bucket"
	// ANNOTATION_GCS_ENDPOINT is for storing the name of the GCS endpoint used by
	// pgBackRest in a cluster
	ANNOTATION_GCS_ENDPOINT = "gcs-endpoint"
	// ANNOTATION_GCS_KEY_TYPE is for storing the GCS key type used by pgBackRest
	// in a cluster
	ANNOTATION_GCS_KEY_TYPE = "gcs-key-type"
	// ANNOTATION_S3_BUCKET is for storing the name of the S3 bucket used by pgBackRest in
	// a cluster
	ANNOTATION_S3_BUCKET = "s3-bucket"
	// ANNOTATION_S3_ENDPOINT is for storing the name of the S3 endpoint used by pgBackRest in
	// a cluster
	ANNOTATION_S3_ENDPOINT = "s3-endpoint"
	// ANNOTATION_S3_REGION is for storing the name of the S3 region used by pgBackRest in
	// a cluster
	ANNOTATION_S3_REGION = "s3-region"
	// ANNOTATION_S3_URI_STYLE is for storing the the URI style that should be used to access a
	// pgBackRest repository
	ANNOTATION_S3_URI_STYLE = "s3-uri-style"
	// ANNOTATION_S3_VERIFY_TLS is for storing the setting that determines whether or not TLS should
	// be used to access a pgBackRest repository
	ANNOTATION_S3_VERIFY_TLS = "s3-verify-tls"
	// ANNOTATION_SSHD_PORT is for storing the SSHD port used by the pgBackRest repository
	// service in a cluster
	ANNOTATION_SSHD_PORT = "sshd-port"
	// ANNOTATION_SUPPLEMENTAL_GROUPS is for storing the supplemental groups used with a cluster
	ANNOTATION_SUPPLEMENTAL_GROUPS = "supplemental-groups"
)
