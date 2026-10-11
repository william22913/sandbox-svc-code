package server_config

import (
	"database/sql"

	"github.com/nexsoft-git/nexcommon/bundles"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/http/client"
	"github.com/nexsoft-git/nexcommon/http/endpoint/health"
	mdl "github.com/nexsoft-git/nexcommon/http/server/middleware"
	"github.com/nexsoft-git/nexcommon/token"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/get_list_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"
	"github.com/go-redis/redis/v7"

	"github.com/william22913/sandbox-svc-code/config"
)

// validators groups everything the HTTPController is assembled from.
type validators struct {
	basicValidator    basic_validator.BasicValidator
	tagValidator      tag_validator.TagValidator
	listDataValidator get_list_validator.GetListValidator
	apiConnector      client.APIConnector
	jwtValidator      token.UserJWTValidator
	internalToken     token.InternalJWTValidator
	HttpController    *http_validator.HTTPController
}

// serverAttribute is the composition root: every connection, DAO, service and
// endpoint is built once here at startup and threaded down. Nothing dials a
// connection outside Init().
type serverAttribute struct {
	config config.Configuration

	Redis        *redis.Client
	DBConnection *sql.DB

	Bundles       bundles.Bundles
	errorFormator errors.Formator
	Validator     validators

	MetricMiddleware mdl.MetricMiddleware

	HealthChecker health.HealthEndpoint

	listDAO  listDAO
	Services services
}
