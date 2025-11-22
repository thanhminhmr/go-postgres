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
	"github.com/thanhminhmr/go-exception"
	"go.uber.org/fx"
)

const (
	errorConfig  = exception.String("Postgres: Failed parsing config")
	errorConnect = exception.String("Postgres: Failed to connect to database")
	errorMigrate = exception.String("Postgres: Failed to migrate database")
)

// New connects to the PostgreSQL database that are specified in
// the configuration, migrates the database if required.
func New(
	lifecycle fx.Lifecycle,
	config *Config,
	plan MigrationPlan,
) (Database, error) {
	// parse configuration
	parsedConfig, err := parseConfig(config)
	if err != nil {
		return nil, errorConfig.AddCause(err)
	}
	// try connect
	pool, err := pgxpool.NewWithConfig(context.Background(), parsedConfig)
	if err != nil {
		return nil, errorConnect.AddCause(err)
	}
	// create database
	database := &_database{_connection: _connection[*pgxpool.Pool]{pgx: pool}}
	// migrate database
	if len(plan) > 0 {
		// set timeout if any
		var err error
		if config.MigrationTimeout > 0 {
			err = func() error {
				timeout := time.Duration(config.MigrationTimeout) * time.Second
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				return plan.migrate(ctx, database)
			}()
		} else {
			err = plan.migrate(context.Background(), database)
		}
		if err != nil {
			database.close()
			return nil, errorMigrate.AddCause(err)
		}
	}
	// add on start and on stop hook
	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			database.close()
			return nil
		},
	})
	// return connection
	return database, nil
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
			ctx context.Context,
			level tracelog.LogLevel,
			msg string,
			data map[string]any,
		) {
			logger := zerolog.Ctx(ctx)
			var event *zerolog.Event
			switch level {
			case tracelog.LogLevelError:
				event = logger.Error()
			case tracelog.LogLevelWarn:
				event = logger.Warn()
			case tracelog.LogLevelInfo:
				event = logger.Info()
			case tracelog.LogLevelDebug:
				event = logger.Debug()
			case tracelog.LogLevelTrace:
				event = logger.Trace()
			default:
				return
			}
			event.Any("data", data).Msg(msg)
		}),
		LogLevel: logLevel,
	}
	return parsedConfig, nil
}
