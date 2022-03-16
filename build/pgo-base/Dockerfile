ARG BASE_IMAGE_OS
ARG DOCKERBASEREGISTRY
FROM ${DOCKERBASEREGISTRY}${BASE_IMAGE_OS}

ARG BASEOS
ARG PGVERSION
ARG PG_FULL
ARG PACKAGER
ARG RELVER

MAINTAINER info@crunchydata.com

LABEL vendor="Crunchy Data" \
	url="https://crunchydata.com" \
	release="${RELVER}" \
	postgresql.version.major="${PGVERSION}" \
	postgresql.version="${PG_FULL}" \
	org.opencontainers.image.vendor="Crunchy Data" \
	io.openshift.tags="postgresql,postgres,sql,nosql,crunchy" \
	io.k8s.description="Trusted open source PostgreSQL-as-a-Service"

COPY redhat/atomic/help.1 /help.1
COPY redhat/atomic/help.md /help.md
COPY licenses /licenses

RUN ${PACKAGER} -y update && ${PACKAGER} -y clean all

# Create module file to disable postgres module, microdnf cannot do this with the current version
RUN if [ "$BASEOS" = "ubi8" ] ; then \
    echo "[postgresql]"  >> /etc/dnf/modules.d/postgresql.module \
    && echo "name=postgresql"  >> /etc/dnf/modules.d/postgresql.module \
    && echo "stream=10"  >> /etc/dnf/modules.d/postgresql.module \
    && echo "profiles="  >> /etc/dnf/modules.d/postgresql.module \
    && echo "state=disabled"  >> /etc/dnf/modules.d/postgresql.module ; \
fi

# Crunchy PostgreSQL repository
ADD conf/RPM-GPG-KEY-crunchydata* /
ADD conf/crunchypg${PGVERSION}.repo /etc/yum.repos.d/
RUN rpm --import RPM-GPG-KEY-crunchydata*
