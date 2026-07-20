# Thirty Six Toyota — Task Accountability & Command Center

[![License: AGPL-3.0-or-later](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](LICENSE)

A dealership task-accountability web application for internal operations:
assigning work to service/sales/parts staff, tracking due dates, and holding
assignees accountable for on-time completion. It's a functional fork of
[Vikunja](https://vikunja.io) (an open-source, Go + Vue task-management
platform), reskinned to the Toyota brand and extended with dealership-specific
accountability logic — task status lifecycle, manager sign-off on closing
overdue work, timesheets/reporting, and Postgres-RLS-backed multi-outlet
isolation — that stock Vikunja doesn't have. See
[`plans/thirtysixtoyota-synopsis.md`](plans/thirtysixtoyota-synopsis.md) for
the full rationale, and the other files under [`plans/`](plans/) for the
design of each major feature.

## Security Reports

Send security concerns directly to whoever administers this deployment —
this is an internal tool with no public bug-bounty program.

## Docs

* [`AGENTS.md`](AGENTS.md) / [`CLAUDE.md`](CLAUDE.md) — repo conventions and
  AI-assistant guidance
* [`CONTRIBUTING.md`](CONTRIBUTING.md) — contribution guidelines
* [`DEPLOYMENT.md`](DEPLOYMENT.md) — first-time production deployment, start to finish
* [`docker-compose.yml`](docker-compose.yml) + [`.env.production.example`](.env.production.example) — production stack reference
* [`docker/thirtysixtoyota/README.md`](docker/thirtysixtoyota/README.md) — local dev/demo stack
* [`dev-run.sh`](dev-run.sh) — local dev server against a real Postgres DB

Upstream Vikunja's own docs (build-from-source, magefile targets, testing
conventions) at [vikunja.io/docs](https://vikunja.io/docs/) still apply where
this fork hasn't changed that part of the stack.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## License

Most of this repository is licensed under [AGPL‑3.0‑or‑later](LICENSE).
The contents of [`desktop/`](desktop/) are licensed under
[GPL‑3.0‑or‑later](desktop/LICENSE).

### A note on this fork

This is a modified fork (see the commit history for the Thirty Six Toyota
changes on top of upstream Vikunja). AGPL's network-use clause means anyone
interacting with a running instance over a network — e.g. dealership staff
using the deployed app — has a right to that instance's corresponding
source, including these modifications. For a private, single-dealership
deployment this is low-stakes, but keep this repository (or a copy of it)
reachable to anyone who uses the app, rather than leaving it as an
unshared local checkout, and revisit this if the fork is ever reused across
multiple dealerships as a product.

### Unsplash Images

Background images from Unsplash are distributed under the [Unsplash License](https://unsplash.com/license). The license requires giving credit to the photographer and Unsplash. See [Unsplash’s terms](https://unsplash.com/terms) for more information.
