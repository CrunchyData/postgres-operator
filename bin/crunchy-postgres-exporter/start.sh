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

source /opt/cpm/bin/common_lib.sh
enable_debugging

export PG_EXP_HOME=$(find /opt/cpm/bin/ -type d -name 'postgres_exporter*')
export PG_DIR=$(find /usr/ -type d -name 'pgsql-*')
POSTGRES_EXPORTER_PIDFILE=/tmp/postgres_exporter.pid
CONFIG_DIR='/opt/cpm/conf'
QUERIES=(
    queries_backrest
    queries_global
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
set_default_pg_exporter_env

if [[ ! ${#default_exporter_env_vars[@]} -eq 0 ]]
then
    echo_info "Defaults have been set for the following exporter env vars:"
    echo_info "[${default_exporter_env_vars[*]}]"
fi

# Check that postgres is accepting connections.
echo_info "Waiting for PostgreSQL to be ready.."
while true; do
    ${PG_DIR?}/bin/pg_isready -q -h "${EXPORTER_PG_HOST}" -p "${EXPORTER_PG_PORT}"
    if [ $? -eq 0 ]; then
        break
    fi
    sleep 2
done

echo_info "Checking if "${EXPORTER_PG_USER}" is is created.."
while true; do
    PGPASSWORD="${EXPORTER_PG_PASSWORD}" ${PG_DIR?}/bin/psql -q -h "${EXPORTER_PG_HOST}" -p "${EXPORTER_PG_PORT}" -U "${EXPORTER_PG_USER}" -c "SELECT 1;" "${EXPORTER_PG_DATABASE}"
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
            echo_err "Query file ${query?}.yml does not exist (it should).."
            exit 1
        fi
    done

    VERSION=$(PGPASSWORD="${EXPORTER_PG_PASSWORD}" ${PG_DIR?}/bin/psql -h "${EXPORTER_PG_HOST}" -p "${EXPORTER_PG_PORT}" -U "${EXPORTER_PG_USER}" -qtAX -c "SELECT current_setting('server_version_num')" "${EXPORTER_PG_DATABASE}")
    if (( ${VERSION?} >= 90600 )) && (( ${VERSION?} < 100000 ))
    then
        if [[ -f ${CONFIG_DIR?}/pg96/queries_general.yml ]]
        then
            cat ${CONFIG_DIR?}/pg96/queries_general.yml >> /tmp/queries.yml
        else
            echo_err "Query file queries_general.yml does not exist (it should).."
        fi
    elif (( ${VERSION?} >= 100000 )) && (( ${VERSION?} < 110000 ))
    then
        if [[ -f ${CONFIG_DIR?}/pg10/queries_general.yml ]]
        then
            cat ${CONFIG_DIR?}/pg10/queries_general.yml >> /tmp/queries.yml
        else
            echo_err "Query file queries_general.yml does not exist (it should).."
        fi
        if [[ -f ${CONFIG_DIR?}/pg10/queries_pg_stat_statements.yml ]]
        then
          cat ${CONFIG_DIR?}/pg10/queries_pg_stat_statements.yml >> /tmp/queries.yml
        else
          echo_warn "Query file queries_pg_stat_statements.yml not loaded."
        fi
    elif (( ${VERSION?} >= 110000 )) && (( ${VERSION?} < 120000 ))
    then
        if [[ -f ${CONFIG_DIR?}/pg11/queries_general.yml ]]
        then
            cat ${CONFIG_DIR?}/pg11/queries_general.yml >> /tmp/queries.yml
        else
            echo_err "Query file queries_general.yml does not exist (it should).."
        fi
        if [[ -f ${CONFIG_DIR?}/pg11/queries_pg_stat_statements.yml ]]
        then
          cat ${CONFIG_DIR?}/pg11/queries_pg_stat_statements.yml >> /tmp/queries.yml
        else
          echo_warn "Query file queries_pg_stat_statements.yml not loaded."
        fi
    elif (( ${VERSION?} >= 120000 )) && (( ${VERSION?} < 130000 ))
    then
        if [[ -f ${CONFIG_DIR?}/pg12/queries_general.yml ]]
        then
            cat ${CONFIG_DIR?}/pg12/queries_general.yml >> /tmp/queries.yml
        else
            echo_err "Query file queries_general.yml does not exist (it should).."
        fi
        if [[ -f ${CONFIG_DIR?}/pg12/queries_pg_stat_statements.yml ]]
        then
          cat ${CONFIG_DIR?}/pg12/queries_pg_stat_statements.yml >> /tmp/queries.yml
        else
          echo_warn "Query file queries_pg_stat_statements.yml not loaded."
        fi
        # queries_pg_stat_statements_reset is only available in PG12+. This may
        # need to be updated based on a new path
        if [[ -f ${CONFIG_DIR?}/pg12/queries_pg_stat_statements_reset_info.yml ]];
        then
          cat ${CONFIG_DIR?}/pg12/queries_pg_stat_statements_reset_info.yml >> /tmp/queries.yml
        else
          echo_warn "Query file queries_pg_stat_statements_reset_info.yml not loaded."
        fi
    elif (( ${VERSION?} >= 130000 ))
    then
        if [[ -f ${CONFIG_DIR?}/pg13/queries_general.yml ]]
        then
            cat ${CONFIG_DIR?}/pg13/queries_general.yml >> /tmp/queries.yml
        else
            echo_err "Query file queries_general.yml does not exist (it should).."
        fi
        if [[ -f ${CONFIG_DIR?}/pg13/queries_pg_stat_statements.yml ]]
        then
          cat ${CONFIG_DIR?}/pg13/queries_pg_stat_statements.yml >> /tmp/queries.yml
        else
          echo_warn "Query file queries_pg_stat_statements.yml not loaded."
        fi
        # queries_pg_stat_statements_reset is only available in PG12+. This may
        # need to be updated based on a new path
        if [[ -f ${CONFIG_DIR?}/pg13/queries_pg_stat_statements_reset_info.yml ]];
        then
          cat ${CONFIG_DIR?}/pg13/queries_pg_stat_statements_reset_info.yml >> /tmp/queries.yml
        else
          echo_warn "Query file queries_pg_stat_statements_reset_info.yml not loaded."
        fi
    else
        echo_err "Unknown or unsupported version of PostgreSQL.  Exiting.."
        exit 1
    fi
fi

sed -i \
  -e "s/#PGBACKREST_INFO_THROTTLE_MINUTES#/${PGBACKREST_INFO_THROTTLE_MINUTES:-10}/g" \
  -e "s/#PG_STAT_STATEMENTS_LIMIT#/${PG_STAT_STATEMENTS_LIMIT:-20}/g" \
  -e "s/#PG_STAT_STATEMENTS_THROTTLE_MINUTES#/${PG_STAT_STATEMENTS_THROTTLE_MINUTES:--1}/g" \
  /tmp/queries.yml

PG_OPTIONS="--extend.query-path=${QUERY_DIR?}/queries.yml  --web.listen-address=:${POSTGRES_EXPORTER_PORT}"

echo_info "Starting postgres-exporter.."
DATA_SOURCE_URI="${EXPORTER_PG_HOST}:${EXPORTER_PG_PORT}/${EXPORTER_PG_DATABASE}?${EXPORTER_PG_PARAMS}" DATA_SOURCE_USER="${EXPORTER_PG_USER}" DATA_SOURCE_PASS="${EXPORTER_PG_PASSWORD}" ${PG_EXP_HOME?}/postgres_exporter ${PG_OPTIONS?} >>/dev/stdout 2>&1 &
echo $! > $POSTGRES_EXPORTER_PIDFILE

wait
