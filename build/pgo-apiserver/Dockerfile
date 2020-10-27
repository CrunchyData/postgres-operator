ARG BASEOS
ARG BASEVER
ARG PREFIX
ARG PGVERSION
ARG BACKREST_VERSION
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

LABEL name="pgo-apiserver" \
	summary="Crunchy PostgreSQL Operator - Apiserver" \
	description="Crunchy PostgreSQL Operator - Apiserver"

RUN yum -y install \
	--setopt=skip_missing_names_on_install=False \
	postgresql${PGVERSION} \
	hostname \
	&& yum -y clean all

ADD bin/apiserver /usr/local/bin
ADD installers/ansible/roles/pgo-operator/files/pgo-configs /default-pgo-config
ADD conf/postgres-operator/pgo.yaml /default-pgo-config/pgo.yaml

USER 2

ENTRYPOINT ["/usr/local/bin/apiserver"]
