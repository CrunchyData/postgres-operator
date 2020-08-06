ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-sqlrunner" \
	summary="Crunchy PostgreSQL Operator - SQL Runner" \
	description="Crunchy PostgreSQL Operator - SQL Runner"

ENV PGROOT="/usr/pgsql-${PGVERSION}"

RUN if [ "$DFSET" = "centos" ] ; then \
	${PACKAGER} -y install epel-release \
	&& ${PACKAGER} -y install \
		--setopt=skip_missing_names_on_install=False \
		gettext \
		hostname  \
		nss_wrapper \
		procps-ng \
		postgresql${PGVERSION} \
	&& ${PACKAGER} -y clean all ; \
fi

RUN if [ "$DFSET" = "rhel" ] ; then \
	${PACKAGER} -y install \
		--setopt=skip_missing_names_on_install=False \
		postgresql${PGVERSION} \
	&& ${PACKAGER} -y clean all ; \
fi

RUN mkdir -p /opt/cpm/bin /opt/cpm/conf /pgconf \
 && chown -R 26:26 /opt/cpm /pgconf

ADD bin/pgo-sqlrunner /opt/cpm/bin

VOLUME ["/pgconf"]

USER 26

CMD ["/opt/cpm/bin/start.sh"]
