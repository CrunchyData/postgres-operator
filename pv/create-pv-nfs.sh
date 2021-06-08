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
# By default this script makes some assumptions, i.e:
#
# - You have sudo
# - You have your NFS filesystem mounted to the location you are running this
#   script
# - Your NFS filesystem is mounted to /nfsfileshare
# - Your PV names will be one of "crunchy-pvNNN" where NNN is a natural number
# - Your NFS UID:GID is "nfsnobody:nfsnobody", which correspunds to "65534:65534"
#
# If you want to modify this script defaults, execute the script with -h argument to see the usage help
# And awaaaay we go...

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PGO_NFS_IP=$(ip -o route get to 8.8.8.8 | sed -n 's/.*src \([0-9.]\+\).*/\1/p')
PV_NAME_INPUT="crunchy-pv"
OLD_PV_NAME_INPUT="crunchy-pv"
SCRIPT_NAME="$(basename "$(test -L "$0" && readlink "$0" || echo "$0")")"
PV_COUNT=10
NFS_MOUNT_PATH='/nfsfileshare'

usage(){
    echo "Usage: ./${SCRIPT_NAME} <options>"
    echo "   
  Options :
    -n|--name                         persistent volume name. default is crunchy-pv
    -v|--old-name                     old persisten volume name. Useful when you want to delete old PVs and use a new name.
                                      default is crunchy-pv
    -c|--count                        PV count. how many PVs to create. default is 10
    -m|--nfs-mount                    nfs mount path. default is /nfsfileshare
    -i|--nfs-ip                       nfs ip. default is ${PGO_NFS_IP}
    -h|--help                         show usage
    "
    exit 1

}

if [[ $EUID > 0 ]] 
  then echo "ERROR: Please run as root or use sudo"
  usage
fi

opts=$(getopt \
    -o n:o:c:hm:i: \
    --long 'name:,old-name:,count:,help,nfs-mount:,nfs-ip: ' \
    --name "${SCRIPT_NAME}" \
    -- "$@"
)

eval set --$opts
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            ;;
        -m|--nfs-mount)
            NFS_MOUNT_PATH=$2
            shift 2
            ;;
        -n|--name)
            PV_NAME_INPUT=$2
            OLD_PV_NAME_INPUT=$2
            shift 2
            ;;
        -o|--old-name)
            OLD_PV_NAME_INPUT=$2
            shift 2
            ;;
        -c|--count)
            PV_COUNT=$2
            shift 2
            ;;
        -i|--nfs-ip)
            PGO_NFS_IP=$2
            shift 2
            ;;
        *)
            break
            ;;
    esac
done


PGO_CMD="kubectl"

echo "create the test PV and PVC using the NFS dir"
for i in $(seq 1 $PV_COUNT)
do
  PV_NAME="${PV_NAME_INPUT}${i}"
  OLD_PV_NAME="${OLD_PV_NAME_INPUT}${i}"
  NFS_PV_PATH="${NFS_MOUNT_PATH}/${PV_NAME}"
  
  echo "deleting PV ${OLD_PV_NAME}"
	$PGO_CMD delete pv "${OLD_PV_NAME}"
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
  echo "creating PV ${PV_NAME}"
  echo $MANIFEST | $PGO_CMD create -f -
done
