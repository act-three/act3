package database

import (
	"database/sql"
	"fmt"
)

// TableStat holds the name and row count for a single table.
type TableStat struct {
	Name     string
	RowCount int64
}

// TableStats returns the row count for each user table in the
// database. It queries sqlite_master directly and does not
// depend on sqlc-generated code.
func TableStats(db *sql.DB) ([]TableStat, error) {
	rows, err := db.Query(
		`SELECT name FROM sqlite_master
		 WHERE type = 'table'
		   AND name NOT LIKE 'sqlite_%'
		 ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats := make([]TableStat, len(names))
	for i, name := range names {
		var count int64
		err := db.QueryRow(
			fmt.Sprintf("SELECT COUNT(*) FROM %q", name),
		).Scan(&count)
		if err != nil {
			return nil, err
		}
		stats[i] = TableStat{Name: name, RowCount: count}
	}
	return stats, nil
}
