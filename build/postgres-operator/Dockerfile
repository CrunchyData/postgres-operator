ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

LABEL name="postgres-operator" \
	summary="Crunchy PostgreSQL Operator" \
	description="Crunchy PostgreSQL Operator"

ADD bin/postgres-operator /usr/local/bin

USER 2

CMD ["postgres-operator"]
