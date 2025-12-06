# Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

ARG BASE_VERSION=ubi9
ARG PGADMIN4_VERSION=9.8
ARG PYTHON_VERSION=3.11
ARG PG_MAJOR=18

# Some Python packages available in UBI repositories are too old for pgAdmin
# so we must compile them. Do that in stages to keep build tooling out of
# the final image.
FROM registry.access.redhat.com/ubi9/ubi-minimal AS build-tools

ARG PYTHON_VERSION
ENV PIP_ROOT_USER_ACTION=ignore
RUN \
microdnf install -y --nodocs --setopt install_weak_deps=0 \
'findutils' \
'gcc' \
'krb5-devel' \
"python${PYTHON_VERSION}-devel" \
"python${PYTHON_VERSION}-packaging" \
"python${PYTHON_VERSION}-pip" \
'tar' \
&& microdnf clean all

# We need to build the "gssapi" package for pgAdmin's Kerberos integration.
#
#  1. Install pgAdmin without any dependencies
#  2. Inspect the pgAdmin package and find:
#      - dependencies for the "kerberos" extra
#      - large dependencies, according to `du -sh /usr/local/*/python*/site-packages/* | sort -h`
#  3. Build the "kerberos" dependencies
#  4. Copy their files to /mnt
#
FROM build-tools AS build1
ARG PGADMIN4_VERSION
ARG PYTHON_VERSION
RUN \
"python${PYTHON_VERSION}" -m pip install --no-cache-dir --no-deps \
"pgadmin4==${PGADMIN4_VERSION}" \
wheel \
packaging \
&& "python${PYTHON_VERSION}" -c "$(printf '%s\n' \
'from importlib.metadata import requires' \
'from packaging.markers import default_environment' \
'from packaging.requirements import Requirement' \
\
'env = default_environment() | dict(extra="kerberos")' \
'for s in requires("pgadmin4"):' \
'  r = Requirement(s)' \
\
'  if r.marker and r.marker.evaluate(env) and "kerberos" in str(r.marker):' \
'    r.marker = None' \
'    with open("/pgadmin.kerberos.txt", mode="a") as f:' \
'      print(r, file=f)' \
\
'  if any(s.startswith(p) for p in ["azure", "boto", "google", "urllib"]):' \
'    with open("/pgadmin.large.txt", mode="a") as f:' \
'      print(s, file=f)' \
)" \
&& "python${PYTHON_VERSION}" -m pip install --no-cache-dir --requirement /pgadmin.kerberos.txt \
&& tar --create /usr/local | tar --extract --verbose --directory /mnt

# In a separate stage, build and copy the "large" dependencies we identified earlier.
FROM build-tools AS build2
ARG PYTHON_VERSION
COPY --from=build1 /pgadmin.*.txt /
RUN \
"python${PYTHON_VERSION}" -m pip install --no-cache-dir --requirement /pgadmin.large.txt \
&& tar --create /usr/local | tar --extract --verbose --directory /mnt


# Installed, pgAdmin and its dependencies are more than 700 MiB.
# A single layer this size can be difficult to distribute.
#
# - Cloudflare CDN has a default limit of 500 MiB:
#   https://developers.cloudflare.com/cache/concepts/default-cache-behavior/#upload-limits
#
# Start by copying files from prior stages.
FROM ubi-minimal

# These include pgAdmin itself; ~180 MiB
COPY --from=build1 /mnt/usr/local /usr/local

# These are pgAdmin dependencies; ~200 MiB
COPY --from=build2 /mnt/usr/local /usr/local

# Local files
COPY --chmod=0444 conf/config_system.py /etc/pgadmin/config_system.py

# Next, install system packages; ~150 MiB
ARG PG_MAJOR
ARG PYTHON_VERSION
ARG BASE_VERSION

RUN rpm -ivh "https://dl.fedoraproject.org/pub/epel/epel-release-latest-${BASE_VERSION##ubi}.noarch.rpm" \
&& rpm -ivh "https://download.postgresql.org/pub/repos/yum/reporpms/EL-${BASE_VERSION##ubi}-$(arch)/pgdg-redhat-repo-latest.noarch.rpm" \
&& microdnf install -y --nodocs --setopt install_weak_deps=0 \
--enablerepo='epel' \
'krb5-libs' \
"postgresql${PG_MAJOR}" \
"python${PYTHON_VERSION}-pip" \
'tar' \
&& microdnf clean all \
&& rm -f /etc/yum.repos.d/redhat.repo

# Finally, install the webserver and any remaining pgAdmin dependencies; ~180 MiB.
#
# pgAdmin needs the "setuptools" package but only lists it as a dependency for
# recent Python versions because, prior to Python 3.12, it was automatically
# included in virtual environments.
#
# - https://docs.python.org/3/whatsnew/3.12.html
# - https://github.com/pgadmin-org/pgadmin4/blob/REL-8_14/requirements.txt#L57
ARG PGADMIN4_VERSION
RUN \
microdnf install -y --nodocs \
"python${PYTHON_VERSION}-pip" \
"python${PYTHON_VERSION}-setuptools" \
&& microdnf clean all \
&& rm -f /etc/yum.repos.d/redhat.repo \
&& "python${PYTHON_VERSION}" -m pip install --no-cache-dir \
'gunicorn' \
"pgadmin4[kerberos]==${PGADMIN4_VERSION}" \
&& rm -rf /root/.cache \
&& update-alternatives --install /usr/bin/python3 python3 "/usr/bin/python${PYTHON_VERSION}" 1 \
&& update-alternatives --set python3 "/usr/bin/python${PYTHON_VERSION}"

# config_local.py should be colocated with config.py
COPY --chmod=0444 conf/config_local.py \
    /usr/local/lib/python${PYTHON_VERSION}/site-packages/pgadmin4/config_local.py

USER 2
