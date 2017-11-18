package mig

import (
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"time"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	// postgres driver
	_ "github.com/lib/pq"
)

// Create a templated migration file in dir
func Create(name, dir string) (string, error) {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%v_%v.sql", timestamp, name)

	fpath := filepath.Join(dir, filename)
	tmpl := migrationTemplate

	path, err := writeTemplateToFile(fpath, tmpl, timestamp)
	return path, err
}

// Down rolls back the version by one
func Down(driver, conn, dir string) (name string, err error) {
	db, err := sql.Open(driver, conn)
	if err != nil {
		return "", err
	}

	err = setDialect(driver)
	if err != nil {
		return "", err
	}

	currentVersion, err := getVersion(db)
	if err != nil {
		return "", err
	}

	migrations, err := collectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return "", err
	}

	current, err := migrations.current(currentVersion)
	if err != nil {
		return "", errNoMigration{}
	}

	return current.down(db)
}

// DownAll rolls back all migrations.
// Logs success messages to global writer variable Log.
func DownAll(driver, conn, dir string) (int, error) {
	count := 0

	db, err := sql.Open(driver, conn)
	if err != nil {
		return count, err
	}

	err = setDialect(driver)
	if err != nil {
		return count, err
	}

	migrations, err := collectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return count, err
	}

	for {
		currentVersion, err := getVersion(db)
		if err != nil {
			return count, err
		}

		current, err := migrations.current(currentVersion)
		// no migrations left to run
		if err != nil {
			return count, nil
		}

		name, err := current.down(db)
		if err != nil {
			return count, err
		}

		Log.Write([]byte(fmt.Sprintf("Success   %v\n", name)))
		count++
	}
}

// Up migrates to the highest version available
func Up(driver, conn, dir string) (int, error) {
	count := 0

	db, err := sql.Open(driver, conn)
	if err != nil {
		return count, err
	}

	err = setDialect(driver)
	if err != nil {
		return count, err
	}

	migrations, err := collectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return count, err
	}

	for {
		currentVersion, err := getVersion(db)
		if err != nil {
			return count, err
		}

		next, err := migrations.next(currentVersion)
		// no migrations left to run
		if err != nil {
			return count, nil
		}

		name, err := next.up(db)
		if err != nil {
			return count, err
		}

		Log.Write([]byte(fmt.Sprintf("Success   %v\n", name)))
		count++
	}
}

// UpOne migrates one version
func UpOne(driver, conn, dir string) (name string, err error) {
	db, err := sql.Open(driver, conn)
	if err != nil {
		return "", err
	}

	err = setDialect(driver)
	if err != nil {
		return "", err
	}

	currentVersion, err := getVersion(db)
	if err != nil {
		return "", err
	}

	migrations, err := collectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return "", err
	}

	next, err := migrations.next(currentVersion)
	if err != nil {
		return "", errNoMigration{}
	}

	return next.up(db)
}

// Redo re-runs the latest migration.
func Redo(driver, conn, dir string) (string, error) {
	db, err := sql.Open(driver, conn)
	if err != nil {
		return "", err
	}

	err = setDialect(driver)
	if err != nil {
		return "", err
	}

	currentVersion, err := getVersion(db)
	if err != nil {
		return "", err
	}

	migrations, err := collectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return "", err
	}

	current, err := migrations.current(currentVersion)
	if err != nil {
		return "", errNoMigration{}
	}

	if _, err := current.down(db); err != nil {
		return "", err
	}

	return current.up(db)
}

type migrationStatus struct {
	Applied string
	Name    string
}

// status is a slice of migrationStatus
type status []migrationStatus

// Status returns the status of each migration
func Status(driver, conn, dir string) (status, error) {
	s := status{}

	db, err := sql.Open(driver, conn)
	if err != nil {
		return s, err
	}

	err = setDialect(driver)
	if err != nil {
		return s, err
	}

	migrations, err := collectMigrations(dir, 0, math.MaxInt64)
	if err != nil {
		return s, err
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := getVersion(db); err != nil {
		return s, err
	}

	for _, migration := range migrations {
		s = append(s, migrationStatus{
			Applied: getMigrationStatus(db, migration.version),
			Name:    filepath.Base(migration.source),
		})
	}

	return s, nil
}

// Version returns the current migration version
func Version(driver, conn string) (int64, error) {
	db, err := sql.Open(driver, conn)
	if err != nil {
		return 0, err
	}

	err = setDialect(driver)
	if err != nil {
		return 0, err
	}

	return getVersion(db)
}
