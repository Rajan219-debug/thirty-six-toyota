# Thirty Six Toyota — Docker stack

A self-contained, isolated Docker stack that runs **this fork** of Vikunja (the
custom task-accountability + timesheets/reporting build), not the upstream
`vikunja/vikunja` image.

## Run

```sh
cd docker/thirtysixtoyota
docker compose up -d --build      # first run builds the fork image (frontend + API)
```

Then open <http://localhost:3456>. The **first user you register becomes the
instance admin**.

Stop / start / logs:

```sh
docker compose stop
docker compose start
docker compose logs -f vikunja
```

Rebuild after code changes:

```sh
docker compose up -d --build
```

## What's isolated

Compose project name `thirtysixtoyota`, so nothing merges with other stacks:

- **Volumes** `thirtysixtoyota_db` (Postgres data) and `thirtysixtoyota_files`
  (attachments) — namespaced, survive restarts.
- **Network** `thirtysixtoyota-net`.
- **Containers** `t36-vikunja` and `t36-vikunja-db`.
- Postgres is **internal-only** (no host port published), so it never clashes
  with a local Postgres or the `vikunja-verify-pg` test container.

Only port **3456** (the web app) is published to the host.

## Before exposing it to anyone

- Change `VIKUNJA_SERVICE_JWTSECRET` in `docker-compose.yml`.
- Change the Postgres password (`POSTGRES_PASSWORD` **and** the matching
  `VIKUNJA_DATABASE_PASSWORD`).

## Wipe and start fresh

```sh
docker compose down -v            # -v also deletes the db + files volumes
```
