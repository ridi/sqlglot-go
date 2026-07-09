package sqlglot_test

import (
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	"github.com/sjincho/sqlglot-go/generator"
)

// TestResidualParityFixes locks in the integrator residual-fix round: each case is
// oracle-verified against the pinned reference (sqlglot v30.12.0). The four cases that are
// also corpus rows (mysql PARTITION selection, mysql reserved-keyword quoting, postgres MERGE
// target unqualification, postgres date_add(current_date, ...)) are additionally guarded by
// TestCorpus + the raised minPass* floors; CURDATE and MERGE `UPDATE *` are NOT corpus rows,
// so this is their only regression guard.
func TestResidualParityFixes(t *testing.T) {
	cases := []struct {
		name    string
		dialect string
		sql     string
		want    string
	}{
		// dialect-funcs: CURDATE -> CurrentDate renders bare CURRENT_DATE (no parens); CURTIME
		// still keeps the parenthesized fallback (only currentdate_sql exists upstream).
		{"mysql_curdate", "mysql", "SELECT CURDATE()", "SELECT CURRENT_DATE"},
		{"mysql_curtime", "mysql", "SELECT CURTIME()", "SELECT CURRENT_TIME()"},
		// no-paren CURRENT_DATE keyword (NO_PAREN_FUNCTIONS): the corpus gap plus a couple of
		// base cases that must NOT regress (bare CURRENT_DATE, and current_date as a table name).
		{"pg_date_add_current_date", "postgres", "SELECT date_add(current_date, interval '7' day)", "SELECT CURRENT_DATE + INTERVAL '7 DAY'"},
		{"base_current_date", "", "SELECT CURRENT_DATE", "SELECT CURRENT_DATE"},
		{"base_current_date_at_tz", "", "SELECT CURRENT_DATE AT TIME ZONE 'UTC'", "SELECT CURRENT_DATE AT TIME ZONE 'UTC'"},
		{"base_current_date_table", "", "SELECT * FROM current_date", "SELECT * FROM current_date"},
		// from-dml: mysql FROM-clause PARTITION(...) selection (SUPPORTS_PARTITION_SELECTION);
		// base keeps it as a plain alias, matching upstream.
		{"mysql_partition_from", "mysql", "SELECT * FROM t1 PARTITION(p0)", "SELECT * FROM t1 PARTITION(p0)"},
		// generator: mysql reserved-keyword identifier quoting; base/postgres leave it unquoted.
		{"mysql_reserved_alias", "mysql", "SELECT 1 AS row", "SELECT 1 AS `row`"},
		{"base_reserved_alias", "", "SELECT 1 AS row", "SELECT 1 AS row"},
		{"pg_reserved_alias", "postgres", "SELECT 1 AS row", "SELECT 1 AS row"},
		// generator: postgres MERGE removes the target-table qualifier from WHEN update/insert
		// columns (merge_without_target_sql); base leaves them intact.
		{"pg_merge_unqualify", "postgres", "MERGE INTO x USING (SELECT id) AS y ON a = b WHEN MATCHED THEN UPDATE SET x.a = y.b WHEN NOT MATCHED THEN INSERT (a, b) VALUES (y.a, y.b)", "MERGE INTO x USING (SELECT id) AS y ON a = b WHEN MATCHED THEN UPDATE SET a = y.b WHEN NOT MATCHED THEN INSERT (a, b) VALUES (y.a, y.b)"},
		{"base_merge_keeps_qualifier", "", "MERGE INTO x USING (SELECT id) AS y ON a = b WHEN MATCHED THEN UPDATE SET x.a = y.b WHEN NOT MATCHED THEN INSERT (a, b) VALUES (y.a, y.b)", "MERGE INTO x USING (SELECT id) AS y ON a = b WHEN MATCHED THEN UPDATE SET x.a = y.b WHEN NOT MATCHED THEN INSERT (a, b) VALUES (y.a, y.b)"},
		// generator: MERGE `UPDATE *` renders bare (not `UPDATE SET *`) for every dialect.
		{"pg_merge_update_star", "postgres", "MERGE INTO x USING y ON a = b WHEN MATCHED THEN UPDATE *", "MERGE INTO x USING y ON a = b WHEN MATCHED THEN UPDATE *"},

		// --- second residual round (dialect-divergence review findings) ---
		// dialect-funcs: parenthesized CURRENT_DATE() builds a CurrentDate node too (it's a
		// base FUNCTION_BY_NAME entry), rendering bare CURRENT_DATE for every dialect.
		{"base_current_date_paren", "", "SELECT CURRENT_DATE()", "SELECT CURRENT_DATE"},
		{"mysql_current_date_paren", "mysql", "SELECT CURRENT_DATE()", "SELECT CURRENT_DATE"},
		{"pg_current_date_paren", "postgres", "SELECT CURRENT_DATE()", "SELECT CURRENT_DATE"},
		{"pg_date_add_current_date_paren", "postgres", "SELECT date_add(current_date(), interval '7' day)", "SELECT CURRENT_DATE + INTERVAL '7 DAY'"},
		{"base_current_date_zone", "", "SELECT CURRENT_DATE('UTC')", "SELECT CURRENT_DATE('UTC')"},
		// ddl: a bare CURRENT_DATE column-constraint default now builds CurrentDate (renders
		// bare); CURRENT_TIMESTAMP still keeps the parenthesized Anonymous form (no Kind yet).
		{"base_default_current_date", "", "CREATE TABLE t (c DATE DEFAULT CURRENT_DATE)", "CREATE TABLE t (c DATE DEFAULT CURRENT_DATE)"},
		{"mysql_default_current_date", "mysql", "CREATE TABLE t (c DATE DEFAULT CURRENT_DATE)", "CREATE TABLE t (c DATE DEFAULT CURRENT_DATE)"},
		{"mysql_default_current_timestamp", "mysql", "CREATE TABLE t (c DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)", "CREATE TABLE t (c DATETIME DEFAULT CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP())"},
		// dialect-funcs: base/postgres recognize the no-underscore day/week spellings +
		// LCASE/UCASE (base FUNCTION_BY_NAME), canonicalizing to DAY_OF_MONTH/LOWER; mysql keeps
		// its own spelling. Also guards the mysql.go dedup (DAY_OF_MONTH still -> DAYOFMONTH).
		{"base_dayofmonth", "", "SELECT DAYOFMONTH(x)", "SELECT DAY_OF_MONTH(x)"},
		{"pg_dayofmonth", "postgres", "SELECT DAYOFMONTH(x)", "SELECT DAY_OF_MONTH(x)"},
		{"mysql_dayofmonth", "mysql", "SELECT DAYOFMONTH(x)", "SELECT DAYOFMONTH(x)"},
		{"mysql_day_of_month_dedup", "mysql", "SELECT DAY_OF_MONTH(x)", "SELECT DAYOFMONTH(x)"},
		{"base_lcase", "", "SELECT LCASE(x)", "SELECT LOWER(x)"},
		// generator: SELECT ... INTO is inline only for postgres (SUPPORTS_SELECT_INTO); base/
		// mysql rewrite it to CTAS (UNLOGGED is dropped since base/mysql lack unlogged tables).
		{"base_select_into_ctas", "", "SELECT * INTO foo FROM bar", "CREATE TABLE foo AS SELECT * FROM bar"},
		{"mysql_select_into_ctas", "mysql", "SELECT * INTO foo FROM bar", "CREATE TABLE foo AS SELECT * FROM bar"},
		{"pg_select_into_inline", "postgres", "SELECT * INTO foo FROM bar", "SELECT * INTO foo FROM bar"},
		{"base_select_into_unlogged_ctas", "", "SELECT * INTO UNLOGGED foo FROM bar", "CREATE TABLE foo AS SELECT * FROM bar"},
		{"pg_select_into_unlogged_inline", "postgres", "SELECT * INTO UNLOGGED foo FROM bar", "SELECT * INTO UNLOGGED foo FROM bar"},
		// generator: postgres renders placeholders in pyformat (%(name)s / %s); base keeps :name.
		{"pg_placeholder_named", "postgres", "SELECT * FROM x LIMIT :my_limit", "SELECT * FROM x LIMIT %(my_limit)s"},
		{"pg_placeholder_select", "postgres", "SELECT :hello", "SELECT %(hello)s"},
		{"base_placeholder_named", "", "SELECT * FROM x LIMIT :my_limit", "SELECT * FROM x LIMIT :my_limit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := sqlglot.ParseOne(tc.sql, tc.dialect)
			if err != nil {
				t.Fatalf("ParseOne(%q, %q) error: %v", tc.sql, tc.dialect, err)
			}
			got, err := sqlglot.Generate(expr, tc.dialect, generator.Options{})
			if err != nil {
				t.Fatalf("Generate error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("round-trip mismatch\n  dialect: %q\n  sql:  %s\n  got:  %s\n  want: %s", tc.dialect, tc.sql, got, tc.want)
			}
		})
	}
}
