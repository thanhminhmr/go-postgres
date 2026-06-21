/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"
	"iter"
	"maps"

	"github.com/jackc/pgx/v5"
	"github.com/thanhminhmr/go-exception"
)

const (
	errorCopyAll = exception.String("Postgres: CopyAll failed")
	errorCopyAny = exception.String("Postgres: CopyAny failed")
)

func CopyAllFromSlice[T any, Connection interface {
	Begin(context.Context) (pgx.Tx, error)
}](
	connection Connection, ctx context.Context, tableName string,
	columnNames []string, input []T, outputMapper FromSliceValue[T],
) (errorResult error) {
	return Transaction(connection, ctx, func(ctx context.Context, tx pgx.Tx) error {
		// create source
		source := fromSlice[T]{
			mapper: outputMapper,
			input:  input,
			output: make([]any, len(columnNames)),
			index:  -1,
		}
		// call copy and check the result
		if count, err := tx.CopyFrom(ctx, pgx.Identifier{tableName}, columnNames, &source); err != nil {
			return errorCopyAll.AddCause(err)
		} else if count != int64(len(input)) {
			return errorCopyAll
		}
		return nil
	})
}

func CopyAnyFromSlice[T any, Connection interface {
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
}](
	connection Connection, ctx context.Context, tableName string,
	columnNames []string, input []T, outputMapper FromSliceValue[T],
) (int64, error) {
	// create source
	source := fromSlice[T]{
		mapper: outputMapper,
		input:  input,
		output: make([]any, len(columnNames)),
		index:  -1,
	}
	// call copy and check the result
	count, err := connection.CopyFrom(ctx, pgx.Identifier{tableName}, columnNames, &source)
	if err != nil {
		return count, errorCopyAny.AddCause(err)
	}
	return count, nil
}

type FromSliceValue[T any] = func(output []any, input T) error

type fromSlice[T any] struct {
	mapper FromSliceValue[T]
	input  []T
	output []any
	index  int
	err    error
}

func (f *fromSlice[T]) Next() bool {
	f.index++
	return f.index < len(f.input)
}

func (f *fromSlice[T]) Values() ([]any, error) {
	clear(f.output)
	f.err = f.mapper(f.output, f.input[f.index])
	return f.output, f.err
}

func (f *fromSlice[T]) Err() error {
	return f.err
}

func CopyAllFromMap[K comparable, V any, Connection interface {
	Begin(context.Context) (pgx.Tx, error)
}](
	connection Connection, ctx context.Context, tableName string,
	columnNames []string, input map[K]V, outputMapper FromMapKeyValue[K, V],
) error {
	return Transaction(connection, ctx, func(ctx context.Context, tx pgx.Tx) error {
		next, stop := iter.Pull2(maps.All(input))
		defer stop()
		// create source
		source := fromMap[K, V]{
			mapper: outputMapper,
			output: make([]any, len(columnNames)),
			next:   next,
		}
		// call copy and check the result
		count, err := tx.CopyFrom(ctx, pgx.Identifier{tableName}, columnNames, &source)
		if err != nil {
			return errorCopyAll.AddCause(err)
		} else if count != int64(len(input)) {
			return errorCopyAll
		}
		return nil
	})
}

func CopyAnyFromMap[K comparable, V any, Connection interface {
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
}](
	connection Connection, ctx context.Context, tableName string,
	columnNames []string, input map[K]V, outputMapper FromMapKeyValue[K, V],
) (int64, error) {
	next, stop := iter.Pull2(maps.All(input))
	defer stop()
	// create source
	source := fromMap[K, V]{
		mapper: outputMapper,
		output: make([]any, len(columnNames)),
		next:   next,
	}
	// call copy and check the result
	count, err := connection.CopyFrom(ctx, pgx.Identifier{tableName}, columnNames, &source)
	if err != nil {
		return count, errorCopyAny.AddCause(err)
	}
	return count, nil
}

type FromMapKeyValue[K comparable, V any] = func(output []any, key K, value V) error

type fromMap[K comparable, V any] struct {
	mapper FromMapKeyValue[K, V]
	output []any
	next   func() (K, V, bool)
	err    error
}

func (f *fromMap[K, V]) Next() bool {
	k, v, exists := f.next()
	if !exists {
		return false
	}
	f.err = f.mapper(f.output, k, v)
	return true
}

func (f *fromMap[K, V]) Values() ([]any, error) {
	return f.output, f.err
}

func (f *fromMap[K, V]) Err() error {
	return f.err
}
