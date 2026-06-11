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

// RowScanner is a library-provided callback function used by user-defined
// [RowCollector] to fill each row data into a provided destination.
type RowScanner = func(destination ...any) error

// RowCollector is a user-defined callback function that will be called for every
// row returned in the query, with a provided [RowScanner] to read each row data.
// This function should call the [RowScanner] once to read the data and collect
// that data into a result collection.
type RowCollector = func(ctx context.Context, scanner RowScanner) error
