ARG BASE_IMAGE_OS
ARG DOCKERBASEREGISTRY
FROM ${DOCKERBASEREGISTRY}${BASE_IMAGE_OS}

ARG BASEOS
ARG RELVER
ARG PGVERSION
ARG PG_FULL
ARG PACKAGER
ARG DFSET

MAINTAINER info@crunchydata.com

LABEL vendor="Crunchy Data" \
	url="https://crunchydata.com" \
	release="${RELVER}" \
	postgresql.version.major="${PGVERSION}" \
	postgresql.version="${PG_FULL}" \
	os.version="7.7" \
	org.opencontainers.image.vendor="Crunchy Data" \
	io.openshift.tags="postgresql,postgres,sql,nosql,crunchy" \
	io.k8s.description="Trusted open source PostgreSQL-as-a-Service"

COPY redhat/atomic/help.1 /help.1
COPY redhat/atomic/help.md /help.md
COPY licenses /licenses

RUN if [ "$BASEOS" = "centos7" ]; then \
    ${PACKAGER} -y update \
    && ${PACKAGER} -y clean all ; \
fi

RUN if [ "$BASEOS" = "centos8" ]; then \
    ${PACKAGER} -y update \
    && ${PACKAGER} -y install \
        --setopt=skip_missing_names_on_install=False \
        glibc-langpack-en \
    && ${PACKAGER} -y clean all \
    && ${PACKAGER} -qy module disable postgresql ; \
fi

RUN if [ "$BASEOS" = "rhel7" ] ; then \
	${PACKAGER} -y --enablerepo=rhel-7-server-ose-3.11-rpms update \
	&& ${PACKAGER} -y --enablerepo=rhel-7-server-ose-3.11-rpms clean all ; \
fi

RUN if [ "$BASEOS" = "ubi7" ] ; then \
        ${PACKAGER} -y --enablerepo=rhel-7-server-ose-3.11-rpms update \
        && ${PACKAGER} -y --enablerepo=rhel-7-server-ose-3.11-rpms clean all ; \
fi

RUN if  [ "$BASEOS" = "ubi8" ]; then \
    ${PACKAGER} -y update \
    && ${PACKAGER} -y install \
        glibc-langpack-en \
        procps-ng \
        less \
        vim-minimal \
    && ${PACKAGER} reinstall tzdata -y \
    && ${PACKAGER} -y clean all ; \
fi

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
