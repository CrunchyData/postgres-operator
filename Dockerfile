# Copyright 2017 - 2026 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

FROM docker.io/library/golang:bookworm AS build

WORKDIR /usr/src/app
COPY . .

ENV GOCACHE=/var/cache/go
ENV GOMODCACHE=/var/cache/gomod
RUN --mount=type=cache,target=/var/cache \
<<-SHELL
set -e
go build ./cmd/postgres-operator
go run ./hack/extract-licenses.go licenses postgres-operator

find ./hack/tools/queries '(' -type d -exec chmod 0555 '{}' + ')' -o '(' -type f -exec chmod 0444 '{}' + ')'
find ./licenses '(' -type d -exec chmod 0555 '{}' + ')' -o '(' -type f -exec chmod 0444 '{}' + ')'
SHELL

FROM docker.io/library/debian:bookworm

COPY --from=build /usr/src/app/licenses /licenses
COPY --from=build /usr/src/app/hack/tools/queries /opt/crunchy/conf
COPY --from=build /usr/src/app/postgres-operator /usr/local/bin

USER 2

CMD ["postgres-operator"]
