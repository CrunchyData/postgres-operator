ARG BASEOS
ARG BASEVER
ARG PREFIX
ARG PGVERSION
ARG BACKREST_VERSION
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG BASEOS
ARG PACKAGER

LABEL name="pgo-apiserver" \
	summary="Crunchy PostgreSQL Operator - Apiserver" \
	description="Crunchy PostgreSQL Operator - Apiserver"

RUN if [ "$BASEOS" = "ubi8" ] ; then \
	${PACKAGER} -y install \
	    postgresql${PGVERSION} \
	    hostname \
	&& ${PACKAGER} -y clean all ; \
else \
	${PACKAGER} -y install \
	    --setopt=skip_missing_names_on_install=False \
	    postgresql${PGVERSION} \
	    hostname \
	&& ${PACKAGER} -y clean all ; \
fi

ADD bin/apiserver /usr/local/bin
ADD installers/ansible/roles/pgo-operator/files/pgo-configs /default-pgo-config
ADD conf/postgres-operator/pgo.yaml /default-pgo-config/pgo.yaml

USER 2

ENTRYPOINT ["/usr/local/bin/apiserver"]
