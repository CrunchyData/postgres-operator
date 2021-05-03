ARG BASEOS
ARG BASEVER
ARG PREFIX
FROM ${PREFIX}/pgo-base:${BASEOS}-${BASEVER}

ARG BASEOS
ARG ANSIBLE_VERSION
ARG PACKAGER
ARG DFSET

LABEL name="pgo-deployer" \
    summary="Crunchy PostgreSQL Operator - Installer" \
    description="Crunchy PostgreSQL Operator - Installer"

COPY installers/image/conf/kubernetes.repo /etc/yum.repos.d/kubernetes.repo

RUN if [ "$DFSET" = "centos" ] ; then \
        ${PACKAGER} install -y epel-release \
        && ${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                kubectl \
                ansible-${ANSIBLE_VERSION} \
                which \
                gettext \
                nss_wrapper \
        && ${PACKAGER} -y clean all ; \
fi

RUN if [ "$BASEOS" = "rhel7" ] ; then \
        rm /etc/yum.repos.d/kubernetes.repo \
        && ${PACKAGER} install -y https://download.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
        && ${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                --enablerepo='rhel-7-server-ose-4.4-rpms' \
                openshift-clients \
                ansible-${ANSIBLE_VERSION} \
                which \
                gettext \
                nss_wrapper \
        && ${PACKAGER} -y clean all --enablerepo='rhel-7-server-ose-4.4-rpms' ; \
fi

RUN if [ "$BASEOS" = "ubi7" ] ; then \
	rm /etc/yum.repos.d/kubernetes.repo \
        && ${PACKAGER} install -y https://download.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
        && ${PACKAGER} -y install \
                --setopt=skip_missing_names_on_install=False \
                --enablerepo='rhel-7-server-ose-4.4-rpms' \
                openshift-clients \
                ansible-${ANSIBLE_VERSION} \
                which \
                gettext \
                nss_wrapper \
	&& ${PACKAGER} -y clean all --enablerepo='rhel-7-server-ose-4.4-rpms' ; \
fi

RUN if [ "$BASEOS" = "ubi8" ] ; then \
	rm /etc/yum.repos.d/kubernetes.repo \
        && rpm -ivh https://download.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm \
        && ${PACKAGER} -y install \
                --enablerepo='rhocp-4.5-for-rhel-8-x86_64-rpms' \
                openshift-clients \
                ansible-${ANSIBLE_VERSION} \
                which \
                gettext \
                nss_wrapper \
	&& ${PACKAGER} -y clean all --enablerepo='rhocp-4.5-for-rhel-8-x86_64-rpms' ; \
fi

RUN mkdir -p /opt/cpm/bin

COPY installers/ansible /ansible/postgres-operator
COPY installers/metrics/ansible /ansible/metrics
ADD tools/pgmonitor /opt/crunchy/pgmonitor
COPY installers/image/bin/pgo-deploy.sh /pgo-deploy.sh
ADD bin/common /opt/cpm/bin

ENV ANSIBLE_CONFIG="/ansible/postgres-operator/ansible.cfg"
ENV HOME="/tmp"

# Defines a unique directory name that will be utilized by the nss_wrapper in the UID script
ENV NSS_WRAPPER_SUBDIR="deployer"

ENTRYPOINT ["/opt/cpm/bin/uid_daemon.sh"]

USER 2

CMD ["/pgo-deploy.sh"]
