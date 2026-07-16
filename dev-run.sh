#!/usr/bin/env bash
# dev-run.sh — one-command local dev loop for the Thirty Six Toyota app.
#
# Rebuilds the frontend + Go binary (which embeds the frontend) and (re)starts
# the server against the local `vikunja-verify-pg` Postgres container, bringing
# up Docker + Postgres first if they're down. Local Windows / Git-Bash helper.
#
#   ./dev-run.sh            full rebuild (frontend + backend) + restart
#   ./dev-run.sh --backend  backend-only change: skip the slow frontend build
#   ./dev-run.sh --restart  no build, just (re)start the current binary
#   ./dev-run.sh --mail     also start MailHog + turn email ON (view at :8025)
#                           (combine, e.g. `./dev-run.sh --restart --mail`)
#
# Env overrides: PORT (default 3466), DB (default vikunja_mt_verify).
set -uo pipefail

REPO="$(cd "$(dirname "$0")" && pwd)"
BIN="$REPO/vikunja-mt-verify.exe"
PORT="${PORT:-3466}"
DB="${DB:-vikunja_mt_verify}"
PGC="vikunja-verify-pg"
DOCKER_DESKTOP="/c/Program Files/Docker/Docker/Docker Desktop.exe"

MODE=""
WANT_MAIL=0
for arg in "$@"; do
	case "$arg" in
		--mail) WANT_MAIL=1 ;;
		--restart | --backend) MODE="$arg" ;;
		*) echo "unknown flag: $arg" ; exit 1 ;;
	esac
done

log() { echo "[dev-run] $*"; }

# 1. Docker daemon ---------------------------------------------------------
if ! docker info >/dev/null 2>&1; then
	log "Docker down — starting Docker Desktop…"
	"$DOCKER_DESKTOP" >/dev/null 2>&1 &
	for _ in $(seq 1 50); do docker info >/dev/null 2>&1 && break; sleep 3; done
	docker info >/dev/null 2>&1 || { log "ERROR: Docker did not come up"; exit 1; }
fi

# 2. Postgres container ----------------------------------------------------
if ! docker ps --format '{{.Names}}' | grep -q "^${PGC}$"; then
	log "starting ${PGC}…"
	docker start "$PGC" >/dev/null 2>&1 || docker run -d --name "$PGC" \
		-e POSTGRES_USER=vikunja -e POSTGRES_PASSWORD=vikunja -e POSTGRES_DB=vikunja \
		-p 55432:5432 postgres:16 >/dev/null
fi
for _ in $(seq 1 20); do docker exec "$PGC" pg_isready -U vikunja >/dev/null 2>&1 && break; sleep 2; done
log "Postgres ready"

# 2b. MailHog mail-catcher (only with --mail) ------------------------------
if [ "$WANT_MAIL" = "1" ]; then
	if ! docker ps --format '{{.Names}}' | grep -q '^mailhog$'; then
		log "starting MailHog (mail catcher)…"
		docker start mailhog >/dev/null 2>&1 || docker run -d --name mailhog \
			-p 1025:1025 -p 8025:8025 mailhog/mailhog >/dev/null
	fi
	log "MailHog ready — captured emails at http://localhost:8025"
fi

# 3. Build -----------------------------------------------------------------
if [ "$MODE" != "--restart" ]; then
	if [ "$MODE" != "--backend" ]; then
		log "building frontend (pnpm build)…"
		( cd "$REPO/frontend" && pnpm build ) || { log "ERROR: frontend build failed"; exit 1; }
	fi
	log "building binary (embeds frontend)…"
	( cd "$REPO" && CGO_ENABLED=0 go build -o "$BIN" . ) || { log "ERROR: go build failed"; exit 1; }
fi

# 4. (Re)start -------------------------------------------------------------
taskkill //IM vikunja-mt-verify.exe //F >/dev/null 2>&1 || true

export VIKUNJA_DATABASE_TYPE=postgres VIKUNJA_DATABASE_HOST=localhost:55432 \
	VIKUNJA_DATABASE_USER=vikunja VIKUNJA_DATABASE_PASSWORD=vikunja VIKUNJA_DATABASE_DATABASE="$DB" \
	VIKUNJA_SERVICE_ROOTPATH="$REPO" VIKUNJA_SERVICE_PUBLICURL="http://localhost:$PORT/" \
	VIKUNJA_SERVICE_INTERFACE=":$PORT" VIKUNJA_SERVICE_ENABLEREGISTRATION=true

if [ "$WANT_MAIL" = "1" ]; then
	export VIKUNJA_MAILER_ENABLED=1 VIKUNJA_MAILER_HOST=localhost VIKUNJA_MAILER_PORT=1025 \
		VIKUNJA_MAILER_USERNAME= VIKUNJA_MAILER_PASSWORD= VIKUNJA_MAILER_SKIPTLSVERIFY=1 \
		VIKUNJA_MAILER_FORCESSL=false VIKUNJA_MAILER_FROMEMAIL=noreply@thirtysixtoyota.com
	log "email ON → every task assignment mails the assignee (view at :8025)"
fi

log "starting server on :$PORT (db=$DB)…"
cd "$REPO"
nohup "$BIN" web > "$REPO/dev-run.log" 2>&1 &

for _ in $(seq 1 30); do
	[ "$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/v1/info" 2>/dev/null)" = "200" ] && break
	sleep 2
done
CODE="$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/v1/info" 2>/dev/null)"
if [ "$CODE" = "200" ]; then
	log "UP  →  http://localhost:$PORT   (login boss / testpass123; logs: dev-run.log)"
else
	log "server not responding yet (code=$CODE) — tail of dev-run.log:"
	tail -6 "$REPO/dev-run.log" 2>/dev/null
fi
