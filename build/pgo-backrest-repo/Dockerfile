ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-backrest-repo" \
	summary="Crunchy PostgreSQL Operator - pgBackRest Repository" \
	description="Crunchy PostgreSQL Operator - pgBackRest Repository"

RUN ${PACKAGER} -y install \
	--setopt=skip_missing_names_on_install=False \
	crunchy-backrest-"${BACKREST_VERSION}" \
	hostname \
	openssh-clients \
	openssh-server \
	procps-ng \
	psmisc \
	rsync \
	&& ${PACKAGER} -y clean all

RUN groupadd pgbackrest -g 2000 && useradd pgbackrest -u 2000 -g 2000
ADD bin/pgo-backrest-repo /usr/local/bin
RUN chmod +x /usr/local/bin/pgo-backrest-repo.sh /usr/local/bin/archive-push-s3.sh \
	&& mkdir -p /opt/cpm/bin /etc/pgbackrest \
	&& chown -R pgbackrest:pgbackrest /opt/cpm \
	&& chown -R pgbackrest /etc/pgbackrest

ADD bin/uid_pgbackrest.sh /opt/cpm/bin

RUN chmod g=u /etc/passwd \
	&& chmod g=u /etc/group \
	&& chmod -R g=u /etc/pgbackrest \
	&& rm -f /run/nologin

RUN mkdir /.ssh && chown pgbackrest:pgbackrest /.ssh && chmod o+rwx /.ssh

USER 2000

ENTRYPOINT ["/opt/cpm/bin/uid_pgbackrest.sh"]
VOLUME ["/sshd", "/backrestrepo" ]

CMD ["pgo-backrest-repo.sh"]
