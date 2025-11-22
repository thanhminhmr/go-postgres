/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import (
	"context"
)

type CommandTag interface {
	String() string
	RowsAffected() int64
	Insert() bool
	Update() bool
	Delete() bool
	Select() bool
}

type CommandTagHandler func(ctx context.Context, tag CommandTag) error

type RowScanner func(destination ...any) error

type RowCollector func(ctx context.Context, row RowScanner) error
