package mig

import (
	"strings"
	"testing"
)

func TestSemicolons(t *testing.T) {

	type testData struct {
		line   string
		result bool
	}

	tests := []testData{
		{
			line:   "END;",
			result: true,
		},
		{
			line:   "END; -- comment",
			result: true,
		},
		{
			line:   "END   ; -- comment",
			result: true,
		},
		{
			line:   "END -- comment",
			result: false,
		},
		{
			line:   "END -- comment ;",
			result: false,
		},
		{
			line:   "END \" ; \" -- comment",
			result: false,
		},
	}

	for _, test := range tests {
		r := endsWithSemicolon(test.line)
		if r != test.result {
			t.Errorf("incorrect semicolon. got %v, want %v", r, test.result)
		}
	}
}

func TestSplitStatements(t *testing.T) {

	type testData struct {
		sql       string
		direction bool
		count     int
	}

	tests := []testData{
		{
			sql:       functxt,
			direction: true,
			count:     2,
		},
		{
			sql:       functxt,
			direction: false,
			count:     2,
		},
		{
			sql:       multitxt,
			direction: true,
			count:     2,
		},
		{
			sql:       multitxt,
			direction: false,
			count:     2,
		},
	}

	for _, test := range tests {
		stmts, err := splitSQLStatements(strings.NewReader(test.sql), test.direction)
		if err != nil {
			t.Error(err)
		}
		if len(stmts) != test.count {
			t.Errorf("incorrect number of stmts. got %v, want %v", len(stmts), test.count)
		}
	}
}

var functxt = `-- +mig Up
CREATE TABLE IF NOT EXISTS histories (
  id                BIGSERIAL  PRIMARY KEY,
  current_value     varchar(2000) NOT NULL,
  created_at      timestamp with time zone  NOT NULL
);

-- +mig StatementBegin
CREATE OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
BEGIN
  FOR create_query IN SELECT
      'CREATE TABLE IF NOT EXISTS histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( created_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND created_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( histories );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
  END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;
-- +mig StatementEnd

-- +mig Down
drop function histories_partition_creation(DATE, DATE);
drop TABLE histories;
`

// test multiple up/down transitions in a single script
var multitxt = `-- +mig Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +mig Down
DROP TABLE post;

-- +mig Up
CREATE TABLE fancier_post (
    id int NOT NULL,
    title text,
    body text,
    created_on timestamp without time zone,
    PRIMARY KEY(id)
);

-- +mig Down
DROP TABLE fancier_post;
`
