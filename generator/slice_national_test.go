package generator_test

// Round-trip checks for sliceSQL/nationalSQL (generator/slice_national.go), which port
// slice_sql (generator.py:5667-5673) and national_sql (generator.py:2010-2012). Cases are
// drawn from testdata/parity_gaps.txt (5-9, 35) and the duckdb DASH+COLON edge case from
// _parse_slice (tests/dialects/test_duckdb.py:23), confirmed against the pinned oracle:
//
//	PYTHONPATH=.reference/sqlglot-v30.12.0 python3 -c \
//	  "import sqlglot; print(sqlglot.parse_one(\"x[-4:-1]\").sql())"

import "testing"

func TestSliceSQL(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"x[1:2]", "x[1:2]"},
		{"x[1:]", "x[1:]"},
		{"x[:2]", "x[:2]"},
		{"x[:]", "x[:]"},
		{"x[-4:-1]", "x[-4:-1]"},
		{"x[1:2:3]", "x[1:2:3]"},
		{"x[1:2].b", "x[1:2].b"},
	}
	for _, tc := range cases {
		if got := roundTrip(t, "", tc.sql); got != tc.want {
			t.Errorf("%q ->\n  got  %q\n  want %q", tc.sql, got, tc.want)
		}
	}
}

func TestNationalSQL(t *testing.T) {
	cases := []struct{ dialect, sql, want string }{
		{"", "N'abc'", "N'abc'"},
		{"", "n'abc'", "N'abc'"},
		{"", "SELECT N'abc'", "SELECT N'abc'"},
		{"mysql", "SELECT N'abc'", "SELECT N'abc'"},
		{"postgres", "SELECT N'abc'", "SELECT N'abc'"},
	}
	for _, tc := range cases {
		if got := roundTrip(t, tc.dialect, tc.sql); got != tc.want {
			t.Errorf("%s %q ->\n  got  %q\n  want %q", tc.dialect, tc.sql, got, tc.want)
		}
	}
}
