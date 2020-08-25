ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG BASEOS
ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-backrest-repo-sync" \
	summary="Crunchy PostgreSQL Operator - pgBackRest Repo Sync" \
	description="Synchronizes the contents between two pgBackRest repositories."

RUN if [ "$BASEOS" = "centos7" ] ; then \
        ${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                crunchy-backrest-"${BACKREST_VERSION}" \
                openssh-clients \
                openssh-server \
                postgresql${PGVERSION}-server \
                procps-ng \
                psmisc \
                rsync \
                awscli \
        && ${PACKAGER} -y clean all ; \
fi

RUN if [ "$BASEOS" = "centos8" ] ; then \
	${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                crunchy-backrest-"${BACKREST_VERSION}" \
                openssh-clients \
                openssh-server \
                postgresql${PGVERSION}-server \
                procps-ng \
                psmisc \
                rsync \
                python3-pip \
	&& pip3 install --upgrade awscli \
	&& ${PACKAGER} -y clean all ; \
fi

RUN if [ "$BASEOS" = "rhel7" ] ; then \
	${PACKAGER} -y install \
		--setopt=skip_missing_names_on_install=False \
		--enablerepo=rhel-ha-for-rhel-7-server-rpms \
		crunchy-backrest-"${BACKREST_VERSION}" \
		openssh-clients \
		openssh-server \
		postgresql${PGVERSION}-server \
		procps-ng \
		psmisc \
		rsync \
		awscli \
	&& ${PACKAGER} -y --enablerepo=rhel-ha-for-rhel-7-server-rpms clean all ; \
fi

RUN if [ "$BASEOS" = "ubi7" ] ; then \
	${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                --enablerepo=rhel-ha-for-rhel-7-server-rpms \
                crunchy-backrest-"${BACKREST_VERSION}" \
                openssh-clients \
                openssh-server \
                postgresql${PGVERSION}-server \
                procps-ng \
                psmisc \
                rsync \
                awscli \
	&& ${PACKAGER} -y --enablerepo=rhel-ha-for-rhel-7-server-rpms clean all ; \
fi

RUN if [ "$BASEOS" = "ubi8" ] ; then \
	${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                crunchy-backrest-"${BACKREST_VERSION}" \
                openssh-clients \
                openssh-server \
                postgresql${PGVERSION}-server \
                procps-ng \
                psmisc \
                rsync \
                python3-pip \
	&& pip3 install --upgrade awscli \
	&& ${PACKAGER} -y clean all ; \
fi

RUN groupadd pgbackrest -g 2000 && useradd pgbackrest -u 2000 -g 2000
ADD bin/pgo-backrest-repo-sync/ /usr/local/bin
RUN chmod +x /usr/local/bin/pgo-backrest-repo-sync.sh && \
	mkdir -p /opt/cpm/bin /backrestrepo && \
	chown -R pgbackrest:pgbackrest /opt/cpm /backrestrepo

ADD bin/uid_pgbackrest.sh /opt/cpm/bin

RUN chmod g=u /etc/passwd && \
	chmod g=u /etc/group

RUN mkdir /.ssh && chown pgbackrest:pgbackrest /.ssh && chmod o+rwx /.ssh

VOLUME ["/sshd", "/backrestrepo"]

USER 2000

ENTRYPOINT ["/opt/cpm/bin/uid_pgbackrest.sh"]

CMD ["pgo-backrest-repo-sync.sh"]
