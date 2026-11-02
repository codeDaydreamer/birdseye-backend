#!/usr/bin/env bash
# Nightly database backups for the Birdseye stack (run by birdseye-backup.timer on the VPS).
#
# Dumps the Birdseye MySQL database and the Dynapay Postgres database from their containers,
# compresses them, optionally encrypts them with age, keeps BACKUP_KEEP_DAYS of local copies and
# optionally copies them off the server with rclone. Credentials are read inside each container
# from its own environment, so none are needed here.
#
# Settings (environment, or /etc/birdseye-backup.env):
#   BACKUP_DIR            where dumps go                 (default /var/backups/birdseye)
#   BACKUP_KEEP_DAYS      local copies to keep            (default 14)
#   MYSQL_CONTAINER       Birdseye MySQL container        (default birdseye-mysql)
#   POSTGRES_CONTAINER    Dynapay Postgres container      (default dynapay-postgres)
#   BACKUP_AGE_RECIPIENT  age public key; set it to encrypt dumps (recommended for off-site copies)
#   BACKUP_RCLONE_REMOTE  e.g. "b2:birdseye-backups"; set it to copy dumps off the server
#
# Dynapay's M-Pesa credentials are encrypted with DYNAPAY_ENCRYPTION_KEY. A Postgres dump is
# useless without that key, so keep a copy of it in a password manager; it is NOT in these dumps.
#
# Restore:
#   gunzip -c birdseye-<stamp>.sql.gz | docker exec -i birdseye-mysql sh -c 'exec mysql -uroot -p"$MYSQL_ROOT_PASSWORD"'
#   gunzip -c dynapay-<stamp>.sql.gz  | docker exec -i dynapay-postgres sh -c 'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
#   (for .age files: age -d -i key.txt file.sql.gz.age | gunzip -c | ...)
set -euo pipefail

[[ -f /etc/birdseye-backup.env ]] && source /etc/birdseye-backup.env

BACKUP_DIR=${BACKUP_DIR:-/var/backups/birdseye}
BACKUP_KEEP_DAYS=${BACKUP_KEEP_DAYS:-14}
MYSQL_CONTAINER=${MYSQL_CONTAINER:-birdseye-mysql}
POSTGRES_CONTAINER=${POSTGRES_CONTAINER:-dynapay-postgres}
stamp=$(date -u +%Y%m%dT%H%M%SZ)

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
umask 077

log() { echo "$(date -u +%FT%TZ) $*"; }

# Run a dump command in a container into a compressed file. The file only gets its final name
# once the dump has fully succeeded, so a failed run never leaves a truncated file that looks
# like a good backup.
dump() {
  local container=$1 name=$2 cmd=$3 tmp="$BACKUP_DIR/.$2.partial"
  log "dumping $name from $container"
  if ! docker exec "$container" sh -c "$cmd" | gzip -9 > "$tmp"; then
    rm -f "$tmp"
    log "FAILED: $name"
    return 1
  fi
  if [[ -n "${BACKUP_AGE_RECIPIENT:-}" ]]; then
    age -r "$BACKUP_AGE_RECIPIENT" -o "$tmp.age" "$tmp"
    rm -f "$tmp"
    mv "$tmp.age" "$BACKUP_DIR/$name.age"
  else
    mv "$tmp" "$BACKUP_DIR/$name"
  fi
}

failed=0
dump "$MYSQL_CONTAINER" "birdseye-$stamp.sql.gz" \
  'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysqldump -uroot --single-transaction --routines --triggers --databases "$MYSQL_DATABASE"' \
  || failed=1
dump "$POSTGRES_CONTAINER" "dynapay-$stamp.sql.gz" \
  'exec pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists' \
  || failed=1

if [[ -n "${BACKUP_RCLONE_REMOTE:-}" ]]; then
  log "copying to $BACKUP_RCLONE_REMOTE"
  rclone copy "$BACKUP_DIR" "$BACKUP_RCLONE_REMOTE" --include "*-$stamp.sql.gz*"
fi

find "$BACKUP_DIR" -maxdepth 1 -type f -name '*.sql.gz*' -mtime "+$BACKUP_KEEP_DAYS" -delete
log "done: $(ls -1 "$BACKUP_DIR" | grep -c "$stamp") files for $stamp"
# A non-zero exit marks the systemd unit failed, so a broken backup shows up in `systemctl --failed`
exit "$failed"
