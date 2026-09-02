#!/bin/sh
set -e

BACKUP_DIR="${BACKUP_DIR:-/backups}"
RETAIN_COUNT="${RETAIN_COUNT:-7}"
PGHOST="${PGHOST:-postgres}"
PGUSER="${PGUSER:-vpn}"
PGDATABASE="${PGDATABASE:-vpn}"

mkdir -p "$BACKUP_DIR"

run_backup() {
  TIMESTAMP=$(date +%Y%m%d_%H%M%S)
  FILE="$BACKUP_DIR/vpn_${TIMESTAMP}.sql.gz"

  echo "[$(date -Iseconds)] Starting backup: $FILE"
  pg_dump -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" --clean --if-exists | gzip > "$FILE"

  # Rotate: keep only the newest $RETAIN_COUNT backups.
  ls -1t "$BACKUP_DIR"/vpn_*.sql.gz | tail -n +$((RETAIN_COUNT + 1)) | while read -r old; do
    echo "[$(date -Iseconds)] Removing old backup: $old"
    rm -f "$old"
  done

  echo "[$(date -Iseconds)] Backup done: $FILE"
}

if [ "${1:-}" = "backup" ]; then
  run_backup
  exit 0
fi

# Default container mode: run once, then schedule via cron.
SCHEDULE="${SCHEDULE:-0 3 * * *}"
run_backup

echo "$SCHEDULE /backup/run.sh backup" | crontab -
exec crond -f
