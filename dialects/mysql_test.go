package dialects_test

import (
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	"github.com/sjincho/sqlglot-go/dialects"
	"github.com/sjincho/sqlglot-go/generator"
	"github.com/sjincho/sqlglot-go/optimizer"
	"github.com/sjincho/sqlglot-go/schema"
	"github.com/sjincho/sqlglot-go/tokens"
)

func TestMySQLConfigAndTokenizer(t *testing.T) {
	d, err := dialects.GetOrRaise("mysql")
	if err != nil {
		t.Fatalf("GetOrRaise(mysql): %v", err)
	}
	if d.Name != "mysql" {
		t.Fatalf("Name = %q, want mysql", d.Name)
	}
	if d.NormalizationStrategy != dialects.CaseSensitive {
		t.Fatalf("NormalizationStrategy = %v, want CaseSensitive", d.NormalizationStrategy)
	}
	if d.DPipeIsStringConcat || d.SupportsUserDefinedTypes || !d.SafeDivision {
		t.Fatalf("dialect flags: DPipeIsStringConcat=%v SupportsUserDefinedTypes=%v SafeDivision=%v", d.DPipeIsStringConcat, d.SupportsUserDefinedTypes, d.SafeDivision)
	}
	if d.QuoteStart != "'" || d.QuoteEnd != "'" || d.IdentifierStart != "`" || d.IdentifierEnd != "`" {
		t.Fatalf("delimiters = quote %q/%q identifier %q/%q", d.QuoteStart, d.QuoteEnd, d.IdentifierStart, d.IdentifierEnd)
	}
	if !d.ValidIntervalUnits["SECOND_MICROSECOND"] || !d.ValidIntervalUnits["YEAR_MONTH"] {
		t.Fatalf("missing MySQL compound interval units")
	}

	toks, err := sqlglot.Tokenize("SELECT \"x\", `A`, b'1010', x'1F', 0b1010, 0x1F, @@foo # hi\nFROM t", "mysql")
	if err != nil {
		t.Fatalf("Tokenize(mysql): %v", err)
	}
	wantTypes := []tokens.TokenType{
		tokens.SELECT,
		tokens.STRING,
		tokens.COMMA,
		tokens.IDENTIFIER,
		tokens.COMMA,
		tokens.BIT_STRING,
		tokens.COMMA,
		tokens.HEX_STRING,
		tokens.COMMA,
		tokens.BIT_STRING,
		tokens.COMMA,
		tokens.HEX_STRING,
		tokens.COMMA,
		tokens.SESSION_PARAMETER,
		tokens.VAR,
		tokens.FROM,
		tokens.VAR,
	}
	if len(toks) != len(wantTypes) {
		t.Fatalf("token count = %d, want %d: %s", len(toks), len(wantTypes), tokens.ReprTokens(toks))
	}
	for i, want := range wantTypes {
		if toks[i].TokenType != want {
			t.Fatalf("token %d type = %s, want %s: %s", i, toks[i].TokenType, want, tokens.ReprTokens(toks))
		}
	}
	if toks[1].Text != "x" || toks[3].Text != "A" || toks[5].Text != "1010" || toks[7].Text != "1F" || toks[9].Text != "1010" || toks[11].Text != "1F" {
		t.Fatalf("unexpected token text: %s", tokens.ReprTokens(toks))
	}
	if len(toks[14].Comments) != 1 || toks[14].Comments[0] != " hi" {
		t.Fatalf("hash comment = %#v, want [\" hi\"]", toks[14].Comments)
	}
}

func TestMySQLIdentityRoundTrips(t *testing.T) {
	cases := []identityCase{
		{name: "backtick identifiers", dialect: "mysql", sql: "SELECT `Foo` FROM `Bar`"},
		{name: "double quoted string", dialect: "mysql", sql: `SELECT "x"`, want: "SELECT 'x'"},
		{name: "hash comment", dialect: "mysql", sql: "SELECT a # comment\nFROM t", want: "SELECT a /* comment */ FROM t"},
		{name: "insert default", dialect: "mysql", sql: "INSERT INTO t (i) VALUES (DEFAULT)"},
		{name: "replace command", dialect: "mysql", sql: "REPLACE INTO t (a) VALUES (1)"},
		{name: "update", dialect: "mysql", sql: "UPDATE items SET price = 0 WHERE id >= 5"},
		{name: "delete", dialect: "mysql", sql: "DELETE FROM t WHERE a <= 10"},
		{name: "cte", dialect: "mysql", sql: "WITH x AS (SELECT 1 AS a) SELECT a FROM x"},
		{name: "union", dialect: "mysql", sql: "SELECT a FROM x UNION SELECT b FROM y"},
		{name: "qualified backtick alias", dialect: "mysql", sql: "SELECT `a`.`b` AS `c` FROM `a`"},
		{name: "curdate function", deferredReason: "parser FUNCTIONS + generator TRANSFORM — slice 5b", category: "parser FUNCTIONS/generator TRANSFORM"},
		{name: "timestamp type remap", deferredReason: "generator TYPE_MAPPING for MySQL TIMESTAMP remap — slice 5b", category: "generator TYPE_MAPPING"},
		{name: "logical pipes", deferredReason: "CONJUNCTION/DISJUNCTION operator wiring plus KindXor — slice 5b", category: "parser operator"},
		{name: "show tables", deferredReason: "SHOW command parser/generator support — slice 5b", category: "SHOW/ALTER/DDL"},
		{name: "bit literal identity", deferredReason: "bit/hex literal parser and generator rendering — slice 5b", category: "param/literal rendering"},
		{name: "unsigned ddl", deferredReason: "MySQL DDL constraints and type mapping — slice 5b", category: "SHOW/ALTER/DDL"},
	}
	runIdentityCases(t, "test_mysql validate_identity", cases)
}

func TestMySQLValidateAllKeys(t *testing.T) {
	cases := []identityCase{
		{name: "invalid hex mysql key", dialect: "mysql", sql: "SELECT 0xz", want: "SELECT `0xz`"},
		{name: "bare hex prefix mysql key", dialect: "mysql", sql: "SELECT 0x", want: "SELECT `0x`"},
		{name: "bare bit prefix mysql key", dialect: "mysql", sql: "SELECT 0b", want: "SELECT `0b`"},
		{name: "insert default duckdb key", deferredReason: "cross-dialect: needs duckdb", category: "cross-dialect"},
	}
	runIdentityCases(t, "test_mysql validate_all in-scope keys", cases)
}

func TestMySQLProbeQualifyTraverseScope(t *testing.T) {
	expression, err := sqlglot.ParseOne("SELECT Foo, bar FROM MyTable", "mysql")
	if err != nil {
		t.Fatalf("ParseOne(mysql): %v", err)
	}
	opts := optimizer.DefaultQualifyOpts()
	opts.Dialect = "mysql"
	opts.Schema = schema.M("MyTable", schema.M("Foo", "INT", "bar", "INT"))
	opts.InferSchema = boolPtr(true)
	qualified := optimizer.Qualify(expression, opts)
	got, err := sqlglot.Generate(qualified, "mysql", generator.Options{})
	if err != nil {
		t.Fatalf("Generate(mysql): %v", err)
	}
	want := "SELECT `MyTable`.`Foo` AS `Foo`, `MyTable`.`bar` AS `bar` FROM `MyTable` AS `MyTable`"
	if got != want {
		t.Fatalf("qualified mysql = %q, want %q", got, want)
	}
	if scopes := optimizer.TraverseScope(qualified); len(scopes) != 1 {
		t.Fatalf("TraverseScope(mysql) len = %d, want 1", len(scopes))
	}
}
