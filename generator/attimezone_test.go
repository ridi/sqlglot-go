package generator_test

// Round-trip checks for the three generator additions that close the TYPE/CAST/`::`/AT TIME
// ZONE parity gap: attimezone_sql (generator.py:3995-3998, plus mysql's unsupported override
// at generators/mysql.py:796-798), and pseudotype_sql/objectidentifier_sql (generator.py:
// 2316-2320). Wants confirmed against the pinned oracle:
//
//	PYTHONPATH=.reference/sqlglot-v30.12.0 python3 -c \
//	  "import sqlglot; print(sqlglot.transpile(\"SELECT foo AT TIME ZONE 'UTC'\", read='mysql', write='mysql')[0])"
//	SELECT foo

import "testing"

func TestAtTimeZoneSQL(t *testing.T) {
	cases := []struct{ dialect, sql, want string }{
		{"", "x AT TIME ZONE 'UTC'", "x AT TIME ZONE 'UTC'"},
		{"", "CURRENT_DATE AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Tokyo'", "CURRENT_DATE AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Tokyo'"},
		{"postgres", "x AT TIME ZONE 'UTC'", "x AT TIME ZONE 'UTC'"},
		// MySQL has no AT TIME ZONE syntax: the zone is dropped entirely (and the query is
		// flagged unsupported, at the default WARN level so this doesn't error).
		{"mysql", "SELECT foo AT TIME ZONE 'UTC'", "SELECT foo"},
	}
	for _, tc := range cases {
		if got := roundTrip(t, tc.dialect, tc.sql); got != tc.want {
			t.Errorf("%s %q ->\n  got  %q\n  want %q", tc.dialect, tc.sql, got, tc.want)
		}
	}
}

func TestPseudoTypeAndObjectIdentifierSQL(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"x::cstring", "CAST(x AS CSTRING)"},
		{"x::oid", "CAST(x AS OID)"},
		{"x::regclass", "CAST(x AS REGCLASS)"},
		{"x::regtype", "CAST(x AS REGTYPE)"},
	}
	for _, tc := range cases {
		if got := roundTrip(t, "postgres", tc.sql); got != tc.want {
			t.Errorf("postgres %q ->\n  got  %q\n  want %q", tc.sql, got, tc.want)
		}
	}
}
