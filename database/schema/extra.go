// Package schema contains generated code from sqlc.
// Types and functions defined in this package
// should be consumed only by model code;
// other packages should interace with the database via package model.
package schema

import (
	"database/sql"
)

func (q *Queries) Tx() *sql.Tx {
	return q.db.(*sql.Tx)
}
