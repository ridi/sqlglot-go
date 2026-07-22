package sqlglot_test

import (
	"strings"
	"testing"

	sqlglot "github.com/ridi-oss/sqlglot-go"
	exp "github.com/ridi-oss/sqlglot-go/expressions"
	"github.com/ridi-oss/sqlglot-go/generator"
)

// Postgres `START TRANSACTION [<modes>]` and MySQL `START TRANSACTION [… WITH CONSISTENT SNAPSHOT]`
// parse to exp.Transaction. Pinned upstream maps START->BEGIN only for mysql/presto/oracle (not
// postgres) and does not consume the ONLY token or the WITH CONSISTENT SNAPSHOT phrase, so it
// errors on these valid statements. See DEVIATIONS "Grammar extensions beyond upstream".
//
// Each (sql, dialect) pair below is one the corresponding engine actually accepts: verified against
// PostgreSQL 17.6 and MySQL 8.0.33. Postgres accepts ISOLATION LEVEL / DEFERRABLE but NOT WITH
// CONSISTENT SNAPSHOT; MySQL accepts WITH CONSISTENT SNAPSHOT but not the standalone ISOLATION LEVEL
// spelling. Modes and round-trip SQL are asserted, not just the root Kind.
func TestStartTransaction(t *testing.T) {
	cases := []struct {
		sql     string
		dialect string
		modes   []string
	}{
		{"START TRANSACTION", "postgres", nil},
		{"START TRANSACTION", "mysql", nil},
		{"START TRANSACTION READ ONLY", "postgres", []string{"READ ONLY"}},
		{"START TRANSACTION READ ONLY", "mysql", []string{"READ ONLY"}},
		{"START TRANSACTION READ WRITE", "postgres", []string{"READ WRITE"}},
		{"START TRANSACTION READ WRITE", "mysql", []string{"READ WRITE"}},
		{"START TRANSACTION ISOLATION LEVEL SERIALIZABLE", "postgres", []string{"ISOLATION LEVEL SERIALIZABLE"}},
		{"START TRANSACTION ISOLATION LEVEL REPEATABLE READ", "postgres", []string{"ISOLATION LEVEL REPEATABLE READ"}},
		{"START TRANSACTION READ ONLY, DEFERRABLE", "postgres", []string{"READ ONLY", "DEFERRABLE"}},
		{"START TRANSACTION WITH CONSISTENT SNAPSHOT", "mysql", []string{"WITH CONSISTENT SNAPSHOT"}},
	}
	for _, tc := range cases {
		e, err := sqlglot.ParseOne(tc.sql, tc.dialect)
		if err != nil {
			t.Errorf("%q [%s]: parse: %v", tc.sql, tc.dialect, err)
			continue
		}
		if e.Kind() != exp.KindTransaction {
			t.Errorf("%q [%s]: Kind = %s, want Transaction\n%s", tc.sql, tc.dialect, exp.ClassName(e.Kind()), e.ToS())
			continue
		}
		if got := transactionModes(e); !equalStrings(got, tc.modes) {
			t.Errorf("%q [%s]: modes = %#v, want %#v", tc.sql, tc.dialect, got, tc.modes)
		}
		// The whole statement round-trips: START TRANSACTION normalizes to BEGIN (both are valid
		// transaction starters), carrying the modes through as a comma-separated list.
		want := "BEGIN"
		if len(tc.modes) > 0 {
			want += " " + strings.Join(tc.modes, ", ")
		}
		if got, _ := sqlglot.Generate(e, tc.dialect, generator.Options{}); got != want {
			t.Errorf("%q [%s]: round-trip = %q, want %q", tc.sql, tc.dialect, got, want)
		}
	}
}

// WITH CONSISTENT SNAPSHOT is MySQL-only: real MySQL 8.0.33 accepts it, real PostgreSQL 17.6 rejects
// it (and pinned upstream errors on it in both), so the parser gates it to MySQL.
func TestStartTransactionConsistentSnapshotGating(t *testing.T) {
	if _, err := sqlglot.ParseOne("START TRANSACTION WITH CONSISTENT SNAPSHOT", "mysql"); err != nil {
		t.Errorf("mysql WITH CONSISTENT SNAPSHOT: parse: %v", err)
	}
	for _, dialect := range []string{"postgres", "base"} {
		if _, err := sqlglot.ParseOne("START TRANSACTION WITH CONSISTENT SNAPSHOT", dialect); err == nil {
			t.Errorf("%s WITH CONSISTENT SNAPSHOT: parsed without error, want reject (not valid there)", dialect)
		}
	}
}

func transactionModes(e exp.Expression) []string {
	raw, ok := e.Arg("modes").([]string)
	if !ok {
		return nil
	}
	return raw
}

// BEGIN / COMMIT / ROLLBACK still parse to their own nodes — the START->BEGIN keyword mapping must
// not disturb the sibling transaction statements.
func TestTransactionSiblingsUnaffected(t *testing.T) {
	cases := []struct {
		sql  string
		kind exp.Kind
	}{
		{"BEGIN", exp.KindTransaction},
		{"COMMIT", exp.KindCommit},
		{"ROLLBACK", exp.KindRollback},
	}
	for _, tc := range cases {
		for _, dialect := range []string{"postgres", "mysql"} {
			e, err := sqlglot.ParseOne(tc.sql, dialect)
			if err != nil {
				t.Errorf("%q [%s]: parse: %v", tc.sql, dialect, err)
				continue
			}
			if e.Kind() != tc.kind {
				t.Errorf("%q [%s]: Kind = %s, want %s", tc.sql, dialect, exp.ClassName(e.Kind()), exp.ClassName(tc.kind))
			}
		}
	}
}

// `start` remains usable as an ordinary identifier in Postgres despite the START->BEGIN keyword
// mapping (tokens.BEGIN is in the identifier/alias token sets, the same mechanism MySQL relies on).
func TestStartAsIdentifier(t *testing.T) {
	for _, sql := range []string{
		"SELECT start FROM t",
		"SELECT 1 AS start",
		"SELECT * FROM start",
		"SELECT start.id FROM foo AS start",
	} {
		for _, dialect := range []string{"postgres", "mysql"} {
			if _, err := sqlglot.ParseOne(sql, dialect); err != nil {
				t.Errorf("%q [%s]: parse: %v", sql, dialect, err)
			}
		}
	}
}
