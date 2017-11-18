package mig

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type migrationRecord struct {
	versionId int64
	tstamp    time.Time
	isApplied bool // was this a result of up() or down()
}

type migration struct {
	version  int64
	next     int64  // next version, or -1 if none
	previous int64  // previous version, -1 if none
	source   string // path to .sql script
}

const sqlCmdPrefix = "-- +mig "

var migrationTemplate = template.Must(template.New("mig.sql-migration").Parse(`-- +mig Up

-- +mig Down

`))

func (m *migration) String() string {
	return fmt.Sprintf(m.source)
}

func (m *migration) up(db *sql.DB) (string, error) {
	return m.run(db, true)
}

func (m *migration) down(db *sql.DB) (string, error) {
	return m.run(db, false)
}

func (m *migration) run(db *sql.DB, direction bool) (name string, err error) {
	if err := runMigration(db, m.source, m.version, direction); err != nil {
		return "", err
	}

	return filepath.Base(m.source), nil
}

// look for migration scripts with names in the form:
//  XXX_descriptivename.sql
// where XXX specifies the version number
func numericComponent(name string) (int64, error) {
	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no separator found")
	}

	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("migration IDs must be greater than zero")
	}

	return n, e
}

// Update the version table for the given migration,
// and finalize the transaction.
func finalizeMigration(tx *sql.Tx, direction bool, v int64) error {
	stmt := getDialect().insertVersionSQL()
	if _, err := tx.Exec(stmt, v, direction); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line string) bool {
	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}

	return strings.HasSuffix(prev, ";")
}

// Split the given sql script into individual statements.
//
// The base case is to simply split on semicolons, as these
// naturally terminate a statement.
//
// However, more complex cases like pl/pgsql can have semicolons
// within a statement. For these cases, we provide the explicit annotations
// 'StatementBegin' and 'StatementEnd' to allow the script to
// tell us to ignore semicolons.
func splitSQLStatements(r io.Reader, direction bool) ([]string, error) {
	var err error
	var stmts []string
	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)

	// track the count of each section
	// so we can diagnose scripts with no annotations
	upSections := 0
	downSections := 0

	statementEnded := false
	ignoreSemicolons := false
	directionIsActive := false

	for scanner.Scan() {

		line := scanner.Text()

		// handle any mig-specific commands
		if strings.HasPrefix(line, sqlCmdPrefix) {
			cmd := strings.TrimSpace(line[len(sqlCmdPrefix):])
			switch cmd {
			case "Up":
				directionIsActive = (direction == true)
				upSections++
				break

			case "Down":
				directionIsActive = (direction == false)
				downSections++
				break

			case "StatementBegin":
				if directionIsActive {
					ignoreSemicolons = true
				}
				break

			case "StatementEnd":
				if directionIsActive {
					statementEnded = (ignoreSemicolons == true)
					ignoreSemicolons = false
				}
				break
			}
		}

		if !directionIsActive {
			continue
		}

		if _, err := buf.WriteString(line + "\n"); err != nil {
			panic(fmt.Sprintf("io err: %v", err))
		}

		// Wrap up the two supported cases: 1) basic with semicolon; 2) psql statement
		// Lines that end with semicolon that are in a statement block
		// do not conclude statement.
		if (!ignoreSemicolons && endsWithSemicolon(line)) || statementEnded {
			statementEnded = false
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return stmts, fmt.Errorf("error reading migration: %v", err)
	}

	// diagnose likely migration script errors
	if ignoreSemicolons {
		return stmts, errors.New("saw '-- +mig StatementBegin' with no matching '-- +mig StatementEnd'")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		return stmts, fmt.Errorf("unexpected unfinished SQL query: %s. Missing a semicolon?", bufferRemaining)
	}

	if upSections == 0 && downSections == 0 {
		return stmts, fmt.Errorf(`no up/down annotations found, so no statements were executed`)
	}

	return stmts, err
}

// runMigration runs a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +mig" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runMigration(db *sql.DB, scriptFile string, v int64, direction bool) error {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	f, err := os.Open(scriptFile)
	if err != nil {
		return fmt.Errorf("cannot open migration file %s: %v", scriptFile, err)
	}

	stmts, err := splitSQLStatements(f, direction)
	if err != nil {
		return fmt.Errorf("error splitting migration %s: %v", filepath.Base(scriptFile), err)
	}

	// find each statement, checking annotations for up/down direction
	// and execute each of them in the current transaction.
	// Commits the transaction if successfully applied each statement and
	// records the version into the version table or returns an error and
	// rolls back the transaction.
	for _, query := range stmts {
		if _, err = tx.Exec(query); err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing migration %s: %v", filepath.Base(scriptFile), err)
		}
	}

	if err = finalizeMigration(tx, direction, v); err != nil {
		return fmt.Errorf("error committing migration %s: %v", filepath.Base(scriptFile), err)
	}

	return nil
}
