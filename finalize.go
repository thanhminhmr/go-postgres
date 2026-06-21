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

func Transaction[Connection interface {
	Begin(context.Context) (pgx.Tx, error)
}](
	conn Connection, ctx context.Context, transaction func(ctx context.Context, tx pgx.Tx) error,
) (result error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	result = noCommit{}
	defer func() {
		if result == nil {
			return
		}
		if err := tx.Rollback(ctx); err != nil {
			result = errors.Join(result, err)
		}
	}()
	if err := transaction(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type noCommit struct{}

func (n noCommit) Error() string { return "no commit" }
