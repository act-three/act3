package schema

import (
	"database/sql"
)

func (q *Queries) Tx() *sql.Tx {
	return q.db.(*sql.Tx)
}
