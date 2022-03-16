ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG BASEOS
ARG PGVERSION
ARG PACKAGER

LABEL name="crunchy-postgres-exporter" \
	summary="Metrics exporter for PostgreSQL" \
	description="When run with the crunchy-postgres family of containers, crunchy-postgres-exporter reads the PostgreSQL data directory and has a SQL interface to a database to allow for metrics collection." \
	io.k8s.description="Crunchy PostgreSQL Exporter" \
	io.k8s.display-name="Crunchy PostgreSQL Exporter" \
	io.openshift.tags="postgresql,postgres,monitoring,database,crunchy"

RUN if [ "$BASEOS" = "ubi8" ] ; then \
	${PACKAGER} install -y \
		findutils \
		postgresql${PGVERSION} \
    && ${PACKAGER} -y clean all ; \
fi

RUN mkdir -p /opt/cpm/bin /opt/cpm/conf

ADD postgres_exporter.tar.gz /opt/cpm/bin
ADD tools/pgmonitor/postgres_exporter/common /opt/cpm/conf
ADD tools/pgmonitor/postgres_exporter/linux /opt/cpm/conf
ADD bin/crunchy-postgres-exporter /opt/cpm/bin

RUN chgrp -R 0 /opt/cpm/bin /opt/cpm/conf && \
	chmod -R g=u /opt/cpm/bin/ opt/cpm/conf

# postgres_exporter
EXPOSE 9187

# The VOLUME directive must appear after all RUN directives to ensure the proper
# volume permissions are applied when building the image
VOLUME ["/conf"]

USER 2

CMD ["/opt/cpm/bin/start.sh"]
