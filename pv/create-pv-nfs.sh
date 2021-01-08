#!/bin/bash
# Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# NOTE: this is script is intended for setting up development environments to
# use NFS as the persistent volume storage area. It is **not** intended for
# production.
#
# This script makes some assumptions, i.e:
#
# - You have sudo
# - You have your NFS filesystem mounted to the location you are running this
#   script
# - Your NFS filesystem is mounted to /nfsfileshare
# - Your PV names will be one of "crunchy-pvNNN" where NNN is a natural number
# - Your NFS UID:GID is "nfsnobody:nfsnobody", which correspunds to "65534:65534"
#
# And awaaaay we go...
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "create the test PV and PVC using the NFS dir"
for i in {1..160}
do
  PV_NAME="crunchy-pv${i}"
  NFS_PV_PATH="/nfsfileshare/${PV_NAME}"

  echo "deleting PV ${PV_NAME}"
	$PGO_CMD delete pv "${PV_NAME}"
  sudo rm -rf "${NFS_PV_PATH}"

  # this is the manifest used to create the persistent volumes
  MANIFEST=$(cat <<EOF
    {
      "apiVersion": "v1",
      "kind": "PersistentVolume",
      "metadata": {
        "name": "${PV_NAME}"
      },
      "spec": {
        "capacity": {
            "storage": "1Gi"
        },
        "accessModes": [ "ReadWriteOnce", "ReadWriteMany", "ReadOnlyMany" ],
        "nfs": {
            "path": "${NFS_PV_PATH}",
            "server": "${PGO_NFS_IP}"
        },
        "persistentVolumeReclaimPolicy": "Retain"
      }
    }
EOF
)

  # create the new directory and set the permissions
  sudo mkdir "${NFS_PV_PATH}"
  sudo chown 65534:65534 "${NFS_PV_PATH}"
  sudo chmod ugo=rwx "${NFS_PV_PATH}"

  # create the new persistent volume
  echo $MANIFEST | $PGO_CMD create -f -
done
