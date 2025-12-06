# image-postgres

## How to build

Run `make all` from within this folder. If you're at the root of the postgres-operator repo, run `make -C components/image-postgres all`.
This will build a Postgres image, a Postgres with PostGIS image and a Postgres upgrade image.
To build one of those images individually, use `make build-postgres`, `make build-postgis` or `make build-postgres-upgrade`, respectively.

The default build tool is `buildah`. To change the build tool, change the Makefile or pass a different tool on the command line with `BUILDAH=<build tool>`.

## Changing the postgres or postgis versions

To set a different Postgres or PostGIS version, change the Dockerfile.

## Changing the postgres image tag

The default tag for these images are

`localhost/crunchy-postgres:latest`
`localhost/crunchy-postgres-gis:latest`
`localhost/crunchy-upgrade:latest`

 To change the tags, change the Makefile.