package gormext

import (
	"errors"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	// SupportedDrivers lists the SQL database drivers that are supported.
	SupportedDrivers = "'mariadb', 'mysql', 'postgres', 'sqlite'"

	// SQLDriver enum values.
	PostgreSQL SQLDriver = iota
	MySQL
	SQLite
)

type (
	// SQLLoggerLevel represents the logging level for SQL operations.
	SQLLoggerLevel string

	// SQLDriver represents the type of SQL driver.
	SQLDriver uint

	// DatabaseContext holds configuration settings for the database connection.
	DatabaseContext struct {
		loggerLevel SQLLoggerLevel
		driver      SQLDriver
		dsn         string
	}
)

var (
	// sqlLoggerLevels maps custom logger level strings to GORM's logger.LogLevel values.
	sqlLoggerLevels = map[SQLLoggerLevel]logger.LogLevel{
		"silent":  logger.Silent,
		"info":    logger.Info,
		"warning": logger.Warn,
		"warn":    logger.Warn,
		"error":   logger.Error,
	}

	// sqlDriverAliases maps driver alias strings to SQLDriver enum values.
	sqlDriverAliases = map[string]SQLDriver{
		"postgres": PostgreSQL,
		"mysql":    MySQL,
		"mariadb":  MySQL,
		"sqlite":   SQLite,
	}

	// sqlDriverNames maps SQLDriver enum values to their string aliases.
	sqlDriverNames = map[SQLDriver]string{
		PostgreSQL: "postgres",
		MySQL:      "mysql",
		SQLite:     "sqlite",
	}

	// sqlDrivers maps SQLDriver enum values to functions that return a GORM Dialector.
	sqlDrivers = map[SQLDriver]func(string) gorm.Dialector{
		PostgreSQL: postgres.Open,
		MySQL:      mysql.Open,
		SQLite:     sqlite.Open,
	}

	// ErrInvalidSQLDriver is returned when an unsupported SQL driver is provided.
	ErrInvalidSQLDriver = errors.New("invalid SQL database driver")
)

// NewDatabaseContext creates a new DatabaseContext instance using the provided DSN, driver alias, and logger level.
// It returns an error if the DSN or driver is empty, or if the provided driver alias is not supported.
func NewDatabaseContext(dsn, driver, loggerLevel string) (*DatabaseContext, error) {
	// Validate DSN and driver parameters.
	if dsn == "" || driver == "" {
		return nil, fmt.Errorf("invalid database driver and/or DSN")
	}

	// Create a new DatabaseContext with the given DSN.
	ctx := &DatabaseContext{dsn: dsn}

	// Set the SQL driver based on the provided alias.
	if err := ctx.setDriver(driver); err != nil {
		return nil, err
	}

	// Set the logger level.
	ctx.setLoggerLevel(loggerLevel)
	return ctx, nil
}

// setDriver sets the SQLDriver for the DatabaseContext based on the provided driver alias.
func (ctx *DatabaseContext) setDriver(driverAlias string) error {
	if d, ok := sqlDriverAliases[driverAlias]; ok {
		ctx.driver = d
		return nil
	}
	return fmt.Errorf("%w, supported drivers: %s", ErrInvalidSQLDriver, SupportedDrivers)
}

// setLoggerLevel sets the logger level for the DatabaseContext based on the provided string.
func (ctx *DatabaseContext) setLoggerLevel(level string) {
	lvl := SQLLoggerLevel(level)
	if _, ok := sqlLoggerLevels[lvl]; ok {
		ctx.loggerLevel = lvl
	} else {
		ctx.loggerLevel = "info"
	}
}

// GetDSN returns the Data Source Name (DSN) for the database connection.
func (ctx DatabaseContext) GetDSN() string {
	return ctx.dsn
}

// GetDialector returns a function that creates a GORM Dialector based on the current SQL driver and DSN.
func (ctx DatabaseContext) GetDialector() (func() gorm.Dialector, error) {
	if dialector, ok := sqlDrivers[ctx.driver]; ok {
		return func() gorm.Dialector { return dialector(ctx.dsn) }, nil
	}
	return nil, fmt.Errorf("%w, supported drivers: %s", ErrInvalidSQLDriver, SupportedDrivers)
}

// GetLoggerLevel returns the GORM logger.LogLevel corresponding to the current SQL logger level.
// Defaults to logger.Info if the level is not set.
func (ctx DatabaseContext) GetLoggerLevel() logger.LogLevel {
	if level, ok := sqlLoggerLevels[ctx.loggerLevel]; ok {
		return level
	}
	return logger.Info
}

// GetDriverAlias returns the string alias for the current SQL driver.
func (ctx DatabaseContext) GetDriverAlias() string {
	if alias, ok := sqlDriverNames[ctx.driver]; ok {
		return alias
	}
	return ""
}
