# Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

FROM docker.io/library/golang:bookworm AS build

COPY licenses /licenses
COPY hack/tools/queries /opt/crunchy/conf

WORKDIR /usr/src/app
COPY . .
ENV GOCACHE=/var/cache/go

# Build the operator and assemble the licenses
RUN --mount=type=cache,target=/var/cache/go go build ./cmd/postgres-operator
RUN go run ./hack/extract-licenses.go licenses postgres-operator 

FROM docker.io/library/debian:bookworm

COPY --from=build --chmod=0444 /usr/src/app/licenses /licenses
COPY --from=build --chmod=0444 /opt/crunchy/conf /opt/crunchy/conf
COPY --from=build /usr/src/app/postgres-operator /usr/local/bin

USER 2

CMD ["postgres-operator"]
