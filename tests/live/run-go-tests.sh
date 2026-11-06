#!/bin/sh
# The repo's Go checks, run from a golang image because the dev machine has no
# Go installation. Started by docker-compose.test.yml only.
set -eu

cd /src

echo '--- gofmt ---'
unformatted=$(gofmt -l *.go config constanta dao dto endpoint error middleware router server_config services tests)
if [ -n "$unformatted" ]; then
    echo "unformatted files:" >&2
    echo "$unformatted" >&2
    exit 1
fi
echo 'clean'

echo '--- go vet ---'
go vet ./...

echo '--- go test ---'
go test -mod=vendor -count=1 -timeout 5m ./...

# Read by the compose healthcheck. Reached only when everything above passed.
touch /tests.done

exec tail -f /dev/null
