#!/usr/bin/env bash
# deploy.sh — redeploy the production stack on the VPS.
# Run from your local machine (or a CI job) once the VPS is set up (see
# .env.production.example + docker-compose.yml + Caddyfile):
#
#   ./deploy.sh user@vps-host [/opt/vikunja]
#
# Pulls the latest commit on the remote and rebuilds/restarts the stack.
# Assumes the repo is already cloned on the VPS with a real .env in place —
# this script does not create either.
set -euo pipefail

HOST="${1:?Usage: ./deploy.sh user@host [remote-path]}"
REMOTE_PATH="${2:-/opt/vikunja}"

ssh "$HOST" "cd '$REMOTE_PATH' && git pull && docker compose up -d --build"
