#!/usr/bin/env bash
# cleanup-dev-accounts.sh — find and remove dev/test accounts before go-live.
#
# Run ON THE VPS, from the repo directory, once the production stack is up
# (`docker compose up -d`):
#
#   ./cleanup-dev-accounts.sh              # dry run: lists matching accounts, deletes nothing
#   ./cleanup-dev-accounts.sh --apply      # actually deletes the known dev/test accounts
#
# Deletion goes through `vikunja user delete --now`, not a raw SQL DELETE —
# that's the same cascading-delete path the CLI and account-deletion request
# flow already use (pkg/models/user_delete.go's DeleteUser), so a removed
# account's owned projects/tasks/etc. are cleaned up correctly instead of
# left as orphaned rows or blocked by a foreign key.
#
# Two categories, handled differently:
#   1. Known artifacts (exact allowlist below) — load-test accounts and the
#      demo logins from the mockup build-out. Listed and, with --apply,
#      deleted automatically.
#   2. Anything else with a suspicious-looking username (quotes, angle
#      brackets, SQL/script-injection-style characters — left over from the
#      security sweep's payload-as-username tests) — only ever listed for
#      you to review and delete by hand with `vikunja user delete`. Never
#      auto-deleted: a broad pattern here risks catching a real name.
set -euo pipefail

APPLY=0
if [ "${1:-}" = "--apply" ]; then
	APPLY=1
elif [ -n "${1:-}" ]; then
	echo "Usage: $0 [--apply]" >&2
	exit 1
fi

DB="${DB:-vikunja}"
PSQL="docker compose exec -T db psql -U vikunja -d $DB -tAc"

# Exact matches only, no wildcards on the demo names — a real staff member
# named e.g. "Manager" shouldn't get caught by an overeager LIKE pattern.
KNOWN_PATTERN='^(boss|manager|staff1|staff2)$|^(qa_worker_|mass_w_)'

echo "== Known dev/test accounts (will be deleted with --apply) =="
KNOWN=$($PSQL "SELECT id || E'\t' || username FROM users WHERE username ~ '$KNOWN_PATTERN' ORDER BY id;")
if [ -z "$KNOWN" ]; then
	echo "  (none found)"
else
	echo "$KNOWN" | while IFS=$'\t' read -r id username; do
		echo "  [$id] $username"
	done
fi

echo
echo "== Suspicious-looking accounts (review manually, never auto-deleted) =="
SUSPECT=$($PSQL "SELECT id || E'\t' || username FROM users WHERE username ~ '[<>;''\"=]|--' AND username !~ '$KNOWN_PATTERN' ORDER BY id;")
if [ -z "$SUSPECT" ]; then
	echo "  (none found)"
else
	echo "$SUSPECT" | while IFS=$'\t' read -r id username; do
		echo "  [$id] $username"
	done
	echo "  Remove one with: docker compose exec vikunja /app/vikunja/vikunja user delete <id> --now --confirm"
fi

if [ "$APPLY" != "1" ]; then
	echo
	echo "Dry run only — re-run with --apply to delete the known accounts listed above."
	exit 0
fi

if [ -z "$KNOWN" ]; then
	echo
	echo "Nothing to delete."
	exit 0
fi

echo
echo "About to permanently delete the known accounts listed above and everything they own."
read -r -p "Type YES to continue: " CONFIRM
if [ "$CONFIRM" != "YES" ]; then
	echo "Aborted."
	exit 1
fi

echo "$KNOWN" | while IFS=$'\t' read -r id username; do
	echo "Deleting [$id] $username..."
	docker compose exec -T vikunja /app/vikunja/vikunja user delete "$id" --now --confirm
done

echo "Done."
