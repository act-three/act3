// Package database opens and updates the act3 SQLite database.
//
// The schema is a sequence of DDL updates
// written as files in ddl/NNN_name.up.sql.
// Unapplied updates are applied in order when the database is opened.
// File frozen.txt records the updates
// that can be deployed to production.
// A frozen update is immutable.
//
// To change the schema,
// edit the newest update if it is still unfrozen
// (its name is not yet in frozen.txt).
// Otherwise, add a new file ddl/NNN_name.up.sql
// under the next unused integer NNN and a descriptive name.
// Each update file should have a one-line descriptive comment header,
// followed by one or more DDL or DML statements.
//
//	-- add extended running time
//	ALTER TABLE Movie ADD COLUMN ExtendedRuntime INTEGER;
//	ALTER TABLE Episode ADD COLUMN ExtendedRuntime INTEGER;
//
// Before an update can be deployed to production,
// it must be frozen with [ily.dev/act3/cmd/freeze].
package database
