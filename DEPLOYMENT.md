# First-time production deployment

This is a one-time setup. After this is done once, every future update is a
single command from your own machine: `./deploy.sh user@vps-host`.

If you're handing this to a contractor: this whole document is the brief.
Nothing else needed except SSH access to the server they set up.

## What you need before starting

- **A VPS** — any provider works (DigitalOcean, Hetzner, Linode, AWS
  Lightsail...). Cheapest tier that has 2GB+ RAM is fine. Ubuntu 22.04 or
  24.04 recommended.
- **A domain** (or subdomain, e.g. `tasks.yourdealership.com`) you can point
  DNS at. Needs an A record → the VPS's IP address, and ports 80 + 443 open.

## 1. Install Docker on the VPS

SSH into the fresh server, then:

```bash
curl -fsSL https://get.docker.com | sh
```

That's the whole install — Docker's official script handles everything.

## 2. Clone the repo

```bash
mkdir -p /opt/vikunja && cd /opt/vikunja
git clone https://github.com/Rajan219-debug/thirty-six-toyota.git .
```

(Needs read access to the repo — add the deploying machine's SSH key as a
deploy key on GitHub, or clone over HTTPS with a personal access token, if
the repo isn't public.)

## 3. Create the real `.env`

```bash
cp .env.production.example .env
nano .env   # or vim, whatever's installed
```

Fill in every blank value. Concretely:

- `VIKUNJA_JWT_SECRET` — run `openssl rand -hex 32` on the server and paste
  the output in. **Never reuse a value shown anywhere else** (chat, docs,
  examples) — generate it fresh, on this machine, right now.
- `VIKUNJA_DB_PASSWORD` and `VIKUNJA_APP_DB_PASSWORD` — same idea, two
  *different* random values. `openssl rand -hex 24` works fine for both.
- `VIKUNJA_DOMAIN` / `VIKUNJA_PUBLIC_URL` — your real domain from the
  prerequisites above.
- Leave `VIKUNJA_MULTITENANCY_ENFORCE=0` and `VIKUNJA_MAILER_ENABLED=false`
  for now — see "After it's running" below.

Every value's own comment in that file explains what it's for and why it
matters — read them, they're not filler.

## 4. Start it

```bash
docker compose up -d --build
```

First run takes a few minutes (building the image). Caddy will automatically
request a TLS certificate from Let's Encrypt the moment it can reach your
domain on port 80 — no manual cert setup.

Check it's up:

```bash
docker compose ps        # all three services should show "healthy" / "running"
curl -I https://your-domain-here
```

## 5. Register the first account

Visit `https://your-domain-here` in a browser and register. **The first
account created becomes the instance admin** — do this yourself, don't let
someone else create the first account.

At this point the app is live and usable, single-outlet, no email
notifications yet. That's a legitimate stopping point — the two items below
are real but not blocking.

## After it's running (do these once things are confirmed working)

**Multi-outlet isolation** (if you have more than one outlet/dealership
location): re-run the two-tenant isolation test against this real database,
confirm the app is connecting as the non-superuser `vikunja_app` role (it
is, by default, from this compose file — just confirm nothing was changed),
then set `VIKUNJA_MULTITENANCY_ENFORCE=1` in `.env` and
`docker compose up -d` again to restart with it on. Full detail and the
exact risk if this is skipped or rushed:
`plans/thirtysixtoyota-multitenancy.md`.

**Email notifications** (task-assignment / overdue alerts — a core part of
what this app is for): pick a transactional mail provider, set the
`VIKUNJA_MAILER_*` values in `.env` (the file has copy-paste examples for
SendGrid/Postmark/SES/Mailgun), `VIKUNJA_MAILER_ENABLED=true`, restart, and
verify by assigning yourself a task and confirming the email arrives.

**Backups**: `db-backup-prod.sh` exists for this — see its own header
comment for the one line to add to cron.

## Every update after today

From your own machine, not the server:

```bash
./deploy.sh user@your-vps-host
```

That pulls the latest commit and rebuilds/restarts on the VPS. One command,
no SSH session needed.
