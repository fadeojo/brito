package mig

import (
	"database/sql"
	"fmt"
	"os"
)

// sqlDialect abstracts the details of specific SQL dialects
// for mig's few SQL specific statements
type sqlDialect interface {
	createVersionTableSQL() string // sql string to create the mig_migrations table
	insertVersionSQL() string      // sql string to insert the initial version table row
	versionQuery(db *sql.DB) (*sql.Rows, error)
}

var dialect sqlDialect = &postgresDialect{}

func getDialect() sqlDialect {
	return dialect
}

func setDialect(d string) error {
	switch d {
	case "postgres":
		dialect = &postgresDialect{}
	case "mysql":
		dialect = &mySQLDialect{}
	case "sqlite3":
		fmt.Println("sqlite3 not supported")
		os.Exit(1)
		//dialect = &sqlite3Dialect{}
	default:
		return fmt.Errorf("%q: unknown dialect", d)
	}

	return nil
}

type postgresDialect struct{}
type mySQLDialect struct{}
type sqlite3Dialect struct{}

func (postgresDialect) createVersionTableSQL() string {
	return `CREATE TABLE mig_migrations (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (postgresDialect) insertVersionSQL() string {
	return "INSERT INTO mig_migrations (version_id, is_applied) VALUES ($1, $2);"
}

func (postgresDialect) versionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from mig_migrations ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (mySQLDialect) createVersionTableSQL() string {
	return `CREATE TABLE mig_migrations (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (mySQLDialect) insertVersionSQL() string {
	return "INSERT INTO mig_migrations (version_id, is_applied) VALUES (?, ?);"
}

func (mySQLDialect) versionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from mig_migrations ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (sqlite3Dialect) createVersionTableSQL() string {
	return `CREATE TABLE mig_migrations (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
            );`
}

func (sqlite3Dialect) insertVersionSQL() string {
	return "INSERT INTO mig_migrations (version_id, is_applied) VALUES (?, ?);"
}

func (sqlite3Dialect) versionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from mig_migrations ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}
