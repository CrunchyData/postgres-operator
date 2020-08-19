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

// a list of container images that are available
const (
	CONTAINER_IMAGE_PGO_BACKREST              = "pgo-backrest"
	CONTAINER_IMAGE_PGO_BACKREST_REPO         = "pgo-backrest-repo"
	CONTAINER_IMAGE_PGO_BACKREST_REPO_SYNC    = "pgo-backrest-repo-sync"
	CONTAINER_IMAGE_PGO_BACKREST_RESTORE      = "pgo-backrest-restore"
	CONTAINER_IMAGE_PGO_CLIENT                = "pgo-client"
	CONTAINER_IMAGE_PGO_RMDATA                = "pgo-rmdata"
	CONTAINER_IMAGE_PGO_SQL_RUNNER            = "pgo-sqlrunner"
	CONTAINER_IMAGE_CRUNCHY_ADMIN             = "crunchy-admin"
	CONTAINER_IMAGE_CRUNCHY_BACKREST_RESTORE  = "crunchy-backrest-restore"
	CONTAINER_IMAGE_CRUNCHY_POSTGRES_EXPORTER = "crunchy-postgres-exporter"
	CONTAINER_IMAGE_CRUNCHY_GRAFANA           = "crunchy-grafana"
	CONTAINER_IMAGE_CRUNCHY_PGADMIN           = "crunchy-pgadmin4"
	CONTAINER_IMAGE_CRUNCHY_PGBADGER          = "crunchy-pgbadger"
	CONTAINER_IMAGE_CRUNCHY_PGBOUNCER         = "crunchy-pgbouncer"
	CONTAINER_IMAGE_CRUNCHY_PGDUMP            = "crunchy-pgdump"
	CONTAINER_IMAGE_CRUNCHY_PGRESTORE         = "crunchy-pgrestore"
	CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA       = "crunchy-postgres-ha"
	CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA   = "crunchy-postgres-gis-ha"
	CONTAINER_IMAGE_CRUNCHY_PROMETHEUS        = "crunchy-prometheus"
)

// a map of the "RELATED_IMAGE_*" environmental variables to their defined
// container image names, which allows certain packagers to inject the full
// definition for where to pull a container image from
//
// See: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/contributors/design-proposals/related-images.md
var RelatedImageMap = map[string]string{
	"RELATED_IMAGE_PGO_BACKREST":              CONTAINER_IMAGE_PGO_BACKREST,
	"RELATED_IMAGE_PGO_BACKREST_REPO":         CONTAINER_IMAGE_PGO_BACKREST_REPO,
	"RELATED_IMAGE_PGO_BACKREST_REPO_SYNC":    CONTAINER_IMAGE_PGO_BACKREST_REPO_SYNC,
	"RELATED_IMAGE_PGO_BACKREST_RESTORE":      CONTAINER_IMAGE_PGO_BACKREST_RESTORE,
	"RELATED_IMAGE_PGO_CLIENT":                CONTAINER_IMAGE_PGO_CLIENT,
	"RELATED_IMAGE_PGO_RMDATA":                CONTAINER_IMAGE_PGO_RMDATA,
	"RELATED_IMAGE_PGO_SQL_RUNNER":            CONTAINER_IMAGE_PGO_SQL_RUNNER,
	"RELATED_IMAGE_CRUNCHY_ADMIN":             CONTAINER_IMAGE_CRUNCHY_ADMIN,
	"RELATED_IMAGE_CRUNCHY_BACKREST_RESTORE":  CONTAINER_IMAGE_CRUNCHY_BACKREST_RESTORE,
	"RELATED_IMAGE_CRUNCHY_POSTGRES_EXPORTER": CONTAINER_IMAGE_CRUNCHY_POSTGRES_EXPORTER,
	"RELATED_IMAGE_CRUNCHY_PGADMIN":           CONTAINER_IMAGE_CRUNCHY_PGADMIN,
	"RELATED_IMAGE_CRUNCHY_PGBADGER":          CONTAINER_IMAGE_CRUNCHY_PGBADGER,
	"RELATED_IMAGE_CRUNCHY_PGBOUNCER":         CONTAINER_IMAGE_CRUNCHY_PGBOUNCER,
	"RELATED_IMAGE_CRUNCHY_PGDUMP":            CONTAINER_IMAGE_CRUNCHY_PGDUMP,
	"RELATED_IMAGE_CRUNCHY_PGRESTORE":         CONTAINER_IMAGE_CRUNCHY_PGRESTORE,
	"RELATED_IMAGE_CRUNCHY_POSTGRES_HA":       CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA,
	"RELATED_IMAGE_CRUNCHY_POSTGRES_GIS_HA":   CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA,
}
