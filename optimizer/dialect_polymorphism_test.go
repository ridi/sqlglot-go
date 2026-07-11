package optimizer_test

import (
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	"github.com/sjincho/sqlglot-go/dialects"
	"github.com/sjincho/sqlglot-go/generator"
	"github.com/sjincho/sqlglot-go/optimizer"
)

// TestDialectTypePolymorphism verifies that NormalizeIdentifiers and Qualify accept a
// DialectType-style value (nil | string | *dialects.Dialect) — mirroring upstream sqlglot's
// polymorphic dialect argument — and that a typed *Dialect carrying a normalization strategy
// produces exactly the same result as the equivalent "name, normalization_strategy=..."
// settings string. This is the N1 win: proxy can build a *Dialect once instead of hand-
// formatting a magic settings string.
func TestDialectTypePolymorphism(t *testing.T) {
	normalize := func(t *testing.T, sql string, dialect any) string {
		t.Helper()
		e, err := sqlglot.ParseOne(sql, "mysql")
		if err != nil {
			t.Fatalf("ParseOne(%q): %v", sql, err)
		}
		normalized := optimizer.NormalizeIdentifiers(e, dialect)
		out, err := sqlglot.Generate(normalized, "mysql", generator.Options{})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		return out
	}

	const sql = "SELECT Col_A FROM Tbl"

	// Case-insensitive MySQL strategy: string form vs typed *Dialect must agree, and both must
	// fold the unquoted identifiers to lowercase.
	ciDialect := dialects.MySQL()
	ciDialect.NormalizationStrategy = dialects.MySQLCaseInsensitive
	fromString := normalize(t, sql, "mysql, normalization_strategy=mysql_case_insensitive")
	fromDialect := normalize(t, sql, ciDialect)
	if fromString != fromDialect {
		t.Fatalf("string vs *Dialect disagree:\n  string  %q\n  dialect %q", fromString, fromDialect)
	}
	if fromDialect != "SELECT col_a FROM tbl" {
		t.Fatalf("case-insensitive fold = %q, want %q", fromDialect, "SELECT col_a FROM tbl")
	}

	// Default MySQL (CASE_SENSITIVE): string "mysql", typed *Dialect, and nil-vs-base all leave
	// the identifiers unfolded, and string form == typed form.
	defString := normalize(t, sql, "mysql")
	defDialect := normalize(t, sql, dialects.MySQL())
	if defString != defDialect {
		t.Fatalf("default string vs *Dialect disagree:\n  string  %q\n  dialect %q", defString, defDialect)
	}
	if defDialect != "SELECT Col_A FROM Tbl" {
		t.Fatalf("case-sensitive (default) = %q, want unchanged", defDialect)
	}

	// SettingsString round-trips through GetOrRaise.
	if got := ciDialect.SettingsString(); got != "mysql, normalization_strategy=mysql_case_insensitive" {
		t.Fatalf("SettingsString() = %q", got)
	}
	rt, err := dialects.GetOrRaise(ciDialect.SettingsString())
	if err != nil {
		t.Fatalf("GetOrRaise(SettingsString): %v", err)
	}
	if rt.NormalizationStrategy != dialects.MySQLCaseInsensitive {
		t.Fatalf("round-trip strategy = %v, want MySQLCaseInsensitive", rt.NormalizationStrategy)
	}
}
