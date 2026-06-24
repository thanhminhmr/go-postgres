/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func Transaction[Conn Querier, Result any](
	conn Conn, ctx context.Context, transaction func(tx pgx.Tx) (Result, error),
) (result Result, resultErr error) {
	tx, resultErr := conn.Begin(ctx)
	if resultErr != nil {
		return result, resultErr
	}
	resultErr = noCommit{}
	defer func() {
		if resultErr != nil {
			if err := tx.Rollback(ctx); err != nil {
				resultErr = errors.Join(resultErr, err)
			}
		}
	}()
	result, err := transaction(tx)
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

type noCommit struct{}

func (n noCommit) Error() string { return "no commit" }
