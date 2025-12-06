# image-pgadmin

## How to build

Run `make build` from within this folder. If you're at the root of the postgres-operator repo, run `make -C components/image-pgadmin build`.

The default build tool is `buildah`. To change the build tool, change the Makefile or pass a different tool on the command line with `BUILDAH=<build tool>`.

## Changing the pgAdmin version

To set the pgAdmin version, change the Dockerfile.

## Changing the pgAdmin image tag

The default tag for this image is `localhost/crunchy-pgadmin:latest`. To change tag, change the Makefile.