FROM docker.io/library/centos:latest

RUN curl -Lo /usr/local/bin/jq -s "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64" \
 && chmod +x /usr/local/bin/jq \
 && sha256sum -c <<< "SHA256 (/usr/local/bin/jq) = af986793a515d500ab2d35f8d2aecd656e764504b789b66d7e1a0b727a124c44"

RUN dnf update -d1 -y \
 && dnf install -d1 -y gettext glibc-langpack-en make ncurses python3 tree zip \
 && dnf clean all

ARG OLM_SDK_VERSION
RUN python3 -m pip install operator-courier yq \
 && curl -Lo /usr/local/bin/operator-sdk -s "https://github.com/operator-framework/operator-sdk/releases/download/v${OLM_SDK_VERSION}/operator-sdk-v${OLM_SDK_VERSION}-x86_64-linux-gnu" \
 && chmod +x /usr/local/bin/operator-sdk \
 && sha256sum -c <<< "SHA256 (/usr/local/bin/operator-sdk) = 5c8c06bd8a0c47f359aa56f85fe4e3ee2066d4e51b60b75e131dec601b7b3cd6"

COPY --from=docker.io/bitnami/kubectl:1.11 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.11
COPY --from=docker.io/bitnami/kubectl:1.12 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.12
COPY --from=docker.io/bitnami/kubectl:1.13 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.13
COPY --from=docker.io/bitnami/kubectl:1.14 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.14
COPY --from=docker.io/bitnami/kubectl:1.15 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.15
COPY --from=docker.io/bitnami/kubectl:1.16 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.16
COPY --from=docker.io/bitnami/kubectl:1.17 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.17
COPY --from=docker.io/bitnami/kubectl:1.18 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.18
COPY --from=docker.io/bitnami/kubectl:1.19 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl-1.19
