package server_config

import (
	"github.com/nexsoft-git/nexcommon/bundles"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/http/client"
	"github.com/nexsoft-git/nexcommon/http/endpoint/health"
	"github.com/nexsoft-git/nexcommon/server/metrics"
	"github.com/nexsoft-git/nexcommon/token"
	"github.com/nexsoft-git/nexcommon/util"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/get_list_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"
	"github.com/nexsoft-git/nexlogger"

	"github.com/william22913/sandbox-svc-code/config"
	"github.com/william22913/sandbox-svc-code/middleware"
)

func NewServerAttribute(
	config config.Configuration,
) serverAttribute {
	return serverAttribute{
		config: config,
	}
}

// Init builds every connection, validator, DAO, service and endpoint, in the
// order later phases depend on. Called exactly once from main.
func (s *serverAttribute) Init() (err error) {
	s.Redis = util.ConnectRedis(
		util.NewRedisParam(
			s.config.Redis.Host,
			s.config.Redis.Port,
		).
			DB(s.config.Redis.DB).
			MaxRetries(s.config.Redis.MaxRetries).
			Password(s.config.Redis.Password).
			Username(s.config.Redis.Username),
	)

	s.DBConnection = util.GetDbConnection(
		util.DBAddressParam().
			Host(s.config.Postgresql.Host).
			Port(s.config.Postgresql.Port).
			DBName(s.config.Postgresql.DBName).
			DefaultSchema(s.config.Postgresql.DefaultSchema).
			Username(s.config.Postgresql.Username).
			Password(s.config.Postgresql.Password),
	)

	s.Bundles, err = bundles.NewBundles("i18n", "en-US")
	if err != nil {
		return
	}

	s.Validator.tagValidator = tag_validator.NewTagValidator()
	s.Validator.basicValidator = basic_validator.BasicValidator{}
	s.Validator.listDataValidator = get_list_validator.NewGetListValidator(s.Validator.basicValidator)

	s.Validator.HttpController = http_validator.NewHTTPController(
		s.config.Token.FixedInternalToken,
	)

	s.errorFormator = errors.NewErrorFormator(
		s.Bundles,
	).DefaultInternalCode("E-5-SBX-SRV-001").
		DefaultLanguage("en-US").
		Version(s.config.Server.Version)

	nexlogger.InitLoggerModel(
		s.config.Server.ResourceID,
		s.config.Server.Application,
		s.config.Server.Version,
	)

	s.Validator.HttpController.Version(s.config.Server.Version)
	s.Validator.HttpController.BasicValidator(s.Validator.basicValidator)
	s.Validator.HttpController.TagValidator(s.Validator.tagValidator)
	s.Validator.HttpController.ListDataValidator(s.Validator.listDataValidator)
	s.Validator.HttpController.LogEndpointsHit()
	s.Validator.HttpController.Formator(s.errorFormator)
	s.Validator.HttpController.Redis(s.Redis)

	// Built once and shared: NewMetrics registers its collectors on the default
	// Prometheus registry, so constructing it twice would silently leave the
	// second object unregistered (Register's error is ignored upstream).
	metricRegistry := metrics.NewMetrics()
	s.Validator.apiConnector = client.NewAPIConnector(metricRegistry)

	// The user-token validator resolves the caller's identity out of Redis (see
	// UserAccessValidator in nexcommon-go-standards). It only falls through to
	// JWT parsing + the external role-mapping API on a cache miss, which the
	// token seed fixture avoids entirely.
	s.Validator.jwtValidator = token.NewJWTTokenValidator(
		token.JWTValidatorConfig{
			ResourceID:  s.config.Server.ResourceID,
			FixedToken:  s.config.Token.FixedInternalToken,
			ClientID:    s.config.Credentials.ClientID,
			InternalKey: s.config.Token.InternalKey,
			AuthorizationAPI: token.AuthApi{
				Host:        s.config.AuthorizationAPI.Host,
				RoleMapping: token.Path{Uri: s.config.AuthorizationAPI.RoleMapping, Method: "POST"},
			},
		},
		s.Validator.basicValidator,
		s.Validator.apiConnector,
		s.Redis,
	)
	s.Validator.HttpController.TokenValidator(s.Validator.jwtValidator)

	s.Validator.internalToken = token.NewJWTTokenInternalValidator(
		token.JWTValidatorConfig{
			ResourceID:  s.config.Server.ResourceID,
			InternalKey: s.config.Token.InternalKey,
			ClientID:    s.config.Credentials.ClientID,
		},
		s.Validator.basicValidator,
		s.Validator.apiConnector,
		s.Redis,
	)
	s.Validator.HttpController.InternalTokenValidator(s.Validator.internalToken)

	s.MetricMiddleware = middleware.NewStatusAwareMetricsMiddleware()
	s.MetricMiddleware.SupportMetrics(metricRegistry)

	s.HealthChecker = health.NewHealthEndpoint(
		health.NewListTools("db", health.NewDBConnChecker(s.DBConnection)),
		health.NewListTools("redis", health.NewRedisConnChecker(s.Redis.Conn())),
	)

	s.InitDAO()
	s.InitServices()

	return
}
