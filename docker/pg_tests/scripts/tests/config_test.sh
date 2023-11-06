#!/bin/sh
set -e -x
CONFIG_FILE="/tmp/configs/config_test_config.json"
mkdir /tmp/storage

COMMON_CONFIG="/tmp/configs/common_config.json"
TMP_CONFIG="/tmp/configs/tmp_config.json"
cat ${CONFIG_FILE} > ${TMP_CONFIG}

echo "," >> ${TMP_CONFIG}
cat ${COMMON_CONFIG} >> ${TMP_CONFIG}
/tmp/scripts/wrap_config_file.sh ${TMP_CONFIG}

/usr/lib/postgresql/10/bin/initdb ${PGDATA}

echo "archive_mode = on" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_command = '/usr/bin/timeout 600 /usr/bin/wal-g --config=${TMP_CONFIG} wal-push %p'" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_timeout = 600" >> /var/lib/postgresql/10/main/postgresql.conf

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start

/tmp/scripts/wait_while_pg_not_ready.sh

wal-g --config=${TMP_CONFIG} delete everything FORCE --confirm

pgbench -i -s 4 postgres
pg_dumpall -f /tmp/dump1
pgbench -c 2 -T 100000000 -S &
sleep 1
wal-g --config=${TMP_CONFIG} backup-push ${PGDATA}

pkill -9 postgres

rm -rf "${PGDATA}"

wal-g --config=${TMP_CONFIG} --turbo backup-fetch "${PGDATA}" LATEST

echo "restore_command = 'echo \"WAL file restoration: %f, %p\"&& /usr/bin/wal-g --config=${TMP_CONFIG} wal-fetch \"%f\" \"%p\"'" > "${PGDATA}"/recovery.conf

/usr/lib/postgresql/10/bin/pg_ctl -D "${PGDATA}" -w start

/tmp/scripts/wait_while_pg_not_ready.sh

pg_dumpall -f /tmp/dump2

diff /tmp/dump1 /tmp/dump2

pkill -9 postgres
rm -rf "${PGDATA}"
rm ${TMP_CONFIG}
