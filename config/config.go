package config

import (
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/nexsoft-git/nexlogger"
	"github.com/nexsoft-git/nexlogger/log"
)

// AppConfig is the process-wide configuration, populated exactly once by init().
var AppConfig Configuration

type Configuration struct {
	Log              Log              `envconfig:"log"`
	Server           Server           `envconfig:"server"`
	Postgresql       Postgresql       `envconfig:"postgresql"`
	Redis            Redis            `envconfig:"redis"`
	Token            Token            `envconfig:"token"`
	Credentials      Credentials      `envconfig:"credentials"`
	AuthorizationAPI AuthorizationAPI `envconfig:"authorization_api"`
}

type Log struct {
	Level int `envconfig:"level" default:"1"`
}

type Server struct {
	Port        int    `envconfig:"port" default:"8080"`
	Version     string `envconfig:"version" default:"1.0.0"`
	ResourceID  string `envconfig:"resource_id" default:"sandbox-svc"`
	Application string `envconfig:"application" default:"sandbox-svc"`
	Temp        string `envconfig:"temp" default:"./temp"`
}

type Postgresql struct {
	Host              string `envconfig:"host" default:"localhost"`
	Port              int    `envconfig:"port" default:"5432"`
	Username          string `envconfig:"username" default:"postgres"`
	Password          string `envconfig:"password" default:"postgres"`
	DBName            string `envconfig:"dbname" default:"sandbox_svc"`
	SSLMode           string `envconfig:"sslmode" default:"disable"`
	DefaultSchema     string `envconfig:"defaultschema" default:"public"`
	MaxOpenConnection int    `envconfig:"maxopenconnection" default:"25"`
	MaxIdleConnection int    `envconfig:"maxidleconnection" default:"25"`
}

type Redis struct {
	Host       string `envconfig:"host" default:"localhost"`
	Port       int    `envconfig:"port" default:"6379"`
	DB         int    `envconfig:"db" default:"0"`
	Password   string `envconfig:"password"`
	Username   string `envconfig:"username"`
	MaxRetries int    `envconfig:"max_retries" default:"3"`
}

// Token holds the shared secrets the HTTP controller and its validators are
// built from. FixedInternalToken is what NewHTTPController compares against for
// FixedTokenValidator routes; InternalKey signs/verifies internal tokens.
type Token struct {
	FixedInternalToken string `envconfig:"fixed_internal_token" default:"sandbox-svc-fixed-token"`
	InternalKey        string `envconfig:"internal_key" default:"sandbox-svc-internal-key"`
}

type Credentials struct {
	ClientID string `envconfig:"client_id" default:"sandbox-svc"`
}

// AuthorizationAPI is only consulted by UserAccessValidator on a Redis token
// cache-miss — see the seed fixture in sql_migrations/seed, which pre-populates the
// token key so this path never fires in the sandbox.
type AuthorizationAPI struct {
	Host        string `envconfig:"host" default:""`
	RoleMapping string `envconfig:"role_mapping" default:""`
}

func init() {
	// The logger must be live before envconfig.Process, or a configuration
	// failure calls log.Fatal and exits with no output at all.
	log.InitiateLogger(nexlogger.New(nexlogger.MultiLevelWriter(os.Stderr)))

	if err := envconfig.Process("sandbox_svc", &AppConfig); err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	nexlogger.InitLoggerModel(
		AppConfig.Server.ResourceID,
		AppConfig.Server.Application,
		AppConfig.Server.Version,
	)
	nexlogger.SetGlobalLevel(nexlogger.Level(AppConfig.Log.Level))

	log.Info().
		Str("application", AppConfig.Server.Application).
		Int("port", AppConfig.Server.Port).
		Msg("configuration loaded")
}
