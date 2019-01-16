#!/bin/bash

set -e -u

export PGPASSFILE=/tmp/pgpass

cat >> "${PGPASSFILE?}" <<-EOF
${PG_HOST?}:${PG_PORT?}:${PG_DATABASE?}:${PG_USER?}:${PG_PASSWORD?}
EOF
chmod 0600 ${PGPASSFILE?}

for sql in /pgconf/*.sql
do
    psql -d ${PG_DATABASE?} -U ${PG_USER?} \
         -p ${PG_PORT?} -h ${PG_HOST?} \
         -f ${sql?}
done

exit 0
