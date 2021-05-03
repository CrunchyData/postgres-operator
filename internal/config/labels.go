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

// resource labels used by the operator
const (
	LABEL_NAME        = "name"
	LABEL_SELECTOR    = "selector"
	LABEL_OPERATOR    = "postgres-operator"
	LABEL_PG_CLUSTER  = "pg-cluster"
	LABEL_PG_DATABASE = "pgo-pg-database"
)

const LABEL_PGTASK = "pg-task"

const LABEL_RESTART = "restart"

const (
	LABEL_RMDATA = "pgrmdata"
)

const (
	LABEL_PGPOLICY           = "pgpolicy"
	LABEL_PVCNAME            = "pvcname"
	LABEL_EXPORTER           = "crunchy-postgres-exporter"
	LABEL_ARCHIVE            = "archive"
	LABEL_ARCHIVE_TIMEOUT    = "archive-timeout"
	LABEL_NODE_AFFINITY_TYPE = "node-affinity-type"
	LABEL_NODE_LABEL_KEY     = "NodeLabelKey"
	LABEL_NODE_LABEL_VALUE   = "NodeLabelValue"
	LABEL_REPLICA_NAME       = "replica-name"
	LABEL_CCP_IMAGE_TAG_KEY  = "ccp-image-tag"
	LABEL_CCP_IMAGE_KEY      = "ccp-image"
	LABEL_IMAGE_PREFIX       = "image-prefix"
	LABEL_POD_ANTI_AFFINITY  = "pg-pod-anti-affinity"
)

const (
	LABEL_REPLICA_COUNT       = "replica-count"
	LABEL_STORAGE_CONFIG      = "storage-config"
	LABEL_NODE_LABEL          = "node-label"
	LABEL_VERSION             = "version"
	LABEL_PGO_VERSION         = "pgo-version"
	LABEL_DELETE_DATA         = "delete-data"
	LABEL_DELETE_DATA_STARTED = "delete-data-started"
	LABEL_DELETE_BACKUPS      = "delete-backups"
	LABEL_IS_REPLICA          = "is-replica"
	LABEL_IS_BACKUP           = "is-backup"
	LABEL_RM_TOLERATIONS      = "rmdata-tolerations"
	LABEL_STARTUP             = "startup"
	LABEL_SHUTDOWN            = "shutdown"
)

// label for the pgcluster upgrade
const LABEL_UPGRADE = "upgrade"

const (
	LABEL_BACKREST         = "pgo-backrest"
	LABEL_BACKREST_JOB     = "pgo-backrest-job"
	LABEL_BACKREST_RESTORE = "pgo-backrest-restore"
	LABEL_CONTAINER_NAME   = "containername"
	LABEL_POD_NAME         = "podname"
	// #nosec: G101
	LABEL_BACKREST_REPO_SECRET          = "backrest-repo-config"
	LABEL_BACKREST_COMMAND              = "backrest-command"
	LABEL_BACKREST_RESTORE_FROM_CLUSTER = "backrest-restore-from-cluster"
	LABEL_BACKREST_RESTORE_OPTS         = "backrest-restore-opts"
	LABEL_BACKREST_BACKUP_OPTS          = "backrest-backup-opts"
	LABEL_BACKREST_OPTS                 = "backrest-opts"
	LABEL_BACKREST_PITR_TARGET          = "backrest-pitr-target"
	LABEL_BACKREST_STORAGE_TYPE         = "backrest-storage-type"
	LABEL_BACKREST_S3_VERIFY_TLS        = "backrest-s3-verify-tls"
	LABEL_BACKUP_TYPE_BACKREST          = "pgbackrest"
	LABEL_BACKUP_TYPE_PGDUMP            = "pgdump"
)

const (
	LABEL_PGDUMP_COMMAND = "pgdump"
	LABEL_PGDUMP_RESTORE = "pgdump-restore"
	LABEL_PGDUMP_OPTS    = "pgdump-opts"
	LABEL_PGDUMP_HOST    = "pgdump-host"
	LABEL_PGDUMP_DB      = "pgdump-db"
	LABEL_PGDUMP_USER    = "pgdump-user"
	LABEL_PGDUMP_PORT    = "pgdump-port"
	LABEL_PGDUMP_ALL     = "pgdump-all"
	LABEL_PGDUMP_PVC     = "pgdump-pvc"
)

