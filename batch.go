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

type Batch interface {
	Exec(handler CommandTagHandler, sql string, args ...any)
	Query(collector RowCollector, handler CommandTagHandler, sql string, args ...any)
	QueryRow(collector RowCollector, sql string, args ...any)
	Send() error
	__()
}

type _batch struct {
	ctx        context.Context
	batch      pgx.Batch
	connection atomic.Value
}

const (
	errorBatchExec          = exception.String("Postgres: Exec in batch failed")
	errorBatchQuery         = exception.String("Postgres: Query in batch failed")
	errorBatchQueryRow      = exception.String("Postgres: QueryRow in batch failed")
	errorBatchQueryRowEmpty = exception.String("Postgres: QueryRow in batch failed, no row returned")
	errorBatchQueryRowMany  = exception.String("Postgres: QueryRow in batch failed, more than one row returned")
	errorBatchSend          = exception.String("Postgres: Send batch failed")
)

func (b *_batch) Exec(handler CommandTagHandler, sql string, args ...any) {
	query := b.batch.Queue(sql, args...)
	if handler != nil {
		query.Exec(func(tag pgconn.CommandTag) error {
			if err := handler(b.ctx, tag); err != nil {
				return errorBatchExec.AddCause(err)
			}
			return nil
		})
	}
}

func (b *_batch) Query(collector RowCollector, handler CommandTagHandler, sql string, args ...any) {
	if collector == nil {
		panic("BUG: collector is nil")
	}
	b.batch.Queue(sql, args...).Query(func(rows pgx.Rows) (errorResult error) {
		var ex exception.Exception
		defer func() {
			rows.Close()
			if err := rows.Err(); err != nil {
				if ex != nil {
					ex = ex.AddSuppressed(err)
				} else {
					ex = errorBatchQuery.AddCause(err)
				}
			} else if ex == nil && handler != nil {
				if err := handler(b.ctx, rows.CommandTag()); err != nil {
					ex = errorBatchQuery.AddCause(err)
				}
			}
			errorResult = ex
		}()
		for rows.Next() {
			if err := collector(b.ctx, rows.Scan); err != nil {
				ex = errorBatchQuery.AddCause(err)
				return
			}
		}
		return
	})
}

func (b *_batch) QueryRow(collector RowCollector, sql string, args ...any) {
	if collector == nil {
		panic("BUG: collector is nil")
	}
	b.batch.Queue(sql, args...).Query(func(rows pgx.Rows) (errorResult error) {
		var ex exception.Exception
		defer func() {
			rows.Close()
			if err := rows.Err(); err != nil {
				if ex != nil {
					ex = ex.AddSuppressed(err)
				} else {
					ex = errorBatchQueryRow.AddCause(err)
				}
			}
			errorResult = ex
		}()
		if !rows.Next() {
			ex = errorBatchQueryRowEmpty
			return
		}
		if err := collector(b.ctx, rows.Scan); err != nil {
			ex = errorBatchQueryRow.AddCause(err)
			return
		}
		if rows.Next() {
			ex = errorBatchQueryRowMany
			return
		}
		return
	})
}

func (b *_batch) Send() error {
	if connection, _ := b.connection.Swap(nil).(Database); connection == nil {
		panic("BUG: batch already sent")
	} else if err := connection.internalSendBatch(b.ctx, &b.batch).Close(); err != nil {
		return errorBatchSend.AddCause(err)
	}
	return nil
}

func (b *_batch) __() {}
