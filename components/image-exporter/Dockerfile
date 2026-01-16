# Copyright 2025 - 2026 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

FROM registry.access.redhat.com/ubi9/go-toolset AS build

# NOTE: The TARGETARCH and TARGETOS arguments are populated automatically by Buildah and Docker.
# https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETARCH
ARG TARGETOS

# Run as user 0 to allow later chmod'ing
USER 0

# The hack/extract-licenses.go file is accessible from
# the root of this repo because we specify a second build-context;
# see the Makefile#build target.
COPY --chmod=0777 hack/extract-licenses.go .

# Get pre-built exporter binary from github releases and retrieve licenses
# using our extract-licenses.go func, which will create the licenses folder.
# Note: The postgres_exporter image does not have the LICENSE, which we want
# to include, so we pull from the GitHub release.
RUN \
<<-SHELL
set -e

wget https://github.com/prometheus-community/postgres_exporter/releases/download/v0.18.1/postgres_exporter-0.18.1.${TARGETOS}-${TARGETARCH}.tar.gz
tar -xzf postgres_exporter-0.18.1.${TARGETOS}-${TARGETARCH}.tar.gz
go run ./extract-licenses.go ./licenses postgres_exporter-0.18.1.${TARGETOS}-${TARGETARCH}/postgres_exporter
SHELL

COPY licenses/LICENSE.txt licenses/LICENSE.txt

# Make sure all licenses are readable by owner:group:other
RUN \
<<-SHELL
set -e

mv ./postgres_exporter-0.18.1.${TARGETOS}-${TARGETARCH}/LICENSE ./licenses
find ./licenses '(' -type d -exec chmod 0555 '{}' + ')'
find ./licenses '(' -type f -exec chmod 0444 '{}' + ')'
SHELL

# Assemble PGO-usable exporter image with licenses.
# License permissions have been set in build, but ensure
# that exporter binary is executable by owner:group:other.
FROM registry.access.redhat.com/ubi9/ubi-minimal

COPY --from=build --chmod=0555 \
    /opt/app-root/src/postgres_exporter-*/postgres_exporter \
    /usr/local/bin/postgres_exporter
COPY --from=build /opt/app-root/src/licenses/ licenses/

USER 2
