package dialects

import "testing"

// TestFoldIdentifierName covers the string-level fold (the N3 route-(b) API): applying a
// dialect's normalization strategy to a bare name, with the relation-vs-column role supplied
// by the isTable parameter instead of an AST parent.
func TestFoldIdentifierName(t *testing.T) {
	pg := Postgres() // Lowercase
	mysqlDefault := MySQL()
	mysqlCI := MySQL()
	mysqlCI.NormalizationStrategy = MySQLCaseInsensitive
	mysqlLCTN0 := MySQL()
	mysqlLCTN0.NormalizationStrategy = MySQLCaseSensitiveTableNames

	cases := []struct {
		name    string
		d       *Dialect
		in      string
		isTable bool
		want    string
	}{
		// Postgres folds unquoted names to lower, role-independent.
		{"pg column", pg, "Foo", false, "foo"},
		{"pg table", pg, "Foo", true, "foo"},
		// MySQL default (CASE_SENSITIVE) never folds.
		{"mysql default column", mysqlDefault, "Foo", false, "Foo"},
		{"mysql default table", mysqlDefault, "Foo", true, "Foo"},
		// lctn=1/2: everything folds (MySQL's accent-preserving .tolower).
		{"mysql ci table", mysqlCI, "Users", true, "users"},
		{"mysql ci column", mysqlCI, "RRN", false, "rrn"},
		{"mysql ci accent", mysqlCI, "CAFÉ", false, "café"},
		// lctn=0 (role-aware): relation names case-sensitive, column names fold.
		{"mysql lctn0 table preserved", mysqlLCTN0, "Users", true, "Users"},
		{"mysql lctn0 column folded", mysqlLCTN0, "RRN", false, "rrn"},
		{"mysql lctn0 accent column", mysqlLCTN0, "NIÑO", false, "niño"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.FoldIdentifierName(tc.in, tc.isTable); got != tc.want {
				t.Fatalf("FoldIdentifierName(%q, isTable=%v) = %q, want %q", tc.in, tc.isTable, got, tc.want)
			}
		})
	}
}
