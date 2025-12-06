# image-exporter

## How to build

Run `make build` from within this folder. If you're at the root of the postgres-operator repo, run `make -C components/image-exporter build`.

The default build tool is `buildah`. To change the build tool, change the Makefile or pass a different tool on the command line with `BUILDAH=<build tool>`.

## Changing the postgres_exporter version

To set the postgres_exporter version, change the Dockerfile.

## Changing the postgres_exporter image tag

The default tag for this image is `localhost/crunchy-postgres-exporter:latest`. To change tag, change the Makefile.
