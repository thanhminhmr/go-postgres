/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thanhminhmr/go-exception"
)

type Transaction interface {
	Connection

	// Finalize safely concludes a database transaction by either committing or
	// rolling back. It rolls back the transaction if any of the following conditions
	// are met: the input error is not null, a panic occurred earlier in execution,
	// the provided context has an error (canceled or expired), or the commit
	// operation itself fails. This ensures reliable and consistent transaction
	// handling.
	Finalize(ctx context.Context, errorResult *error)
}

type _transaction struct {
	_connection[pgx.Tx]
}

const (
	errorRollbackContext = exception.String("Postgres: Finalize failed, transaction rollback on context error")
	errorRollbackCommit  = exception.String("Postgres: Finalize failed, transaction rollback on commit error")
	errorRollback        = exception.String("Postgres: Finalize failed, transaction rollback on error")
	errorRollbackFailed  = exception.String("Postgres: Finalize failed, transaction rollback also failed")
)

func (t _transaction) Finalize(ctx context.Context, errorResult *error) {
	if errorResult == nil {
		panic("BUG: errorResult is nil")
	}
	var recovered any
	var ex exception.Exception
	// check for commit condition and try to commit
	if *errorResult != nil {
		// transaction rollback on error
	} else if recovered = recover(); recovered != nil {
		// transaction rollback on panic without changing anything
	} else if err := ctx.Err(); err != nil {
		ex = errorRollbackContext.AddCause(err)
	} else if err := t.pgx.Commit(ctx); err != nil {
		ex = errorRollbackCommit.AddCause(err)
	} else {
		return
	}
	// either commit condition failed or commit failed, try rolling back
	if err := t.pgx.Rollback(ctx); err != nil && recovered == nil {
		if ex == nil {
			// only wrap the error if needed
			var ok bool
			//goland:noinspection GoTypeAssertionOnErrors
			if ex, ok = (*errorResult).(exception.Exception); !ok {
				ex = errorRollback.AddCause(*errorResult)
			}
		}
		ex = ex.AddSuppressed(errorRollbackFailed.AddCause(err))
	}
	// if recovered from panic, re-panic as it
	if recovered != nil {
		panic(recovered)
	}
	// if error got wrapped, return the wrapped error
	if ex != nil {
		*errorResult = ex
	}
}
