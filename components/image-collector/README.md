# image-collector

An image used in the `collector` container to export metrics and manage log retention/rotation.

## How to build

Run `make build` from within this folder. If you're at the root of the postgres-operator repo, run `make -C components/image-collector build`.

The default build tool is `buildah`. To change the build tool, change the Makefile or pass a different tool on the command line with `BUILDAH=<build tool>`.

## Changing the collector version

To set the collector version, change the Dockerfile.

## Changing the collector image tag

The default tag for this image is `localhost/crunchy-collector:latest`. To change tag, change the Makefile.
