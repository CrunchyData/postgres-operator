#!/bin/bash

set -e -u

REGISTRY='us.gcr.io/container-suite'
VERSION=${CO_IMAGE_TAG?}
IMAGES=(
pgo-lspvc
pgo-rmdata
pgo-load
postgres-operator
pgo-apiserver
pgo-backrest
)

function echo_green() {
    echo -e "\033[0;32m"
    echo "$1"
    echo -e "\033[0m"
}

gcloud auth login
gcloud config set project container-suite
gcloud auth configure-docker

for image in "${IMAGES[@]}"
do
    echo_green "=> Pulling ${REGISTRY?}/${image?}:${VERSION?}.."
    docker pull ${REGISTRY?}/${image?}:${VERSION?}
    docker tag ${REGISTRY?}/${image?}:${VERSION?} crunchydata/${image?}:${VERSION?}
done

echo_green "=> Done!"

exit 0