const (
	LABEL_RESTORE_TYPE_PGRESTORE = "pgrestore"
	LABEL_PGRESTORE_COMMAND      = "pgrestore"
	LABEL_PGRESTORE_HOST         = "pgrestore-host"
	LABEL_PGRESTORE_DB           = "pgrestore-db"
	LABEL_PGRESTORE_USER         = "pgrestore-user"
	LABEL_PGRESTORE_PORT         = "pgrestore-port"
	LABEL_PGRESTORE_FROM_CLUSTER = "pgrestore-from-cluster"
	LABEL_PGRESTORE_FROM_PVC     = "pgrestore-from-pvc"
	LABEL_PGRESTORE_OPTS         = "pgrestore-opts"
	LABEL_PGRESTORE_PITR_TARGET  = "pgrestore-pitr-target"
)

const (
	LABEL_DATA_ROOT   = "data-root"
	LABEL_PVC_NAME    = "pvc-name"
	LABEL_VOLUME_NAME = "volume-name"
)

const (
	LABEL_SESSION_ID = "sessionid"
	LABEL_USERNAME   = "username"
	LABEL_ROLENAME   = "rolename"
	LABEL_PASSWORD   = "password"
)

const (
	LABEL_PGADMIN              = "crunchy-pgadmin"
	LABEL_PGADMIN_TASK_ADD     = "pgadmin-add"
	LABEL_PGADMIN_TASK_CLUSTER = "pgadmin-cluster"
	LABEL_PGADMIN_TASK_DELETE  = "pgadmin-delete"
)

const LABEL_PGBOUNCER = "crunchy-pgbouncer"

const (
	LABEL_JOB_NAME             = "job-name"
	LABEL_PGBACKREST_STANZA    = "pgbackrest-stanza"
	LABEL_PGBACKREST_DB_PATH   = "pgbackrest-db-path"
	LABEL_PGBACKREST_REPO_PATH = "pgbackrest-repo-path"
	LABEL_PGBACKREST_REPO_HOST = "pgbackrest-repo-host"
)

const LABEL_PGO_BACKREST_REPO = "pgo-backrest-repo"

const (
	LABEL_DEPLOYMENT_NAME = "deployment-name"
	LABEL_SERVICE_NAME    = "service-name"
	LABEL_CURRENT_PRIMARY = "current-primary"
)

const LABEL_CLAIM_NAME = "claimName"

const (
	LABEL_PGO_PGOUSER = "pgo-pgouser"
	LABEL_PGO_PGOROLE = "pgo-pgorole"
	LABEL_PGOUSER     = "pgouser"
	LABEL_WORKFLOW_ID = "workflowid" // NOTE: this now matches crv1.PgtaskWorkflowID
)

const (
	LABEL_TRUE  = "true"
	LABEL_FALSE = "false"
)

const (
	LABEL_NAMESPACE             = "namespace"
	LABEL_PGO_INSTALLATION_NAME = "pgo-installation-name"
	LABEL_VENDOR                = "vendor"
	LABEL_CRUNCHY               = "crunchydata"
	LABEL_PGO_CREATED_BY        = "pgo-created-by"
	LABEL_PGO_UPDATED_BY        = "pgo-updated-by"
)

const GLOBAL_CUSTOM_CONFIGMAP = "pgo-custom-pg-config"

const (
	LABEL_PGHA_SCOPE               = "crunchy-pgha-scope"
	LABEL_PGHA_CONFIGMAP           = "pgha-config"
	LABEL_PGHA_BACKUP_TYPE         = "pgha-backup-type"
	LABEL_PGHA_ROLE                = "role"
	LABEL_PGHA_ROLE_PRIMARY        = "master"
	LABEL_PGHA_ROLE_REPLICA        = "replica"
	LABEL_PGHA_BOOTSTRAP           = "pgha-bootstrap"
	LABEL_PGHA_BOOTSTRAP_NAMESPACE = "pgha-bootstrap-namespace"
)
