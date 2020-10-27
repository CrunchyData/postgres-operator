#!/bin/bash

# Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

source /opt/cpm/bin/common_lib.sh
enable_debugging

export PG_EXP_HOME=$(find /opt/cpm/bin/ -type d -name 'postgres_exporter*')
export PG_DIR=$(find /usr/ -type d -name 'pgsql-*')
POSTGRES_EXPORTER_PIDFILE=/tmp/postgres_exporter.pid
CONFIG_DIR='/opt/cpm/conf'
QUERIES=(
    queries_backrest
    queries_common
    queries_per_db
    queries_nodemx
)

function trap_sigterm() {
    echo_info "Doing trap logic.."

    echo_warn "Clean shutdown of postgres-exporter.."
    kill -SIGINT $(head -1 ${POSTGRES_EXPORTER_PIDFILE?})
}

# Set default env vars for the postgres exporter container
set_default_postgres_exporter_env() {
    if [[ ! -v POSTGRES_EXPORTER_PORT ]]
    then
        export POSTGRES_EXPORTER_PORT="9187"
        default_exporter_env_vars+=("POSTGRES_EXPORTER_PORT=${POSTGRES_EXPORTER_PORT}")
    fi
}

# Set default PG env vars for the exporter container
set_default_pg_exporter_env() {

    if [[ ! -v EXPORTER_PG_HOST ]]
    then
        export EXPORTER_PG_HOST="127.0.0.1"
        default_exporter_env_vars+=("EXPORTER_PG_HOST=${EXPORTER_PG_HOST}")
    fi

    if [[ ! -v EXPORTER_PG_PORT ]]
    then
        export EXPORTER_PG_PORT="5432"
        default_exporter_env_vars+=("EXPORTER_PG_PORT=${EXPORTER_PG_PORT}")
    fi

    if [[ ! -v EXPORTER_PG_DATABASE ]]
    then
        export EXPORTER_PG_DATABASE="postgres"
        default_exporter_env_vars+=("EXPORTER_PG_DATABASE=${EXPORTER_PG_DATABASE}")
    fi

    if [[ ! -v EXPORTER_PG_USER ]]
    then
        export EXPORTER_PG_USER="ccp_monitoring"
        default_exporter_env_vars+=("EXPORTER_PG_USER=${EXPORTER_PG_USER}")
    fi

    env_check_err "EXPORTER_PG_PASSWORD"
}

trap 'trap_sigterm' SIGINT SIGTERM

set_default_postgres_exporter_env

if [[ ! -v DATA_SOURCE_NAME ]]
then
    set_default_pg_exporter_env
    if [[ ! -z "${EXPORTER_PG_PARAMS}" ]]
    then
        EXPORTER_PG_PARAMS="?${EXPORTER_PG_PARAMS}"
    fi
    export DATA_SOURCE_NAME="postgresql://${EXPORTER_PG_USER}:${EXPORTER_PG_PASSWORD}\
@${EXPORTER_PG_HOST}:${EXPORTER_PG_PORT}/${EXPORTER_PG_DATABASE}${EXPORTER_PG_PARAMS}"
fi



if [[ ! ${#default_exporter_env_vars[@]} -eq 0 ]]
then
    echo_info "Defaults have been set for the following exporter env vars:"
    echo_info "[${default_exporter_env_vars[*]}]"
fi

# Check that postgres is accepting connections.
echo_info "Waiting for PostgreSQL to be ready.."
while true; do
    ${PG_DIR?}/bin/pg_isready -d ${DATA_SOURCE_NAME}
    if [ $? -eq 0 ]; then
        break
    fi
    sleep 2
done

echo_info "Checking if PostgreSQL is accepting queries.."
while true; do
    ${PG_DIR?}/bin/psql "${DATA_SOURCE_NAME}" -c "SELECT now();"
    if [ $? -eq 0 ]; then
        break
    fi
    sleep 2
done

if [[ -f /conf/queries.yml ]]
then
    echo_info "Custom queries configuration detected.."
    QUERY_DIR='/conf'
else
    echo_info "No custom queries detected. Applying default configuration.."
    QUERY_DIR='/tmp'

    touch ${QUERY_DIR?}/queries.yml && > ${QUERY_DIR?}/queries.yml
    for query in "${QUERIES[@]}"
    do
        if [[ -f ${CONFIG_DIR?}/${query?}.yml ]]
        then
            cat ${CONFIG_DIR?}/${query?}.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file ${query?}.yml does not exist (it should).."
            exit 1
        fi
    done

    VERSION=$(${PG_DIR?}/bin/psql "${DATA_SOURCE_NAME}" -qtAX -c "SELECT current_setting('server_version_num')")
    if (( ${VERSION?} >= 90500 )) && (( ${VERSION?} < 90600 ))
    then
        if [[ -f ${CONFIG_DIR?}/queries_pg95.yml ]]
        then
            cat ${CONFIG_DIR?}/queries_pg95.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file queries_pg95.yml does not exist (it should).."
        fi
    elif (( ${VERSION?} >= 90600 )) && (( ${VERSION?} < 100000 ))
    then
        if [[ -f ${CONFIG_DIR?}/queries_pg96.yml ]]
        then
            cat ${CONFIG_DIR?}/queries_pg96.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file queries_pg96.yml does not exist (it should).."
        fi
    elif (( ${VERSION?} >= 100000 )) && (( ${VERSION?} < 110000 ))
    then
        if [[ -f ${CONFIG_DIR?}/queries_pg10.yml ]]
        then
            cat ${CONFIG_DIR?}/queries_pg10.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file queries_pg10.yml does not exist (it should).."
        fi
    elif (( ${VERSION?} >= 110000 )) && (( ${VERSION?} < 120000 ))
    then
        if [[ -f ${CONFIG_DIR?}/queries_pg11.yml ]]
        then
            cat ${CONFIG_DIR?}/queries_pg11.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file queries_pg11.yml does not exist (it should).."
        fi
    elif (( ${VERSION?} >= 120000 )) && (( ${VERSION?} < 130000 ))
    then
        if [[ -f ${CONFIG_DIR?}/queries_pg12.yml ]]
        then
            cat ${CONFIG_DIR?}/queries_pg12.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file queries_pg12.yml does not exist (it should).."
        fi
        elif (( ${VERSION?} >= 130000 ))
    then
        if [[ -f ${CONFIG_DIR?}/queries_pg13.yml ]]
        then
            cat ${CONFIG_DIR?}/queries_pg13.yml >> /tmp/queries.yml
        else
            echo_err "Custom Query file queries_pg12.yml does not exist (it should).."
        fi
    else
        echo_err "Unknown or unsupported version of PostgreSQL.  Exiting.."
        exit 1
    fi
fi

sed -i "s/#PGBACKREST_INFO_THROTTLE_MINUTES#/${PGBACKREST_INFO_THROTTLE_MINUTES:-10}/g" /tmp/queries.yml

PG_OPTIONS="--extend.query-path=${QUERY_DIR?}/queries.yml  --web.listen-address=:${POSTGRES_EXPORTER_PORT}"

echo_info "Starting postgres-exporter.."
${PG_EXP_HOME?}/postgres_exporter ${PG_OPTIONS?} >>/dev/stdout 2>&1 &
echo $! > $POSTGRES_EXPORTER_PIDFILE

wait
