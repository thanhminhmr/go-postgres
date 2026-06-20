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

type ResultCollector[Value any] = func(ctx context.Context, scanner RowScanner, result *Value) error

func QueryOne[Value any](
	conn Connection, ctx context.Context, collector ResultCollector[Value], query string, args ...any,
) (*Value, error) {
	scanner, err := conn.QueryRow(ctx, query, args)
	if err != nil {
		return nil, err
	} else if scanner == nil {
		return nil, nil
	}
	var result Value
	if err := collector(ctx, scanner, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func QueryMany[Value any](
	conn Connection, ctx context.Context, collector ResultCollector[Value], query string, args ...any,
) ([]Value, CommandTag, error) {
	var results []Value
	tag, err := conn.Query(ctx, func(ctx context.Context, scanner RowScanner) error {
		results = append(results, make([]Value, 1)...)
		return collector(ctx, scanner, &results[len(results)-1])
	}, query, args)
	if err != nil {
		return nil, CommandTag{}, err
	}
	return results, tag, nil
}
