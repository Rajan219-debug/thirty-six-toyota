# Multi-Outlet (Multi-Tenancy) — Developer Guide

This document explains how the Thirty Six Toyota fork isolates data between
**outlets** (dealership locations: Faridabad, Palwal, …) so one deployment can
serve many outlets without them seeing each other's data.

Read this before touching anything tenancy-related. It is written to be
understood without prior context.

---

## 1. The idea in one paragraph

There is **one app and one Postgres database**. Every row that belongs to an
outlet carries a `tenant_id` column pointing at the `tenants` table. Instead of
adding `WHERE tenant_id = ?` to hundreds of queries (easy to forget → data
leak), isolation is enforced by **PostgreSQL Row-Level Security (RLS)**: the
database itself refuses to return or write rows that don't belong to the current
request's outlet. A forgotten filter in Go code therefore *cannot* leak data.

## 2. Who sees what (the access model)

| Role | Reads | Writes |
|------|-------|--------|
| **Staff** | only their own outlet | only their own outlet |
| **HQ / group manager** (`users.is_hq = true`) | **all** outlets (group dashboards & reports) | pinned to one active outlet |

A user's outlet is `users.tenant_id`; whether they are HQ is `users.is_hq`.
Both are set by an instance admin (see §6).

## 3. How it works (the mechanism)

Three moving parts:

### a) Schema — every scoped table gets a `tenant_id`
On every boot, `db.EnsureTenantColumns()` (in [`pkg/db/db.go`](../pkg/db/db.go))
runs idempotent DDL:
- creates the `tenants` table and seeds a default outlet (id 1),
- adds `tenant_id bigint NOT NULL DEFAULT 1 REFERENCES tenants(id)` to every
  registered table **except** an allow-list of globals (see §5),
- adds `users.is_hq`.

It iterates `db.RegisteredTableNames()`, so **any table added to the app later is
scoped automatically** — you don't have to remember to add tenancy to it.

> Why a startup hook and not a migration? On a *fresh* database, Vikunja's
> migration runner (`xormigrate`) builds the schema from Go structs and marks all
> migrations as already-applied *without running them*. A DDL-only migration would
> therefore be skipped on fresh installs. The boot hook covers fresh installs,
> upgrades, and future tables with one mechanism.

`tenant_id` is deliberately **kept out of the Go model structs**. The ORM never
sees it, so inserts fall through to the column `DEFAULT` and RLS does the
filtering. A nice side effect: a user cannot change their own `tenant_id`/`is_hq`
through the normal user-update endpoints, because those fields aren't in the struct.

### b) Enforcement — RLS policies
When enforcement is **on** (see §4), `db.ArmTenantRLS()` enables `FORCE ROW LEVEL
SECURITY` and installs one policy per scoped table:

```sql
CREATE POLICY tenant_isolation ON <table>
  USING       (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint
               OR current_setting('app.is_hq', true) = 'on')
  WITH CHECK  (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint);
```

- `USING` controls what rows a query can **see** → HQ (`app.is_hq='on'`) sees all;
  staff see only their outlet.
- `WITH CHECK` controls what a row may be **written** as → always pinned to the
  session's `app.current_tenant`, so even HQ writes land in one concrete outlet.
- `NULLIF(...,'')` handles a pooled connection whose transaction-local setting has
  reverted to an empty string — empty → `NULL` → no rows match (fail-safe).

### c) Binding — telling the DB which outlet this request is for
`app.current_tenant` / `app.is_hq` are Postgres session settings ("GUCs"). They
are set **per request transaction** with `set_config(..., true)` (transaction-
local, so nothing leaks across pooled connections).

`db.BindTenantForUser(s, userID)` reads the user's outlet + HQ flag from the
RLS-exempt `users` table and sets the two GUCs. It is called through
`db.NewBoundSession(userID)`, which is used at the request seams:

- **Generic CRUD** — `pkg/web/handler/core.go` binds every standard
  create/read/update/delete (projects, tasks, labels, teams, …) in one place.
- **Custom Toyota handlers** — the v2 dashboard, reports, accountability,
  documents and time-tracking handlers call `db.NewBoundSession(a.GetID())`.

Request flow, end to end:

```
HTTP request → auth resolves the user → handler opens db.NewBoundSession(userID)
  → set_config('app.current_tenant', <user's outlet>)   (+ app.is_hq if HQ)
  → runs its queries → RLS filters every row to that outlet → commit
```

## 4. Turning enforcement on/off

Enforcement is gated by an environment variable and is **OFF by default**:

```
VIKUNJA_MULTITENANCY_ENFORCE=1   # arm RLS on boot
```

`db.MultiTenancyEnforced()` reads it. With it **off**, all the tenancy plumbing
is inert: columns exist, sessions bind, but no RLS policy is armed, so the app
behaves exactly like a normal single-outlet app. This lets the binding layer roll
out safely before isolation is switched on.

> **Postgres only.** RLS is a Postgres feature; the tenancy model requires it.
> The hooks no-op on other databases.

## 5. Table classification

**Scoped** (get `tenant_id` + RLS): projects, tasks, teams, labels, buckets,
time_entries, files, notifications, folders, documents, conversations, notes, and
~25 more — everything the app registers, minus the exceptions below.

