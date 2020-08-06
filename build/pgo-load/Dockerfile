ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-load" \
	summary="Crunchy PostgreSQL Operator - Load Data" \
	description="Loads a CSV or JSON file into the database."

RUN if [ "$DFSET" = "centos" ] ; then \
	yum -y install epel-release \
	&& yum install -y \
		--setopt=skip_missing_names_on_install=False \
		gettext \
		hostname \
		nss_wrapper \
		openssh-clients \
		procps-ng \
		postgresql${PGVERSION} \
	&& yum -y clean all ; \
fi

RUN if [ "$DFSET" = "rhel" ] ; then \
	${PACKAGER} install -y \
 		--setopt=skip_missing_names_on_install=False \
		gettext \
		hostname \
		procps-ng \
		postgresql${PGVERSION} \
	&& ${PACKAGER} -y clean all ; \
fi

RUN mkdir -p /opt/cpm/bin /opt/cpm/conf
ADD bin/pgo-load/ /opt/cpm/bin
ADD bin/uid_postgres.sh /opt/cpm/bin
ADD conf/pgo-load/ /opt/cpm/conf
RUN chown -R 26:26 /opt/cpm

RUN chmod g=u /etc/passwd && \
        chmod g=u /etc/group

VOLUME /pgdata

ENTRYPOINT ["/opt/cpm/bin/uid_postgres.sh"]

USER 26

CMD ["/opt/cpm/bin/start.sh"]
