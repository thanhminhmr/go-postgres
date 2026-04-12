/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"
	"sync/atomic"

	"github.com/thanhminhmr/go-exception"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Connection interface {
	// Begin starts a transaction.
	Begin(ctx context.Context) (Transaction, error)

	// Batch creates a batch of commands.
	Batch(ctx context.Context) Batch

	// Exec execute the command.
	Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)

	// Query scan the result rows by calling the collector repeatedly.
	Query(ctx context.Context, collector RowCollector, sql string, args ...any) (CommandTag, error)

	// QueryRow expects the result is at most one row. Returns nil [RowScanner] if
	// the result is empty.
	QueryRow(ctx context.Context, sql string, args ...any) (RowScanner, error)

	// internalSendBatch internal function to send a batch
	internalSendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults

	// internalCopyFrom internal function to copy any data from source to database
	internalCopyFrom(
		ctx context.Context,
		tableName string,
		columnNames []string,
		source pgx.CopyFromSource,
	) (int64, error)
}

type _pgxConnection interface {
	Begin(context.Context) (pgx.Tx, error)
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type _connection[pgxConnection _pgxConnection] struct {
	pgx pgxConnection
}

const (
	errorBegin        = exception.String("Postgres: Begin transaction failed")
	errorExec         = exception.String("Postgres: Exec failed")
	errorQuery        = exception.String("Postgres: Query failed")
	errorQueryRow     = exception.String("Postgres: QueryRow failed")
	errorQueryRowMany = exception.String("Postgres: QueryRow failed, more than one row returned")
)

func (c _connection[pgxConnection]) Begin(ctx context.Context) (Transaction, error) {
	if tx, err := c.pgx.Begin(ctx); err == nil {
		return &_transaction{
			_connection: _connection[pgx.Tx]{
				pgx: tx,
			},
		}, nil
	} else {
		return nil, errorBegin.AddCause(err)
	}
}

func (c _connection[pgxConnection]) Batch(ctx context.Context) Batch {
	return &_batch{
		ctx:        ctx,
		batch:      pgx.Batch{},
		connection: atomic.Value{},
	}
}

func (c _connection[pgxConnection]) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	if tag, err := c.pgx.Exec(ctx, sql, args...); err != nil {
		return nil, errorExec.AddCause(err)
	} else {
		return &tag, nil
	}
}

func (c _connection[pgxConnection]) Query(
	ctx context.Context,
	collector RowCollector,
	sql string,
	args ...any,
) (tag CommandTag, errorResult error) {
	if collector == nil {
		panic("BUG: collector is nil")
	}
	if rows, err := c.pgx.Query(ctx, sql, args...); err != nil {
		return nil, errorQuery.AddCause(err)
	} else {
		var ex exception.Exception
		defer func() {
			rows.Close()
			if err := rows.Err(); err != nil {
				if ex != nil {
					ex = ex.AddSuppressed(err)
				} else {
					ex = errorQuery.AddCause(err)
				}
			} else if ex == nil {
				tag = rows.CommandTag()
			}
			errorResult = ex
		}()
		for rows.Next() {
			if err := collector(ctx, rows.Scan); err != nil {
				ex = errorQuery.AddCause(err)
				return
			}
		}
		return
	}
}

func (c _connection[pgxConnection]) QueryRow(ctx context.Context, sql string, args ...any) (RowScanner, error) {
	if rows, err := c.pgx.Query(ctx, sql, args...); err != nil {
		return nil, errorQueryRow.AddCause(err)
	} else {
		if !rows.Next() {
			return nil, nil
		}
		return func(destination ...any) (errorResult error) {
			var ex exception.Exception
			defer func() {
				rows.Close()
				if err := rows.Err(); err != nil {
					if ex != nil {
						ex = ex.AddSuppressed(err)
					} else {
						ex = errorQueryRow.AddCause(err)
					}
				}
				errorResult = ex
			}()
			if err := rows.Scan(destination...); err != nil {
				ex = errorQueryRow.AddCause(err)
				return
			}
			if rows.Next() {
				ex = errorQueryRowMany
				return
			}
			return
		}, nil
	}
}

func (c _connection[pgxConnection]) internalSendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return c.pgx.SendBatch(ctx, batch)
}

func (c _connection[pgxConnection]) internalCopyFrom(
	ctx context.Context,
	tableName string,
	columnNames []string,
	source pgx.CopyFromSource,
) (int64, error) {
	return c.pgx.CopyFrom(ctx, pgx.Identifier{tableName}, columnNames, source)
}
