package sqlglot_test

import (
	"testing"

	sqlglot "github.com/ridi-oss/sqlglot-go"
	exp "github.com/ridi-oss/sqlglot-go/expressions"
	"github.com/ridi-oss/sqlglot-go/generator"
)

// Bare niladic system-value functions complete the port of NO_PAREN_FUNCTIONS
// (parser.py:431-438 + parsers/postgres.py:145-152, parsers/mysql.py:55-59). Before this, only
// CURRENT_DATE/CURRENT_TIME were wired; every other bare form (current_timestamp, current_user,
// session_user, current_catalog, current_schema, localtime, localtimestamp) parsed to a plain
// Column, which an AST consumer that keys on "is a function named X" could not distinguish from a
// user column. The set is dialect-specific and matches both pinned upstream and the real engines:
// verified against PostgreSQL 17.6 and MySQL 8.0.33.

func niladicProjection(t *testing.T, sql, dialect string) exp.Expression {
	t.Helper()
	e, err := sqlglot.ParseOne(sql, dialect)
	if err != nil {
		t.Fatalf("%s [%s]: parse: %v", sql, dialect, err)
	}
	projections := e.(*exp.Node).Expressions()
	if len(projections) != 1 {
		t.Fatalf("%s [%s]: want 1 projection, got %d:\n%s", sql, dialect, len(projections), e.ToS())
	}
	return projections[0]
}

func TestNiladicFunctions(t *testing.T) {
	// kind is the Kind the bare keyword must parse to; "" means it stays a Column (reserved word
	// not wired as a niladic in that dialect, matching upstream + the real server). sql is the
	// round-trip output the port must produce for that dialect.
	cases := []struct {
		name   string
		pgKind exp.Kind
		pgSQL  string
		myKind exp.Kind
		mySQL  string
	}{
		{"current_date", exp.KindCurrentDate, "SELECT CURRENT_DATE", exp.KindCurrentDate, "SELECT CURRENT_DATE"},
		{"current_time", exp.KindCurrentTime, "SELECT CURRENT_TIME()", exp.KindCurrentTime, "SELECT CURRENT_TIME()"},
		{"current_timestamp", exp.KindCurrentTimestamp, "SELECT CURRENT_TIMESTAMP", exp.KindCurrentTimestamp, "SELECT CURRENT_TIMESTAMP()"},
		{"current_user", exp.KindCurrentUser, "SELECT CURRENT_USER", exp.KindCurrentUser, "SELECT CURRENT_USER()"},
		{"session_user", exp.KindSessionUser, "SELECT SESSION_USER", exp.KindColumn, "SELECT session_user"},
		{"current_catalog", exp.KindCurrentCatalog, "SELECT CURRENT_CATALOG", exp.KindColumn, "SELECT current_catalog"},
		{"current_schema", exp.KindCurrentSchema, "SELECT CURRENT_SCHEMA", exp.KindColumn, "SELECT current_schema"},
		{"localtime", exp.KindLocaltime, "SELECT LOCALTIME", exp.KindLocaltime, "SELECT LOCALTIME"},
		{"localtimestamp", exp.KindLocaltimestamp, "SELECT LOCALTIMESTAMP", exp.KindLocaltimestamp, "SELECT LOCALTIMESTAMP"},
		// current_role has no CURRENT_ROLE keyword in either tokenizer ("current_role" lexes as
		// VAR), so it stays a Column in both — matching upstream and the real servers.
		{"current_role", exp.KindColumn, "SELECT current_role", exp.KindColumn, "SELECT current_role"},
	}

	for _, tc := range cases {
		for _, d := range []struct {
			dialect string
			kind    exp.Kind
			sql     string
		}{
			{"postgres", tc.pgKind, tc.pgSQL},
			{"mysql", tc.myKind, tc.mySQL},
		} {
			in := "SELECT " + tc.name
			proj := niladicProjection(t, in, d.dialect)
			if proj.Kind() != d.kind {
				t.Errorf("%q [%s]: Kind = %s, want %s\n%s",
					in, d.dialect, exp.ClassName(proj.Kind()), exp.ClassName(d.kind), proj.ToS())
			}
			out, err := sqlglot.Generate(niladicRoot(t, in, d.dialect), d.dialect, generator.Options{})
			if err != nil {
				t.Fatalf("%q [%s]: generate: %v", in, d.dialect, err)
			}
			if out != d.sql {
				t.Errorf("%q [%s]: round-trip = %q, want %q", in, d.dialect, out, d.sql)
			}
		}
	}
}

func niladicRoot(t *testing.T, sql, dialect string) exp.Expression {
	t.Helper()
	e, err := sqlglot.ParseOne(sql, dialect)
	if err != nil {
		t.Fatalf("%s [%s]: parse: %v", sql, dialect, err)
	}
	return e
}

