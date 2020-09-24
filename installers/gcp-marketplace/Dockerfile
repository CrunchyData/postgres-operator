ARG MARKETPLACE_VERSION
FROM gcr.io/cloud-marketplace-tools/k8s/deployer_envsubst:${MARKETPLACE_VERSION} AS build

# Verify Bash (>= 4.3) has `wait -n`
RUN bash -c 'echo -n & wait -n'


FROM gcr.io/cloud-marketplace-tools/k8s/deployer_envsubst:${MARKETPLACE_VERSION}

RUN install -D /bin/create_manifests.sh /opt/postgres-operator/cloud-marketplace-tools/bin/create_manifests.sh

# https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html#installing-ansible-on-debian
RUN if [ -f /etc/os-release ] && [ debian = "$(. /etc/os-release; echo $ID)" ] && [ 10 -ge "$(. /etc/os-release; echo $VERSION_ID)" ]; then \
      apt-get update && apt-get install -y --no-install-recommends gnupg && rm -rf /var/lib/apt/lists/* && \
      wget -qO- 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x93C4A3FD7BB9C367' | apt-key add && \
      echo > /etc/apt/sources.list.d/ansible.list deb http://ppa.launchpad.net/ansible/ansible-2.9/ubuntu trusty main ; \
    fi

RUN apt-get update \
 && apt-get install -y --no-install-recommends ansible=2.9.* openssh-client \
 && rm -rf /var/lib/apt/lists/*

COPY installers/ansible/* \
     /opt/postgres-operator/ansible/
COPY installers/favicon.png \
     installers/gcp-marketplace/install-job.yaml \
     installers/gcp-marketplace/install.sh \
     installers/gcp-marketplace/values.yaml \
     /opt/postgres-operator/

COPY installers/gcp-marketplace/install-hook.sh \
     /bin/create_manifests.sh
COPY installers/gcp-marketplace/schema.yaml \
     /data/
COPY installers/gcp-marketplace/application.yaml \
     /data/manifest/
COPY installers/gcp-marketplace/test-pod.yaml \
     /data-test/manifest/

ARG PGO_VERSION
RUN for file in \
      /data/schema.yaml \
      /data/manifest/application.yaml \
    ; do envsubst '$PGO_VERSION' < "$file" > /tmp/sponge && mv /tmp/sponge "$file" ; done
