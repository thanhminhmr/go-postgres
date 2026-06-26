/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thanhminhmr/go-exception"
)

type MigrationPlan = []MigrationRecord

type MigrationRecord = struct {
	Id      string
	Queries []MigrationQuery
}

type MigrationQuery = struct {
	Sql  string
	Args []any
}

const migrationCreateTable =
// language=PostgreSQL
`CREATE TABLE IF NOT EXISTS _migrations_ (
	id VARCHAR(31) COLLATE "ucs_basic" PRIMARY KEY NOT NULL,
	applied_at TIMESTAMP WITH TIME ZONE NOT NULL
)`

// language=PostgreSQL
const migrationSelectIds = `SELECT id FROM _migrations_`

// language=PostgreSQL
const migrationCreateRecord = `INSERT INTO _migrations_ (id, applied_at) VALUES ($1, $2)`

const errorMigrationRecord = exception.String("Postgres: Failed to create migration record")

func migrateAll(ctx context.Context, database *pgxpool.Pool, plan MigrationPlan) error {
	// create migration table
	if _, err := database.Exec(ctx, migrationCreateTable); err != nil {
		return err
	}
	// query previous migration record ids
	rows, err := database.Query(ctx, migrationSelectIds)
	if err != nil {
		return err
	}
	// scan previous migration record ids into a map
	appliedIds := map[string]struct{}{}
	var appliedId string
	if _, err := pgx.ForEachRow(
		rows, []any{&appliedId}, func() error { appliedIds[appliedId] = struct{}{}; return nil },
	); err != nil {
		return err
	}
	// run migration plans
	for _, record := range plan {
		// check if migration is already existed
		if _, exists := appliedIds[record.Id]; exists {
			continue
		}
		// apply migration
		if _, err := Transaction(database, ctx, func(tx pgx.Tx) (struct{}, error) {
			// run each query
			for _, query := range record.Queries {
				if _, err := tx.Exec(ctx, query.Sql, query.Args...); err != nil {
					return struct{}{}, err
				}
			}
			// create migration record
			if tag, err := tx.Exec(ctx, migrationCreateRecord, record.Id, time.Now()); err != nil {
				return struct{}{}, err
			} else if tag.RowsAffected() != 1 {
				return struct{}{}, errorMigrationRecord
			}
			return struct{}{}, nil
		}); err != nil {
			return err
		}
	}
	return nil
}
