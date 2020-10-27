ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-client" \
  summary="Crunchy PostgreSQL Operator - pgo client" \
  description="Crunchy PostgreSQL Operator - pgo client"

ADD bin/pgo /usr/local/bin

ENV PGO_APISERVER_URL=${PGO_APISERVER_URL}
ENV PGOUSERNAME=${PGOUSERNAME}
ENV PGOUSERPASS=${PGOUSERPASS}
ENV PGO_CA_CERT=${PGO_CA_CERT}
ENV PGO_CLIENT_CERT=${PGO_CLIENT_CERT}
ENV PGO_CLIENT_KEY=${PGO_CLIENT_KEY}

RUN chmod +x /usr/local/bin/pgo

USER 2

CMD tail -f /dev/null
