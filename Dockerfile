# syntax=docker/dockerfile:1

# Build stage.
#
# `go mod download` is deliberately never run: nexcommon is a private module and
# the build container has no credentials for it. The vendored tree committed to
# the repo is the only trustworthy source in here — fetching would either fail
# outright or, worse, silently resolve to a stale module cache and produce a
# binary built from code that is not what the repo contains.
FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
COPY vendor/ vendor/

# Named explicitly rather than `COPY . .` so the image context stays close to
# the binary's actual inputs — tests, docs and the API artifacts are not part of
# the build.
COPY *.go ./
COPY config/ config/
COPY constanta/ constanta/
COPY dao/ dao/
COPY dto/ dto/
COPY endpoint/ endpoint/
COPY error/ error/
COPY i18n/ i18n/
COPY middleware/ middleware/
COPY sql_migrations/ sql_migrations/
COPY router/ router/
COPY server_config/ server_config/
COPY services/ services/

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -trimpath -o /out/sandbox-svc .

# Runtime stage.
FROM alpine:3.20

# ca-certificates for outbound TLS; tzdata so timestamps are not UTC-only.
RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -u 10001 app

WORKDIR /app
COPY --from=build /out/sandbox-svc .

# The i18n bundles are data, not compiled in. Without this copy the service
# starts fine and answers every error with a raw i18n key instead of a message —
# nothing errors, the responses are just wrong.
COPY --from=build /src/i18n i18n/

# Read at startup by main.go's dbmigration call, so they must be present in the
# runtime image, not only in the build stage.
COPY --from=build /src/sql_migrations sql_migrations/

# config's Server.Temp default (./temp) is used for temp files; it must exist and
# be writable by the unprivileged user.
RUN mkdir -p /app/temp && chown -R app:app /app

USER app
EXPOSE 8080

ENTRYPOINT ["/app/sandbox-svc"]
