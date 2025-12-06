# image-pgbouncer

## How to build

Run `make build` from within this folder. If you're at the root of the postgres-operator repo, run `make -C components/image-pgbouncer build`.

The default build tool is `buildah`. To change the build tool, change the Makefile or pass a different tool on the command line with `BUILDAH=<build tool>`.

## Changing the pgbouncer version

To set the pgBouncer version, change the Dockerfile.

## Changing the pgbouncer image tag

The default tag for this image is `localhost/crunchy-pgbouncer:latest`. To change tag, change the Makefile.