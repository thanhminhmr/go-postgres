/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type Database interface {
	Connection

	close()
}

type _database struct {
	_connection[*pgxpool.Pool]
}

func (d _database) close() {
	d.pgx.Close()
}
