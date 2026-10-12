package util

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/nexsoft-git/nexlogger/log"
)

// passwordInConnStr matches the "password = <value>" segment the
// Password() builder above appends to a connection string, so it can
// be redacted before that string ever reaches a log line. The DB
// password is user-supplied and has no fixed character set, so this
// matches up to the next space (the same separator every other
// builder method here uses between key/value pairs) rather than
// trying to enumerate what a password can contain.
var passwordInConnStr = regexp.MustCompile(`(?i)password\s*=\s*\S+`)

func redactPassword(connStr string) string {
	return passwordInConnStr.ReplaceAllString(connStr, "password=***")
}

func DBAddressParam() *dBAddressParam {
	return &dBAddressParam{
		maxOpenConnection: 500,
		maxIdleConnection: 100,
	}
}

type dBAddressParam struct {
	address           string
	defaultSchema     string
	maxOpenConnection int
	maxIdleConnection int
}

func (d *dBAddressParam) Username(username string) *dBAddressParam {
	d.address += fmt.Sprintf(" user = %s", username)
	return d
}

func (d *dBAddressParam) Password(password string) *dBAddressParam {
	d.address += fmt.Sprintf(" password = %s", password)
	return d
}

func (d *dBAddressParam) DBName(dbName string) *dBAddressParam {
	d.address += fmt.Sprintf(" dbname = %s", dbName)
	return d
}

func (d *dBAddressParam) SSLMode(sslMode string) *dBAddressParam {
	d.address += fmt.Sprintf(" sslmode = %s", sslMode)
	return d
}

func (d *dBAddressParam) Host(host string) *dBAddressParam {
	d.address += fmt.Sprintf(" host = %s", host)
	return d
}

func (d *dBAddressParam) Port(port int) *dBAddressParam {
	if port > 0 {
		d.address += fmt.Sprintf(" port = %d", port)
	}

	return d
}

func (d *dBAddressParam) Address(address string) *dBAddressParam {
	d.address = address
	return d
}

func (d *dBAddressParam) DefaultSchema(schema string) *dBAddressParam {
	d.defaultSchema = schema
	return d
}

func (d *dBAddressParam) MaxOpenConnection(con int) *dBAddressParam {
	d.maxOpenConnection = con
	return d
}

func (d *dBAddressParam) MaxIdleConnection(con int) *dBAddressParam {
	d.maxIdleConnection = con
	return d
}

type DBInfo struct {
	instance      *sql.DB
	driver        string
	connectionStr string
	setParams     []string
}

var instance *sql.DB

func GetDbConnection(
	param *dBAddressParam,
) *sql.DB {
	_dbInfo := DBInfo{
		instance:      nil,
		driver:        "pgx",
		connectionStr: param.address,
	}

	if param.defaultSchema != "" {
		_dbInfo.setParams = []string{fmt.Sprintf("search_path = '%s'", param.defaultSchema)}
	}

	_db, _err := getInstance(_dbInfo)
	if _err != nil {
		log.Fatal().
			Err(_err).
			Msg("Error Find When Connect to Database")
	}

	_db.SetMaxOpenConns(param.maxOpenConnection)
	_db.SetMaxIdleConns(param.maxIdleConnection)

	return _db
}

func getInstance(
	connInfo DBInfo,
) (
	*sql.DB,
	error,
) {
	var _errOpen error

	dbConnStr := connInfo.connectionStr
	if connInfo.setParams != nil && len(connInfo.setParams) > 0 {

		for _, _param := range connInfo.setParams {
			dbConnStr = dbConnStr + " " + _param
		}
	}

	instance, _errOpen = sql.Open(connInfo.driver, dbConnStr)

	if _errOpen != nil {
		// connInfo.connectionStr previously went into this log line
		// verbatim, including the literal "password = <value>"
		// segment the Password() builder appends above -- every
		// connection failure logged the real DB password in
		// plaintext. Log a redacted copy instead; connInfo itself is
		// a local value-type copy (passed by value into this
		// function), so this doesn't touch the caller's own data.
		redacted := connInfo
		redacted.connectionStr = redactPassword(redacted.connectionStr)
		log.Error().Err(_errOpen).Msg(fmt.Sprintf("Connect failed to DB %v", redacted))
		instance = nil
	}

	return instance, _errOpen
}
