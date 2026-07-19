#!/usr/bin/env bash
# db-backup.sh — dump the Thirty Six Toyota Postgres database to a timestamped
# .sql file on your Windows disk. Safe to run anytime; keeps the last 14 dumps.
#
#   ./db-backup.sh                 # write a backup now
#   DB=other_db ./db-backup.sh     # back up a different database
#
# Restore a dump with:
#   docker exec -i vikunja-verify-pg psql -U vikunja -d vikunja_mt_verify < backups/<file>.sql
set -euo pipefail

PGC="vikunja-verify-pg"
DB="${DB:-vikunja_mt_verify}"
DIR="$(cd "$(dirname "$0")" && pwd)/backups"
KEEP=14

mkdir -p "$DIR"

if ! docker ps --format '{{.Names}}' | grep -q "^${PGC}$"; then
	echo "ERROR: container '$PGC' is not running — start it with ./dev-run.sh first." >&2
	exit 1
fi

STAMP="$(date +%Y-%m-%d_%H%M%S)"
OUT="$DIR/${DB}_${STAMP}.sql"

# --clean --if-exists makes the dump self-restoring (drops old objects first).
docker exec "$PGC" pg_dump -U vikunja --clean --if-exists "$DB" > "$OUT"

# Guard against a truncated/empty dump.
if [ ! -s "$OUT" ]; then
	echo "ERROR: backup is empty — dump failed, removing." >&2
	rm -f "$OUT"
	exit 1
fi

# Retention: keep only the newest $KEEP dumps.
ls -1t "$DIR"/${DB}_*.sql 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f

echo "OK  backup written: $OUT  ($(wc -c < "$OUT") bytes)"
echo "    restore: docker exec -i $PGC psql -U vikunja -d $DB < \"$OUT\""
