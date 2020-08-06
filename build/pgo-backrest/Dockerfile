ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG PGVERSION
ARG BACKREST_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-backrest" \
    summary="Crunchy PostgreSQL Operator - pgBackRest" \
    description="pgBackRest image that is integrated for use with Crunchy Data's PostgreSQL Operator."

RUN ${PACKAGER} -y install \
	--setopt=skip_missing_names_on_install=False \
	postgresql${PGVERSION}-server \
	crunchy-backrest-"${BACKREST_VERSION}" \
	&& ${PACKAGER} -y clean all

RUN mkdir -p /opt/cpm/bin /pgdata /backrestrepo && chown -R 26:26 /opt/cpm
ADD bin/pgo-backrest/ /opt/cpm/bin
ADD bin/uid_postgres.sh /opt/cpm/bin

RUN chmod g=u /etc/passwd && \
        chmod g=u /etc/group

USER 26
ENTRYPOINT ["/opt/cpm/bin/uid_postgres.sh"]
VOLUME ["/pgdata","/backrestrepo"]
CMD ["/opt/cpm/bin/pgo-backrest"]
