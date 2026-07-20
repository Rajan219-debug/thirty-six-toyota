#!/usr/bin/env bash
# db-backup-prod.sh — dump the production Postgres DB and ship it off-box.
# A lost VPS should never mean a lost database, so this always tries to
# upload the dump; it only stays local-only if you haven't configured a
# remote yet (and says so loudly).
#
# Run on the VPS, from the repo directory, once the production stack is up
# (`docker compose up -d`). Wire it into cron for a daily backup:
#
#   crontab -e
#   0 3 * * * cd /opt/vikunja && ./db-backup-prod.sh >> /var/log/vikunja-backup.log 2>&1
#
# Configure exactly ONE of these in .env (next to docker-compose.yml):
#   BACKUP_REMOTE       an rclone remote:path, e.g. b2:mydealer-backups/vikunja
#                        (rclone config sets up the remote itself; any
#                        S3-compatible provider works — Backblaze B2, Hetzner
#                        Object Storage, AWS S3, etc.)
#   BACKUP_S3_BUCKET    an s3://bucket/prefix, uploaded via `aws s3 cp`
#                        instead (needs the aws CLI + AWS_* credentials env)
#
# RESTORE DRILL — do this once before go-live, and periodically after:
#   1. gunzip -c backups/<file>.sql.gz > /tmp/restore.sql
#   2. Spin up a scratch Postgres (or a fresh vikunja_app-owned DB) and:
#        docker compose exec -T db psql -U vikunja -d postgres -c \
#          "CREATE DATABASE restore_test OWNER vikunja_app;"
#        docker compose exec -T db psql -U vikunja -d restore_test < /tmp/restore.sql
#   3. Confirm real data comes back (row counts, spot-check a task), then
#        docker compose exec -T db psql -U vikunja -d postgres -c \
#          "DROP DATABASE restore_test;"
#   A backup nobody has restored from is a hope, not a backup.
set -euo pipefail

cd "$(dirname "$0")"
[ -f .env ] && set -a && source .env && set +a

DB="${VIKUNJA_DB_NAME:-vikunja}"
KEEP_LOCAL=7
DIR="./backups"
mkdir -p "$DIR"

STAMP="$(date +%Y-%m-%d_%H%M%S)"
OUT="$DIR/${DB}_${STAMP}.sql.gz"

# --clean --if-exists makes the dump self-restoring (drops old objects first).
docker compose exec -T db pg_dump -U vikunja --clean --if-exists "$DB" | gzip > "$OUT"

if [ ! -s "$OUT" ]; then
	echo "ERROR: backup is empty — dump failed, removing." >&2
	rm -f "$OUT"
	exit 1
fi

echo "OK  local backup: $OUT ($(wc -c < "$OUT") bytes)"

if [ -n "${BACKUP_REMOTE:-}" ]; then
	rclone copy "$OUT" "$BACKUP_REMOTE"
	echo "OK  uploaded via rclone to $BACKUP_REMOTE"
elif [ -n "${BACKUP_S3_BUCKET:-}" ]; then
	aws s3 cp "$OUT" "$BACKUP_S3_BUCKET/$(basename "$OUT")"
	echo "OK  uploaded via aws s3 to $BACKUP_S3_BUCKET"
else
	echo "WARNING: no BACKUP_REMOTE or BACKUP_S3_BUCKET set in .env — this backup stayed local-only, which defeats the point of an off-box backup. Set one before going live." >&2
fi

# Local retention: keep only the newest few — the off-box copy is the real
# history, this is just enough to restore quickly without a download.
ls -1t "$DIR"/"${DB}"_*.sql.gz 2>/dev/null | tail -n +$((KEEP_LOCAL + 1)) | xargs -r rm -f
