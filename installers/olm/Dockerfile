FROM docker.io/library/centos:latest

RUN curl -Lo /usr/local/bin/jq -s "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64" \
 && chmod +x /usr/local/bin/jq \
 && sha256sum -c <<< "SHA256 (/usr/local/bin/jq) = af986793a515d500ab2d35f8d2aecd656e764504b789b66d7e1a0b727a124c44"

RUN curl -Lo /usr/local/bin/yq -s "https://github.com/mikefarah/yq/releases/download/2.4.1/yq_linux_amd64" \
 && chmod +x /usr/local/bin/yq \
 && sha256sum -c <<< "SHA256 (/usr/local/bin/yq) = 754c6e6a7ef92b00ef73b8b0bb1d76d651e04d26aa6c6625e272201afa889f8b"

RUN dnf update -d1 -y \
 && dnf install -d1 -y gettext glibc-langpack-en make ncurses python3 tree zip \
 && dnf clean all

ARG OLM_VERSION
RUN python3 -m pip install operator-courier \
 && curl -Lo /usr/local/bin/operator-sdk -s "https://github.com/operator-framework/operator-sdk/releases/download/v${OLM_VERSION}/operator-sdk-v${OLM_VERSION}-x86_64-linux-gnu" \
 && chmod +x /usr/local/bin/operator-sdk \
 && sha256sum -c <<< "SHA256 (/usr/local/bin/operator-sdk) = 18446d7f31ffcf059be22f767d66a7df585e81b596a3f3e91e3a8eba571eb927"

COPY --from=docker.io/bitnami/kubectl:1.11 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.11
COPY --from=docker.io/bitnami/kubectl:1.12 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.12
COPY --from=docker.io/bitnami/kubectl:1.13 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.13
COPY --from=docker.io/bitnami/kubectl:1.14 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.14
COPY --from=docker.io/bitnami/kubectl:1.15 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.15
COPY --from=docker.io/bitnami/kubectl:1.16 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.16
COPY --from=docker.io/bitnami/kubectl:1.17 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.17
