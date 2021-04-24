ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG BASEOS
ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-scheduler" \
	summary="Crunchy PostgreSQL Operator - Scheduler" \
	description="Crunchy PostgreSQL Operator - Scheduler"

RUN if [ "$DFSET" = "centos" ] ; then \
	mkdir -p /opt/cpm/bin /opt/cpm/conf /configs \
	&& chown -R 2:2 /opt/cpm /configs \
	&& ${PACKAGER} -y install epel-release \
	&& ${PACKAGER} -y install \
		--setopt=skip_missing_names_on_install=False \
		gettext \
		hostname  \
		nss_wrapper \
		procps-ng \
	&& ${PACKAGER} -y clean all ; \
fi

RUN if [ "$DFSET" = "rhel" ] ; then \
	mkdir -p /opt/cpm/bin /opt/cpm/conf /pgo-config \
	&& chown -R 2:2 /opt/cpm /pgo-config ; \
fi

RUN if  [ "$BASEOS" = "ubi8" ]; then \
    ${PACKAGER} -y install \
        findutils \
        procps \
    && ${PACKAGER} -y clean all ; \
fi

ADD bin/pgo-scheduler /opt/cpm/bin
ADD installers/ansible/roles/pgo-operator/files/pgo-configs /default-pgo-config
ADD conf/postgres-operator/pgo.yaml /default-pgo-config/pgo.yaml

USER 2

CMD ["/opt/cpm/bin/start.sh"]
