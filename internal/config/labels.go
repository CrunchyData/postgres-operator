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

// resource labels used by the operator
const LABEL_NAME = "name"
const LABEL_SELECTOR = "selector"
const LABEL_OPERATOR = "postgres-operator"
const LABEL_PG_CLUSTER = "pg-cluster"
const LABEL_PG_CLUSTER_IDENTIFIER = "pg-cluster-id"
const LABEL_PG_DATABASE = "pgo-pg-database"

const LABEL_PGTASK = "pg-task"

const LABEL_AUTOFAIL = "autofail"
const LABEL_FAILOVER = "failover"

const LABEL_TARGET = "target"
const LABEL_RMDATA = "pgrmdata"

const LABEL_PGPOLICY = "pgpolicy"
const LABEL_INGEST = "ingest"
const LABEL_PGREMOVE = "pgremove"
const LABEL_PVCNAME = "pvcname"
const LABEL_EXPORTER = "crunchy-postgres-exporter"
const LABEL_EXPORTER_PG_USER = "ccp_monitoring"
const LABEL_ARCHIVE = "archive"
const LABEL_ARCHIVE_TIMEOUT = "archive-timeout"
const LABEL_CUSTOM_CONFIG = "custom-config"
const LABEL_NODE_LABEL_KEY = "NodeLabelKey"
const LABEL_NODE_LABEL_VALUE = "NodeLabelValue"
const LABEL_REPLICA_NAME = "replica-name"
const LABEL_CCP_IMAGE_TAG_KEY = "ccp-image-tag"
const LABEL_CCP_IMAGE_KEY = "ccp-image"
const LABEL_IMAGE_PREFIX = "image-prefix"
const LABEL_SERVICE_TYPE = "service-type"
const LABEL_POD_ANTI_AFFINITY = "pg-pod-anti-affinity"
const LABEL_SYNC_REPLICATION = "sync-replication"

const LABEL_REPLICA_COUNT = "replica-count"
const LABEL_STORAGE_CONFIG = "storage-config"
const LABEL_NODE_LABEL = "node-label"
const LABEL_VERSION = "version"
const LABEL_PGO_VERSION = "pgo-version"
const LABEL_DELETE_DATA = "delete-data"
const LABEL_DELETE_DATA_STARTED = "delete-data-started"
const LABEL_DELETE_BACKUPS = "delete-backups"
const LABEL_IS_REPLICA = "is-replica"
const LABEL_IS_BACKUP = "is-backup"
const LABEL_STARTUP = "startup"
const LABEL_SHUTDOWN = "shutdown"

// label for the pgcluster upgrade
const LABEL_UPGRADE = "upgrade"

const LABEL_BACKREST = "pgo-backrest"
const LABEL_BACKREST_JOB = "pgo-backrest-job"
const LABEL_BACKREST_RESTORE = "pgo-backrest-restore"
const LABEL_CONTAINER_NAME = "containername"
const LABEL_POD_NAME = "podname"
const LABEL_BACKREST_REPO_SECRET = "backrest-repo-config"
const LABEL_BACKREST_COMMAND = "backrest-command"
const LABEL_BACKREST_RESTORE_FROM_CLUSTER = "backrest-restore-from-cluster"
const LABEL_BACKREST_RESTORE_OPTS = "backrest-restore-opts"
const LABEL_BACKREST_BACKUP_OPTS = "backrest-backup-opts"
const LABEL_BACKREST_OPTS = "backrest-opts"
const LABEL_BACKREST_PITR_TARGET = "backrest-pitr-target"
const LABEL_BACKREST_STORAGE_TYPE = "backrest-storage-type"
const LABEL_BACKREST_S3_VERIFY_TLS = "backrest-s3-verify-tls"
const LABEL_BADGER = "crunchy-pgbadger"
const LABEL_BADGER_CCPIMAGE = "crunchy-pgbadger"
const LABEL_BACKUP_TYPE_BACKREST = "pgbackrest"
const LABEL_BACKUP_TYPE_PGDUMP = "pgdump"

const LABEL_PGDUMP_COMMAND = "pgdump"
const LABEL_PGDUMP_RESTORE = "pgdump-restore"
const LABEL_PGDUMP_OPTS = "pgdump-opts"
const LABEL_PGDUMP_HOST = "pgdump-host"
const LABEL_PGDUMP_DB = "pgdump-db"
const LABEL_PGDUMP_USER = "pgdump-user"
const LABEL_PGDUMP_PORT = "pgdump-port"
const LABEL_PGDUMP_ALL = "pgdump-all"
const LABEL_PGDUMP_PVC = "pgdump-pvc"

