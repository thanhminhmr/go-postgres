/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

type CommandTag = pgconn.CommandTag

type CommandTagHandler = func(ctx context.Context, tag CommandTag) error

type RowScanner = func(destination ...any) error

type RowCollector = func(ctx context.Context, row RowScanner) error
