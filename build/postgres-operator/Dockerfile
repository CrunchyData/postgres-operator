ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="postgres-operator" \
	summary="Crunchy PostgreSQL Operator" \
	description="Crunchy PostgreSQL Operator"

RUN if [ "$DFSET" = "centos" ] ; then \
        ${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
		hostname \
                postgresql${PGVERSION} \
        && ${PACKAGER} -y clean all ; \
fi

RUN if [ "$DFSET" = "rhel" ] ; then \
	${PACKAGER} -y install \
		--setopt=skip_missing_names_on_install=False \
		postgresql${PGVERSION} \
	&& ${PACKAGER} -y clean all ; \
fi

ADD bin/postgres-operator /usr/local/bin
ADD installers/ansible/roles/pgo-operator/files/pgo-configs /default-pgo-config
ADD conf/postgres-operator/pgo.yaml /default-pgo-config/pgo.yaml

USER 2

ENTRYPOINT ["postgres-operator"]
