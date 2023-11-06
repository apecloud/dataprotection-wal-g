# WAL-G for FoundationDB

**Work in progress**

You can use wal-g as a tool for encrypting, compressing FoundationDB backups and push/fetch them to/from storage.

Usage
-----

### ``backup-fetch``

Command for sending backup from storage to stream in order to restore it in the database.

```bash
wal-g backup-fetch example_backup
```

Variable _WALG_STREAM_RESTORE_COMMAND_ is required for use backup-fetch
(eg. ```TMP_DIR=$(mktemp -d) && chmod 777 $TMP_DIR && tar -xf - -C $TMP_DIR && BACKUP_DIR=$(find $TMP_DIR -mindepth 1 -print -quit) && fdbrestore start -r file://$BACKUP_DIR -w --dest_cluster_file "/etc/foundationdb/fdb.cluster"  1>&2```)

WAL-G can also fetch the latest backup using:

```bash
wal-g backup-fetch LATEST
```

### ``backup-push``

Command for compressing, encrypting and sending backup from stream to storage.

```bash
wal-g backup-push
```

Variable _WALG_STREAM_CREATE_COMMAND_ is required for use backup-push 
(eg. ```TMP_DIR=$(mktemp -d) && chmod 777 $TMP_DIR && fdbbackup start -d file://$TMP_DIR -w 1>&2 && tar -c -C $TMP_DIR .```)


