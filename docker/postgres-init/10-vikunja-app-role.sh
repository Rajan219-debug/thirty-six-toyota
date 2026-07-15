#!/bin/bash
# Creates the non-superuser role the Vikunja app connects as.
#
# Row-Level Security (the multi-outlet isolation) is SILENTLY BYPASSED by
# superusers and BYPASSRLS roles. The stock POSTGRES_USER is a superuser, so the
# app must NOT connect as it. This role is a plain LOGIN role: it creates and
# owns the tables on first boot (so the schema-DDL hook works and FORCE RLS
# applies to it), but it cannot bypass RLS.
#
# Runs once, on first database initialisation, as the superuser.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE ROLE vikunja_app LOGIN PASSWORD '${VIKUNJA_APP_DB_PASSWORD:-vikunja_app}';
    GRANT CONNECT ON DATABASE ${POSTGRES_DB} TO vikunja_app;
    -- CREATE lets it build the schema on first boot (PG15+ no longer grants this
    -- to non-owners by default); USAGE lets it use the schema thereafter.
    GRANT USAGE, CREATE ON SCHEMA public TO vikunja_app;
EOSQL
