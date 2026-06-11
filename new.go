/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"github.com/thanhminhmr/go-common/ctrl"
	"github.com/thanhminhmr/go-exception"
)

// Config defines the options that are used when connecting to a PostgreSQL instance
type Config struct {
	Address          string `env:"POSTGRES_ADDRESS" validate:"required,hostname_port"`
	Username         string `env:"POSTGRES_USER" validate:"required"`
	Password         string `env:"POSTGRES_PASSWORD" validate:"required"`
	DatabaseName     string `env:"POSTGRES_DB_NAME" validate:"required"`
	LogLevel         string `env:"POSTGRES_LOG_LEVEL" validate:"oneof=trace debug info warn error none" default:"info"`
	MigrationTimeout uint   `env:"POSTGRES_MIGRATION_TIMEOUT" default:"30"`
}

const (
	errorConfig  = exception.String("Postgres: Failed parsing config")
	errorConnect = exception.String("Postgres: Failed to connect to database")
	errorMigrate = exception.String("Postgres: Failed to migrate database")
)

func New(config *Config, plan MigrationPlan) Database {
	var databaseResult Database
	ctrl.RegisterWithTimeout(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
		// parse configuration
		parsedConfig, err := parseConfig(config)
		if err != nil {
			panic(errorConfig.AddCause(err))
		}
		// try connect
		pool, err := pgxpool.NewWithConfig(ctx, parsedConfig)
		if err != nil {
			panic(errorConnect.AddCause(err))
		}
		// create database
		database := _database{_connection: _connection[*pgxpool.Pool]{pgx: pool}}
		// migrate database
		if len(plan) > 0 {
			if err := plan.migrate(ctx, database); err != nil {
				database.close()
				panic(errorMigrate.AddCause(err))
			}
		}
		// return the database
		databaseResult = database
		return nil, func(ctx context.Context) { database.close() }
	}, time.Duration(config.MigrationTimeout)*time.Second)
	return databaseResult
}

func parseConfig(config *Config) (*pgxpool.Config, error) {
	// build config url
	targetUrl := &url.URL{
		Scheme: "postgresql",
		Host:   config.Address,
		Path:   config.DatabaseName,
	}
	if config.Username != "" || config.Password != "" {
		targetUrl.User = url.UserPassword(config.Username, config.Password)
	}
	query := targetUrl.Query()
	query.Add("connect_timeout", "15") // seconds
	query.Add("pool_min_conns", "2")
	query.Add("pool_min_idle_conns", "2")
	query.Add("pool_max_conns", "16")
	query.Add("pool_max_conn_lifetime", "1h")
	query.Add("pool_max_conn_lifetime_jitter", "5m")
	query.Add("pool_max_conn_idle_time", "1m")
	query.Add("pool_health_check_period", "15s")
	targetUrl.RawQuery = query.Encode()
	// parse config
	parsedConfig, err := pgxpool.ParseConfig(targetUrl.String())
	if err != nil {
		return nil, err
	}
	// set log level
	logLevel, err := tracelog.LogLevelFromString(config.LogLevel)
	if err != nil {
		return nil, err
	}
	// set log tracer
	parsedConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger: tracelog.LoggerFunc(func(
			ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any,
		) {
			var ctrlLevel zerolog.Level
			switch level {
			case tracelog.LogLevelError:
				ctrlLevel = zerolog.ErrorLevel
			case tracelog.LogLevelWarn:
				ctrlLevel = zerolog.WarnLevel
			case tracelog.LogLevelInfo:
				ctrlLevel = zerolog.InfoLevel
			case tracelog.LogLevelDebug:
				ctrlLevel = zerolog.DebugLevel
			case tracelog.LogLevelTrace:
				ctrlLevel = zerolog.TraceLevel
			default:
				return
			}
			ctrl.Logger(ctx).Level(ctrlLevel).
				Dict("data", (*zerolog.Event)(nil).CreateDict().Fields(data)).
				Msg(msg)
		}),
		LogLevel: logLevel,
	}
	return parsedConfig, nil
}