const LABEL_RESTORE_TYPE_PGRESTORE = "pgrestore"
const LABEL_PGRESTORE_COMMAND = "pgrestore"
const LABEL_PGRESTORE_HOST = "pgrestore-host"
const LABEL_PGRESTORE_DB = "pgrestore-db"
const LABEL_PGRESTORE_USER = "pgrestore-user"
const LABEL_PGRESTORE_PORT = "pgrestore-port"
const LABEL_PGRESTORE_FROM_CLUSTER = "pgrestore-from-cluster"
const LABEL_PGRESTORE_FROM_PVC = "pgrestore-from-pvc"
const LABEL_PGRESTORE_OPTS = "pgrestore-opts"
const LABEL_PGRESTORE_PITR_TARGET = "pgrestore-pitr-target"

const LABEL_DATA_ROOT = "data-root"
const LABEL_PVC_NAME = "pvc-name"
const LABEL_VOLUME_NAME = "volume-name"

const LABEL_SESSION_ID = "sessionid"
const LABEL_USERNAME = "username"
const LABEL_ROLENAME = "rolename"
const LABEL_PASSWORD = "password"

const LABEL_PGADMIN = "crunchy-pgadmin"
const LABEL_PGADMIN_TASK_ADD = "pgadmin-add"
const LABEL_PGADMIN_TASK_CLUSTER = "pgadmin-cluster"
const LABEL_PGADMIN_TASK_DELETE = "pgadmin-delete"

const LABEL_PGBOUNCER = "crunchy-pgbouncer"

const LABEL_JOB_NAME = "job-name"
const LABEL_PGBACKREST_STANZA = "pgbackrest-stanza"
const LABEL_PGBACKREST_DB_PATH = "pgbackrest-db-path"
const LABEL_PGBACKREST_REPO_PATH = "pgbackrest-repo-path"
const LABEL_PGBACKREST_REPO_HOST = "pgbackrest-repo-host"

const LABEL_PGO_BACKREST_REPO = "pgo-backrest-repo"

// a general label for grouping all the tasks...helps with cleanups
const LABEL_PGO_CLONE = "pgo-clone"

// the individualized step labels
const LABEL_PGO_CLONE_STEP_1 = "pgo-clone-step-1"
const LABEL_PGO_CLONE_STEP_2 = "pgo-clone-step-2"
const LABEL_PGO_CLONE_STEP_3 = "pgo-clone-step-3"

const LABEL_DEPLOYMENT_NAME = "deployment-name"
const LABEL_SERVICE_NAME = "service-name"
const LABEL_CURRENT_PRIMARY = "current-primary"

const LABEL_CLAIM_NAME = "claimName"

const LABEL_PGO_PGOUSER = "pgo-pgouser"
const LABEL_PGO_PGOROLE = "pgo-pgorole"
const LABEL_PGOUSER = "pgouser"
const LABEL_WORKFLOW_ID = "workflowid" // NOTE: this now matches crv1.PgtaskWorkflowID

const LABEL_TRUE = "true"
const LABEL_FALSE = "false"

const LABEL_NAMESPACE = "namespace"
const LABEL_PGO_INSTALLATION_NAME = "pgo-installation-name"
const LABEL_VENDOR = "vendor"
const LABEL_CRUNCHY = "crunchydata"
const LABEL_PGO_CREATED_BY = "pgo-created-by"
const LABEL_PGO_UPDATED_BY = "pgo-updated-by"

const LABEL_FAILOVER_STARTED = "failover-started"

const GLOBAL_CUSTOM_CONFIGMAP = "pgo-custom-pg-config"

const LABEL_PGHA_SCOPE = "crunchy-pgha-scope"
const LABEL_PGHA_CONFIGMAP = "pgha-config"
const LABEL_PGHA_BACKUP_TYPE = "pgha-backup-type"
const LABEL_PGHA_ROLE = "role"
const LABEL_PGHA_ROLE_PRIMARY = "master"
const LABEL_PGHA_ROLE_REPLICA = "replica"
const LABEL_PGHA_BOOTSTRAP = "pgha-bootstrap"
