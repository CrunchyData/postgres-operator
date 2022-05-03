ARG BASE_IMAGE_OS
ARG DOCKERBASEREGISTRY
FROM ${DOCKERBASEREGISTRY}${BASE_IMAGE_OS}

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
	io.k8s.description="Trusted open source PostgreSQL-as-a-Service" \
	name="postgres-operator" \
	summary="Crunchy PostgreSQL Operator" \
	description="Crunchy PostgreSQL Operator"

COPY licenses /licenses

RUN ${PACKAGER} -y update && ${PACKAGER} -y clean all

ADD bin/postgres-operator /usr/local/bin

USER 2

CMD ["postgres-operator"]
