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

// a list of container images that are available
const (
	CONTAINER_IMAGE_PGO_BACKREST              = "crunchy-pgbackrest"
	CONTAINER_IMAGE_PGO_BACKREST_REPO         = "crunchy-pgbackrest-repo"
	CONTAINER_IMAGE_PGO_CLIENT                = "pgo-client"
	CONTAINER_IMAGE_PGO_RMDATA                = "pgo-rmdata"
	CONTAINER_IMAGE_CRUNCHY_POSTGRES_EXPORTER = "crunchy-postgres-exporter"
	CONTAINER_IMAGE_CRUNCHY_GRAFANA           = "crunchy-grafana"
	CONTAINER_IMAGE_CRUNCHY_PGADMIN           = "crunchy-pgadmin4"
	CONTAINER_IMAGE_CRUNCHY_PGBADGER          = "crunchy-pgbadger"
	CONTAINER_IMAGE_CRUNCHY_PGBOUNCER         = "crunchy-pgbouncer"
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
	"RELATED_IMAGE_PGO_CLIENT":                CONTAINER_IMAGE_PGO_CLIENT,
	"RELATED_IMAGE_PGO_RMDATA":                CONTAINER_IMAGE_PGO_RMDATA,
	"RELATED_IMAGE_CRUNCHY_POSTGRES_EXPORTER": CONTAINER_IMAGE_CRUNCHY_POSTGRES_EXPORTER,
	"RELATED_IMAGE_CRUNCHY_PGADMIN":           CONTAINER_IMAGE_CRUNCHY_PGADMIN,
	"RELATED_IMAGE_CRUNCHY_PGBADGER":          CONTAINER_IMAGE_CRUNCHY_PGBADGER,
	"RELATED_IMAGE_CRUNCHY_PGBOUNCER":         CONTAINER_IMAGE_CRUNCHY_PGBOUNCER,
	"RELATED_IMAGE_CRUNCHY_POSTGRES_HA":       CONTAINER_IMAGE_CRUNCHY_POSTGRES_HA,
	"RELATED_IMAGE_CRUNCHY_POSTGRES_GIS_HA":   CONTAINER_IMAGE_CRUNCHY_POSTGRES_GIS_HA,
}
