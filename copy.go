/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"

	"github.com/thanhminhmr/go-exception"
)

const (
	errorCopyAll           = exception.String("Postgres: CopyAll failed")
	errorCopyAllEverything = exception.String("Postgres: CopyAll failed, cannot copy everything from source")
	errorCopyAny           = exception.String("Postgres: CopyAny failed")
)

func CopyAll[T any](
	connection Connection,
	ctx context.Context,
	tableName string,
	columnNames []string,
	input []T,
	outputMapper SliceMapper[T],
) (errorResult error) {
	// create transaction
	transaction, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Finalize(ctx, &errorResult)
	// create source
	source := &fromSlice[T]{
		mapper: outputMapper,
		input:  input,
		output: make([]any, len(columnNames)),
		index:  -1,
	}
	// call raw copy and check the result
	if count, err := transaction.internalCopyFrom(ctx, tableName, columnNames, source); err != nil {
		return errorCopyAll.AddCause(err)
	} else if count != int64(len(input)) {
		return errorCopyAllEverything
	}
	return nil
}

func CopyAny[T any](
	connection Connection,
	ctx context.Context,
	tableName string,
	columnNames []string,
	input []T,
	outputMapper SliceMapper[T],
) (int64, error) {
	source := &fromSlice[T]{
		mapper: outputMapper,
		input:  input,
		output: make([]any, len(columnNames)),
		index:  -1,
	}
	count, err := connection.internalCopyFrom(ctx, tableName, columnNames, source)
	if err != nil {
		return count, errorCopyAny.AddCause(err)
	}
	return count, nil
}

type SliceMapper[T any] func(output []any, input T)

type fromSlice[T any] struct {
	mapper SliceMapper[T]
	input  []T
	output []any
	index  int
}

func (copy *fromSlice[T]) Next() bool {
	copy.index++
	return copy.index < len(copy.input)
}

func (copy *fromSlice[T]) Values() ([]any, error) {
	clear(copy.output)
	copy.mapper(copy.output, copy.input[copy.index])
	return copy.output, nil
}

func (copy *fromSlice[T]) Err() error {
	return nil
}
