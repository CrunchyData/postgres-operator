#!/bin/bash -x

# Copyright 2019 Crunchy Data Solutions, Inc.
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

function trap_sigterm() {
	echo "Signal trap triggered, beginning shutdown.."
	killall sshd
}

trap 'trap_sigterm' SIGINT SIGTERM

# First enable sshd prior to running rsync if using pgbackrest with a repository 
# host
enable_sshd() {

	echo "PGBACKREST env vars are set to:"
	set | grep PGBACKREST

	SSHD_CONFIG=/sshd

	echo "SSHD_CONFIG is.."
	ls $SSHD_CONFIG

	mkdir ~/.ssh/
	cp $SSHD_CONFIG/config ~/.ssh/
	cp $SSHD_CONFIG/id_rsa /tmp
	chmod 400 /tmp/id_rsa ~/.ssh/config

	# start sshd which is used by pgbackrest for remote connections
	/usr/sbin/sshd -D -f $SSHD_CONFIG/sshd_config   &

	echo "sleep 5 secs to let sshd come up before running rsync command"
	sleep 5
}

# Runs rync to sync from a specified source directory to a target directory
rsync_repo() {
	echo "rsync pgbackrest from ${1} to ${2}"
	# note, the "/" after the repo path is important, as we do not want to sync
	# the top level directory
	rsync -a --progress "${1}"  "${2}"
	echo "finished rsync"
}

# Use the aws cli sync command to sync files from a source location to a target
# location.  The this inlcudes syncing files between who s3 locations, 
# syncing a local directory to s3, or syncing from s3 to a local directory.
aws_sync_repo() {

	export AWS_CA_BUNDLE="${PGBACKREST_REPO1_S3_CA_FILE}"
	export AWS_ACCESS_KEY_ID="${PGBACKREST_REPO1_S3_KEY}"
	export AWS_SECRET_ACCESS_KEY="${PGBACKREST_REPO1_S3_KEY_SECRET}"
	export AWS_DEFAULT_REGION="${PGBACKREST_REPO1_S3_REGION}"

	echo "Executing aws s3 sync from source ${1} to target ${2}"
	aws s3 sync "${1}" "${2}"
	echo "Finished aws s3 sync"
}

# If s3 is identifed as the data source, then the aws cli will be utilized to
# sync the repo to the target location in s3.  If local storage is also enabled
# (along with s3) for the cluster, then also use the aws cli to sync the repo
# from s3 to the target volume locally.
#
# If the data source is local (the default if not specified at all), then first
# rsync the repo to the target directory locally.  Then, if s3 storage is also
# enabled (along with local), use the aws cli to sync the local repo to the
# target s3 location.
if [[ "${BACKREST_STORAGE_SOURCE}" == "s3" ]]
then
	aws_source="s3://${PGBACKREST_REPO1_S3_BUCKET}${PGBACKREST_REPO1_PATH}/"
	aws_target="s3://${PGBACKREST_REPO1_S3_BUCKET}${NEW_PGBACKREST_REPO}/"
	aws_sync_repo "${aws_source}" "${aws_target}"
	if [[ "${PGHA_PGBACKREST_LOCAL_S3_STORAGE}" == "true" ]]
	then
		aws_source="s3://${PGBACKREST_REPO1_S3_BUCKET}${PGBACKREST_REPO1_PATH}/"
		aws_target="${NEW_PGBACKREST_REPO}/"
		aws_sync_repo "${aws_source}" "${aws_target}"
	fi
else
	enable_sshd # enable sshd for rsync
	
	rsync_source="${PGBACKREST_REPO1_HOST}:${PGBACKREST_REPO1_PATH}/"
	rsync_target="$NEW_PGBACKREST_REPO"
	rsync_repo "${rsync_source}" "${rsync_target}"
	if [[ "${PGHA_PGBACKREST_LOCAL_S3_STORAGE}" == "true" ]]
	then
		aws_source="${NEW_PGBACKREST_REPO}/"
		aws_target="s3://${PGBACKREST_REPO1_S3_BUCKET}${NEW_PGBACKREST_REPO}/"
		aws_sync_repo "${aws_source}" "${aws_target}"
	fi
fi
