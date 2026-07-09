package generator_test

import "testing"

// TestLockSQL ports the row-locking-read generator coverage from
// tests/dialects/test_mysql.py:1080-1082 (LOCK IN SHARE MODE canonicalizes to FOR SHARE) and
// tests/dialects/test_postgres.py:1772-1775 (FOR SHARE/FOR UPDATE/FOR NO KEY UPDATE/FOR KEY SHARE
// round-trip as identities with an OF clause).
func TestLockSQL(t *testing.T) {
	if got, want := roundTrip(t, "mysql", "SELECT * FROM t LOCK IN SHARE MODE"), "SELECT * FROM t FOR SHARE"; got != want {
		t.Fatalf("LOCK IN SHARE MODE -> %q, want %q", got, want)
	}

	for _, keyType := range []string{"FOR SHARE", "FOR UPDATE", "FOR NO KEY UPDATE", "FOR KEY SHARE"} {
		sql := "SELECT 1 FROM foo AS x " + keyType + " OF x"
		if got := roundTrip(t, "postgres", sql); got != sql {
			t.Fatalf("%q ->\n  got  %q\n  want %q", sql, got, sql)
		}
	}
}
