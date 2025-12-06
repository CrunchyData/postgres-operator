# Copyright 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

# Rather than build the binary, retrieve the already-built binary from
# the OpenTelemetry image
FROM otel/opentelemetry-collector-contrib:0.139.0 AS collector

# Aggregate the collector licenses from binary
# and from root of the PGO repo
FROM registry.access.redhat.com/ubi9/go-toolset AS build

# Run as user 0 to allow later chmod'ing
USER 0

COPY --from=collector --chmod=0777 /otelcol-contrib /otelcol-contrib
COPY hack/extract-licenses.go .
RUN go run ./extract-licenses.go /licenses /otelcol-contrib 
COPY licenses/LICENSE.txt /licenses/LICENSE.txt

# Make sure all licenses are readable by owner:group:other
RUN \
<<-SHELL
set -e

find /licenses '(' -type d -exec chmod 0555 '{}' + ')'
find /licenses '(' -type f -exec chmod 0444 '{}' + ')'
SHELL

# Assemble the complete image with required packages for OpenTelemetry
# and related activity (e.g., log retention, process restarting)
FROM registry.access.redhat.com/ubi9/ubi-minimal

COPY --from=build --chmod=0777 /otelcol-contrib /otelcol-contrib
COPY --from=build /licenses /licenses

RUN microdnf install -y 'logrotate' 'procps-ng'

USER 2
