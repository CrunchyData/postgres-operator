ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-event" \
    summary="Crunchy PostgreSQL Operator - pgo-event" \
    description="Crunchy PostgreSQL Operator - pgo-event"

ADD bin/pgo-event /usr/local/bin

USER 2

ENTRYPOINT ["/usr/local/bin/pgo-event.sh"]