**Global — no `tenant_id`** (`tenantGlobalTables` in `db.go`): `tenants`,
`migration`, `unsplash_photos`, `license_status`, plus the ephemeral auth tables
`sessions`, `oauth_codes`, `totp`, `user_tokens`.

**Has `tenant_id` but RLS-exempt** (`rlsExemptTables` in `db.go`): `users`,
`api_tokens`, `link_shares`. These are read during authentication — by
email/token/hash — *before* a tenant is known, so a policy would filter auth
itself. Because `users` is RLS-exempt, the global user-directory search is scoped
in application code instead (`models.FilterUsersByOutlet`).

## 6. Operating it (admin)

Instance admins manage outlets from **Admin → Outlets** in the UI, or via the API
(all under `/api/v2/admin`, instance-admin only — see
[`pkg/routes/api/v2/admin_tenants.go`](../pkg/routes/api/v2/admin_tenants.go)):

| Action | Endpoint |
|--------|----------|
| List outlets | `GET /api/v2/admin/tenants` |
| Create outlet | `POST /api/v2/admin/tenants` `{ "name": "Palwal", "slug": "palwal" }` |
| Assign a user to an outlet / set HQ | `PATCH /api/v2/admin/users/{id}/outlet` `{ "tenant_id": 2, "is_hq": false }` |

New users default to outlet 1 until an admin reassigns them.

## 7. Deployment (CRITICAL)

RLS is **silently bypassed by superusers and any `BYPASSRLS` role**. The stock
Postgres image's `POSTGRES_USER` is a superuser, so the app must **not** connect
as it — otherwise every outlet would see everything.

The `docker-compose.yml` in this repo handles it:
- `docker/postgres-init/10-vikunja-app-role.sh` creates a plain, non-superuser
  `vikunja_app` role with `CREATE ON SCHEMA public` (so it can build and **own**
  the tables — required for the DDL hook and for `FORCE RLS` to apply to it).
- The app container connects as `vikunja_app`.

As a safety net, `ArmTenantRLS()` **refuses to start** (fatal) if enforcement is
on but the connected role can bypass RLS — you can't accidentally run "isolated"
while isolating nothing.

## 8. How to extend

- **New table?** Just register it as usual (`GetTables()`). The boot hook scopes
  it automatically. If it must *not* be scoped, add its name to
  `tenantGlobalTables` (no `tenant_id`) or `rlsExemptTables` (column but no
  policy) in `db.go`.
- **New request handler that opens its own session?** Use
  `db.NewBoundSession(a.GetID())` instead of `db.NewSession()` so it's tenant-
  scoped. If you forget, under enforcement it returns **empty** (fail-safe), never
  another outlet's data — so a missed binding shows up as "my data disappeared,"
  not a leak.
- **Background job / cron that spans outlets?** Use `db.NewSystemSession()`
  (reads across all outlets). To *write* per outlet, `set_config('app.current_
  tenant', <target outlet>, true)` before the insert, or `WITH CHECK` rejects it.

## 9. Known limitations / remaining work

Enforcement is off by default precisely because these paths are not yet bound.
**Do not enable enforcement in production until they are done:**

- **Signup** creates a default project/bucket with no bound tenant → fails under
  enforcement. Bind it to the new user's outlet.
- **Crons** (reminders, overdue notifications, token/session cleanup, exports)
  write per-user; they need `NewSystemSession` + per-item `app.current_tenant`.
- **Long-tail handlers** (user settings, webhooks, avatars, backgrounds, CalDAV,
  feeds) still open unbound sessions → return empty under enforcement.
- **Link shares** don't resolve a tenant yet → shared boards return empty under
  enforcement.
- **Backups**: the `tenants` table isn't in the dump set yet, so a cross-machine
  dump/restore loses custom outlets.
- **Audit**: changing a user's outlet/HQ isn't audit-logged (the `is_admin`
  toggle is).

## 10. Verifying

Run against Postgres (RLS needs it). With enforcement on and a non-superuser role:

- **DB level:** `SET app.current_tenant='1'` then `SELECT * FROM tasks` returns
  only outlet-1 rows; `'2'` only outlet-2; `SET app.is_hq='on'` returns all;
  nothing set returns zero.
- **App level:** a staff user in outlet A only ever sees outlet-A data through the
  API; an HQ user sees all; the global user search hides other outlets' users for
  staff but not for HQ.

## Code map

| Concern | File |
|---------|------|
| Schema hook, binding, RLS engine, fail-closed guard | `pkg/db/db.go` (`EnsureTenantColumns`, `BindTenantForUser`, `NewBoundSession`, `NewSystemSession`, `ArmTenantRLS`, `MultiTenancyEnforced`) |
| Wiring hooks into boot | `pkg/initialize/init.go` |
| Generic CRUD binding seam | `pkg/web/handler/core.go` |
| Outlet model + user assignment + user-directory scoping | `pkg/models/tenant.go` |
| Outlet admin API | `pkg/routes/api/v2/admin_tenants.go` |
| Bound Toyota feature handlers | `pkg/routes/api/v2/{dashboard,reports,task_accountability,documents,time_entries}.go` |
| Global user-search scoping | `pkg/routes/api/v2/user_search.go` |
| Outlet admin UI | `frontend/src/views/admin/OutletsView.vue`, `frontend/src/services/admin/tenantService.ts` |
| Deploy role | `docker-compose.yml`, `docker/postgres-init/10-vikunja-app-role.sh` |
