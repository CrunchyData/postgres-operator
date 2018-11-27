package util

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

import ()

// resource labels used by the operator
const LABEL_NAME = "name"
const LABEL_SELECTOR = "selector"
const LABEL_OPERATOR = "postgres-operator"
const LABEL_PG_CLUSTER = "pg-cluster"
const LABEL_PG_DATABASE = "pg-database"
const LABEL_PGBACKUP = "pgbackup"
const LABEL_AUTOFAIL = "autofail"
const LABEL_FAILOVER = "failover"
const LABEL_PRIMARY = "primary"
const LABEL_TARGET = "target"
const LABEL_RMDATA = "pgrmdata"

const LABEL_INGEST = "ingest"
const LABEL_PGREMOVE = "pgremove"
const LABEL_PVCNAME = "pvcname"
const LABEL_COLLECT = "crunchy_collect"
const LABEL_ARCHIVE = "archive"
const LABEL_ARCHIVE_TIMEOUT = "archive-timeout"
const LABEL_CUSTOM_CONFIG = "custom-config"
const LABEL_NODE_LABEL_KEY = "NodeLabelKey"
const LABEL_NODE_LABEL_VALUE = "NodeLabelValue"
const LABEL_REPLICA_NAME = "replica-name"
const LABEL_CCP_IMAGE_TAG_KEY = "ccp-image-tag"
const LABEL_SERVICE_TYPE = "service-type"

const LABEL_REPLICA_COUNT = "replica-count"
const LABEL_RESOURCES_CONFIG = "resources-config"
const LABEL_STORAGE_CONFIG = "storage-config"
const LABEL_NODE_LABEL = "node-label"
const LABEL_VERSION = "version"
const LABEL_PGO_VERSION = "pgo-version"
const LABEL_UPGRADE_DATE = "operator-upgrade-date"
const LABEL_DELETE_DATA = "delete-data"

const LABEL_BACKREST = "pgo-backrest"
const LABEL_CONTAINER_NAME = "containername"
const LABEL_POD_NAME = "podname"
const LABEL_BACKREST_COMMAND = "backrest-command"
const LABEL_BACKREST_RESTORE_FROM_CLUSTER = "backrest-restore-from-cluster"
const LABEL_BACKREST_RESTORE_TO_PVC = "backrest-restore-to-pvc"
const LABEL_BACKREST_RESTORE_OPTS = "backrest-restore-opts"
const LABEL_BACKREST_BACKUP_OPTS = "backrest-backup-opts"
const LABEL_BACKREST_OPTS = "backrest-opts"
const LABEL_BADGER = "crunchy-pgbadger"
const LABEL_BACKUP_TYPE_BASEBACKUP = "pgbasebackup"
const LABEL_BACKUP_TYPE_BACKREST = "pgbackrest"

const LABEL_DATA_ROOT = "data-root"
const LABEL_PVC_NAME = "pvc-name"
const LABEL_VOLUME_NAME = "volume-name"

const LABEL_SESSION_ID = "sessionid"
const LABEL_USERNAME = "username"
const LABEL_PASSWORD = "password"

const LABEL_PGPOOL = "crunchy-pgpool"
const LABEL_PGPOOL_SECRET = "pgpool-secret"
const LABEL_PGPOOL_TASK_ADD = "pgpool-add"
const LABEL_PGPOOL_TASK_DELETE = "pgpool-delete"
const LABEL_PGPOOL_TASK_CLUSTER = "pgpool-cluster"
const LABEL_PGPOOL_TASK_RECONFIGURE = "pgpool-reconfigure"
const LABEL_PGBOUNCER = "crunchy-pgbouncer"
const LABEL_PGBOUNCER_SECRET = "pgbouncer-secret"
const LABEL_PGBOUNCER_TASK_ADD = "pgbouncer-add"
const LABEL_PGBOUNCER_TASK_DELETE = "pgbouncer-delete"
const LABEL_PGBOUNCER_TASK_CLUSTER = "pgbouncer-cluster"
const LABEL_PGBOUNCER_TASK_RECONFIGURE = "pgbouncer-reconfigure"

const LABEL_PGBACKREST_STANZA = "pgbackrest-stanza"
const LABEL_PGBACKREST_DB_PATH = "pgbackrest-db-path"
const LABEL_PGBACKREST_REPO_PATH = "pgbackrest-repo-path"
