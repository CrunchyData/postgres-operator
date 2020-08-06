ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-backrest-restore" \
	summary="Crunchy PostgreSQL Operator - pgBackRest Restore" \
	description="Performs a restore operation for a PostgreSQL database using pgBackRest."

RUN ${PACKAGER} -y install \
	--setopt=skip_missing_names_on_install=False \
	crunchy-backrest-"${BACKREST_VERSION}" \
	openssh-clients \
	openssh-server \
	postgresql${PGVERSION}-server \
	procps-ng \
	psmisc \
	&& ${PACKAGER} -y clean all

RUN mkdir -p /opt/cpm/bin /pgdata /tablespaces && \
		chown -R 26:26 /opt/cpm /tablespaces

ADD bin/pgo-backrest-restore/ /opt/cpm/bin
ADD bin/uid_postgres.sh /opt/cpm/bin

RUN chmod g=u /etc/passwd && \
        chmod g=u /etc/group

RUN mkdir /.ssh && chown 26:0 /.ssh && chmod g+rwx /.ssh

VOLUME ["/sshd", "/pgdata"]

ENTRYPOINT ["/opt/cpm/bin/uid_postgres.sh"]

USER 26

CMD ["/opt/cpm/bin/pgo-backrest-restore.sh"]