// TestNiladicDialectRoundTrip locks the per-dialect round-trip against the pinned-upstream oracle
// across every dialect this port builds these nodes for. It guards two review findings: the default
// ("base") dialect resolving LOCALTIME/CURRENT_CATALOG/SESSION_USER (parsers/base.py, BaseParser is
// the default parser_class), and the Presto-family rendering CurrentTimestamp/CurrentUser bare
// (generators/presto.py) — a previously-regressed same-dialect round-trip. Each want string was
// captured from pinned sqlglot v30.12.0. An empty want means "not asserted here" (e.g. presto
// LOCALTIME niladic completeness is deferred, out of the base+MySQL+Postgres scope).
func TestNiladicDialectRoundTrip(t *testing.T) {
	// columns: base, mysql, postgres, presto, trino, athena, hive
	dialects := []string{"", "mysql", "postgres", "presto", "trino", "athena", "hive"}
	cases := []struct {
		name string
		want [7]string
	}{
		{"current_timestamp", [7]string{"SELECT CURRENT_TIMESTAMP()", "SELECT CURRENT_TIMESTAMP()", "SELECT CURRENT_TIMESTAMP", "SELECT CURRENT_TIMESTAMP", "SELECT CURRENT_TIMESTAMP", "SELECT CURRENT_TIMESTAMP", "SELECT CURRENT_TIMESTAMP()"}},
		{"current_user", [7]string{"SELECT CURRENT_USER()", "SELECT CURRENT_USER()", "SELECT CURRENT_USER", "SELECT CURRENT_USER", "SELECT CURRENT_USER", "SELECT CURRENT_USER", "SELECT CURRENT_USER()"}},
		{"current_catalog", [7]string{"SELECT CURRENT_CATALOG", "SELECT current_catalog", "SELECT CURRENT_CATALOG", "SELECT current_catalog", "SELECT CURRENT_CATALOG", "SELECT CURRENT_CATALOG", "SELECT current_catalog"}},
		{"session_user", [7]string{"SELECT SESSION_USER", "SELECT session_user", "SELECT SESSION_USER", "SELECT session_user", "SELECT session_user", "SELECT session_user", "SELECT session_user"}},
		{"localtime", [7]string{"SELECT LOCALTIME", "SELECT LOCALTIME", "SELECT LOCALTIME", "", "", "", ""}},
	}
	for _, tc := range cases {
		in := "SELECT " + tc.name
		for i, dialect := range dialects {
			want := tc.want[i]
			if want == "" {
				continue
			}
			got, err := sqlglot.Generate(niladicRoot(t, in, dialect), dialect, generator.Options{})
			if err != nil {
				t.Errorf("%q [%s]: generate: %v", in, dialect, err)
				continue
			}
			if got != want {
				t.Errorf("%q [%s]: round-trip = %q, want %q", in, dialect, got, want)
			}
		}
	}
}

// current_schema is the one Postgres niladic that is also a non-reserved keyword usable as an
// unquoted type name: bare it is the function, but `current_schema 'x'` is a user-type typed literal
// (real PG errors with `type "current_schema" does not exist` — the same class as `type 'x'`),
// whereas every other niladic yields a `syntax error` in that position. So the niladic resolution
// must defer to the typed-literal path only for CURRENT_SCHEMA + a following string.
func TestNiladicCurrentSchemaTypedLiteralDualNature(t *testing.T) {
	// bare -> function
	if k := niladicProjection(t, "SELECT current_schema", "postgres").Kind(); k != exp.KindCurrentSchema {
		t.Fatalf("bare current_schema Kind = %s, want CurrentSchema", exp.ClassName(k))
	}
	// followed by a string -> typed literal (Cast with one DataType)
	e, err := sqlglot.ParseOne("SELECT current_schema 'x'", "postgres")
	if err != nil {
		t.Fatalf("`current_schema 'x'`: parse: %v", err)
	}
	if proj := e.(*exp.Node).Expressions()[0]; proj.Kind() != exp.KindCast {
		t.Fatalf("`current_schema 'x'`: Kind = %s, want Cast\n%s", exp.ClassName(proj.Kind()), e.ToS())
	}
	if n := len(e.FindAll(exp.KindDataType)); n != 1 {
		t.Fatalf("`current_schema 'x'`: FindAll(DataType) = %d, want 1\n%s", n, e.ToS())
	}
	// the reserved niladics remain fail-closed in typed-literal position (real PG: syntax error).
	for _, sql := range []string{
		"SELECT current_user 'x'",
		"SELECT session_user 'x'",
		"SELECT current_catalog 'x'",
	} {
		if _, err := sqlglot.ParseOne(sql, "postgres"); err == nil {
			t.Errorf("%q: parsed without error, want fail-closed (reserved value-function, not a type name)", sql)
		}
	}
}
