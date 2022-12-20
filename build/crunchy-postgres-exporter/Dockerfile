FROM registry.access.redhat.com/ubi8/ubi-minimal

ARG PGVERSION

COPY licenses /licenses

# Crunchy PostgreSQL repository
COPY conf/*KEY* /
COPY conf/crunchypg${PGVERSION}.repo /etc/yum.repos.d/
RUN rpm --import ./*GPG-KEY-crunchydata*

RUN { microdnf -y module disable postgresql || true; } \
	&& microdnf -y update \
	&& microdnf install -y \
		findutils \
		postgresql${PGVERSION} \
    && microdnf -y clean all

RUN mkdir -p /opt/cpm/bin /opt/cpm/conf

# Add will extract the exporter into the target directory
ADD hack/tools/postgres_exporter.tar.gz /opt/cpm/bin

COPY hack/tools/pgmonitor/postgres_exporter/common /opt/cpm/conf
COPY hack/tools/pgmonitor/postgres_exporter/linux /opt/cpm/conf
COPY bin/crunchy-postgres-exporter /opt/cpm/bin

RUN chgrp -R 0 /opt/cpm/bin /opt/cpm/conf && \
	chmod -R g=u /opt/cpm/bin/ opt/cpm/conf

# postgres_exporter
EXPOSE 9187

# The VOLUME directive must appear after all RUN directives to ensure the proper
# volume permissions are applied when building the image
VOLUME ["/conf"]

USER 2

CMD ["/opt/cpm/bin/start.sh"]
