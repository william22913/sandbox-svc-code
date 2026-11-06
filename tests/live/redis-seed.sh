#!/bin/sh
# Seeds the three access tokens the collection uses, then stays alive so
# `docker compose up --wait` has a container to wait on.
#
# nexcommon's UserAccessValidator treats the *raw* Authorization value as the
# Redis key and unmarshals the value into AuthAccessTokenModel; the user id it
# ultimately uses comes from `auth`.oauth.user_id. These values therefore have
# to be the exact stored shape, not a JWT.
#
# live-unseeded-token is deliberately NOT written: the collection's invalid-token
# request needs a real cache miss.
set -eu

USER1='{"rid":1,"auth":"{\"oauth\":{\"user_id\":\"1\",\"client_id\":\"sandbox-svc\",\"scope\":\"\"}}","locale":"en-US"}'
USER2='{"rid":2,"auth":"{\"oauth\":{\"user_id\":\"2\",\"client_id\":\"sandbox-svc\",\"scope\":\"\"}}","locale":"en-US"}'
USER3='{"rid":3,"auth":"{\"oauth\":{\"user_id\":\"3\",\"client_id\":\"sandbox-svc\",\"scope\":\"\"}}","locale":"en-US"}'

redis-cli -h "$REDIS_HOST" SET live-user-1-token "$USER1" > /dev/null
redis-cli -h "$REDIS_HOST" SET live-user-2-token "$USER2" > /dev/null
redis-cli -h "$REDIS_HOST" SET live-user-3-token "$USER3" > /dev/null

touch /seed.done

exec tail -f /dev/null
