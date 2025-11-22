/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

// Config defines the options that are used when connecting to a PostgreSQL instance
type Config struct {
	Address          string `env:"POSTGRES_ADDRESS" validate:"required,hostname_port"`
	Username         string `env:"POSTGRES_USER" validate:"required"`
	Password         string `env:"POSTGRES_PASSWORD" validate:"required"`
	DatabaseName     string `env:"POSTGRES_DB_NAME" validate:"required"`
	LogLevel         string `env:"POSTGRES_LOG_LEVEL" validate:"oneof=trace debug info warn error none" default:"info"`
	MigrationTimeout uint   `env:"POSTGRES_MIGRATION_TIMEOUT" default:"30"`
}
