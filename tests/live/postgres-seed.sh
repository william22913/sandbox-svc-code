#!/bin/sh
# Applies tests/live/postgres-seed.sql once the service's migrator has created
# the tables, then stays alive so `docker compose up --wait` has a container to
# wait on.
#
# The service applies migrations/main itself at startup, and this container has
# no ordering dependency on the app — so the wait is a poll for the columns the
# fixture writes to, not a sleep. Both columns come from the last migration
# (3_export_columns.sql), which commits after the tables exist: waiting on the
# table alone leaves a window where the seed would fail with "column does not
# exist". Bounded, so a stack whose migration never finishes fails the up phase
# instead of hanging until the runner times out.
set -eu

attempts=0
until [ "$(psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -tAc \
    "SELECT count(*) FILTER (WHERE (table_name, column_name) IN
       (('catalog_items','price'), ('favorite_items','note'))) = 2
     FROM information_schema.columns
     WHERE table_schema = 'public'")" = "t" ]; do
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 120 ]; then
        echo "catalog_items.price / favorite_items.note did not appear after 120s; the app's migration did not run" >&2
        exit 1
    fi
    sleep 1
done

psql -h "$PGHOST" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f /seed/1_live_seed.sql

# Read by the compose healthcheck. Written only after psql exits 0.
touch /seed.done

exec tail -f /dev/null
