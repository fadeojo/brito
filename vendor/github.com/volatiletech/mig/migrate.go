package mig

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"time"
)

var (
	ErrNoCurrentVersion = errors.New("no current version found")
	ErrNoNextVersion    = errors.New("no next version found")
)

var Log io.Writer

func init() {
	Log = ioutil.Discard
}

type errNoMigration struct{}

func (e errNoMigration) Error() string {
	return "no migrations to execute"
}

// IsNoMigrationError returns true if the error type is of
// errNoMigration, indicating that there is no migration to run
func IsNoMigrationError(err error) bool {
	_, ok := err.(errNoMigration)
	if ok {
		return ok
	}

	return false
}

type migrations []*migration

// helpers so we can use pkg sort
func (m migrations) Len() int      { return len(m) }
func (m migrations) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m migrations) Less(i, j int) bool {
	if m[i].version == m[j].version {
		panic(fmt.Sprintf("mig: duplicate version %v detected:\n%v\n%v", m[i].version, m[i].source, m[j].source))
	}
	return m[i].version < m[j].version
}

func (m migrations) current(current int64) (*migration, error) {
	for i, migration := range m {
		if migration.version == current {
			return m[i], nil
		}
	}

	return nil, ErrNoCurrentVersion
}

func (m migrations) next(current int64) (*migration, error) {
	for i, migration := range m {
		if migration.version > current {
			return m[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

func (m migrations) last() (*migration, error) {
	if len(m) == 0 {
		return nil, ErrNoNextVersion
	}

	return m[len(m)-1], nil
}

func (m migrations) String() string {
	str := ""
	for _, migration := range m {
		str += fmt.Sprintln(migration)
	}
	return str
}

// collect all the valid looking migration scripts in the migrations folder,
// and order them by version.
func collectMigrations(dirpath string, current, target int64) (migrations, error) {
	var migrations migrations

	// extract the numeric component of each migration,
	// filter out any uninteresting files,
	// and ensure we only have one file per migration version.
	files, err := filepath.Glob(dirpath + "/*.sql")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		v, err := numericComponent(file)
		if err != nil {
			return nil, err
		}
		if versionFilter(v, current, target) {
			migration := &migration{version: v, next: -1, previous: -1, source: file}
			migrations = append(migrations, migration)
		}
	}

	migrations = sortAndConnectMigrations(migrations)

	return migrations, nil
}

// sortAndConnectMigrations sorts the migrations based on the version numbers
// and creates a linked list between each migration.
func sortAndConnectMigrations(migrations migrations) migrations {
	// Sort the migrations based on version
	sort.Sort(migrations)

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range migrations {
		prev := int64(-1)
		if i > 0 {
			prev = migrations[i-1].version
			migrations[i-1].next = m.version
		}
		migrations[i].previous = prev
	}

	return migrations
}

// versionFilter returns true if v is within the current version and target
// version range.
func versionFilter(v, current, target int64) bool {
	if target > current {
		return v > current && v <= target
	}

	if target < current {
		return v <= current && v > target
	}

	return false
}

// Create the mig_migrations table
// and insert the initial 0 value into it
func createVersionTable(db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	d := getDialect()

	if _, err := txn.Exec(d.createVersionTableSQL()); err != nil {
		txn.Rollback()
		return err
	}

	version := 0
	applied := true
	if _, err := txn.Exec(d.insertVersionSQL(), version, applied); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// getVersion retrieves the current version for this database.
// Create and initialize the database migration table if it doesn't exist.
func getVersion(db *sql.DB) (int64, error) {
	rows, err := getDialect().versionQuery(db)
	if err != nil {
		return 0, createVersionTable(db)
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.

	toSkip := make([]int64, 0)

	for rows.Next() {
		var row migrationRecord
		if err = rows.Scan(&row.versionId, &row.isApplied); err != nil {
			return 0, fmt.Errorf("error scanning rows: %s", err)
		}

		// have we already marked this version to be skipped?
		skip := false
		for _, v := range toSkip {
			if v == row.versionId {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// if version has been applied we're done
		if row.isApplied {
			return row.versionId, nil
		}

		// latest version of migration has not been applied.
		toSkip = append(toSkip, row.versionId)
	}

	panic("unreachable")
}

func getMigrationStatus(db *sql.DB, version int64) string {
	var row migrationRecord
	q := fmt.Sprintf("SELECT tstamp, is_applied FROM mig_migrations WHERE version_id=%d ORDER BY tstamp DESC LIMIT 1", version)
	e := db.QueryRow(q).Scan(&row.tstamp, &row.isApplied)

	if e != nil && e != sql.ErrNoRows {
		panic(e)
	}

	var appliedAt string

	if row.isApplied {
		appliedAt = row.tstamp.Format(time.ANSIC)
	} else {
		appliedAt = "Pending"
	}

	return appliedAt
}
