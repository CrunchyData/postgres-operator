# Copyright 2017 - 2026 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

FROM registry.access.redhat.com/ubi9/ubi-minimal

ARG PGBOUNCER_VERSION=1.24.1

# The c-ares package is in the rhel-9-for-$(arch)-baseos-rpms repo, so we must enable it
RUN rpm -ivh "https://download.postgresql.org/pub/repos/yum/reporpms/EL-9-$(arch)/pgdg-redhat-repo-latest.noarch.rpm" \
 && microdnf install -y --nodocs --setopt install_weak_deps=0 \
 --enablerepo="rhel-9-for-$(arch)-baseos-rpms" \
 "pgbouncer-${PGBOUNCER_VERSION}" \
 'procps-ng' \
 'tar' \
 && microdnf clean all \
 && rm -f /etc/yum.repos.d/redhat.repo

USER 2
