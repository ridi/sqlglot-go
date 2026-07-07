package parser_test

import (
	"testing"

	exp "github.com/sjincho/sqlglot-go/expressions"
)

func TestParseCastAndTypes(t *testing.T) {
	cast := parseOne(t, "CAST(x AS INT)")
	if cast.Kind() != exp.KindCast || !exp.IsType(exprArg(t, cast, "to"), exp.DTypeInt) {
		t.Fatalf("CAST type mismatch:\n%s", cast.ToS())
	}

	cast = parseOne(t, "CAST(x AS DECIMAL(10, 2))")
	to := exprArg(t, cast, "to")
	if !exp.IsType(to, exp.DTypeDecimal) || len(to.Expressions()) != 2 || to.Expressions()[0].Kind() != exp.KindDataTypeParam {
		t.Fatalf("DECIMAL params mismatch:\n%s", cast.ToS())
	}

	cast = parseOne(t, "x::int")
	if cast.Kind() != exp.KindCast || exprArg(t, cast, "this").Kind() != exp.KindColumn || !exp.IsType(exprArg(t, cast, "to"), exp.DTypeInt) {
		t.Fatalf("dcolon cast mismatch:\n%s", cast.ToS())
	}

	for _, sql := range []string{"TRY_CAST(x AS INT)", "SAFE_CAST(x AS INT)"} {
		tryCast := parseOne(t, sql)
		if tryCast.Kind() != exp.KindTryCast || tryCast.Arg("safe") != true {
			t.Fatalf("%s mismatch:\n%s", sql, tryCast.ToS())
		}
	}
}

func TestParseBracketAndArray(t *testing.T) {
	proj := parseOne(t, "SELECT a[0] FROM t").Expressions()[0]
	if proj.Kind() != exp.KindBracket || exprArg(t, proj, "this").Kind() != exp.KindColumn {
		t.Fatalf("bracket mismatch:\n%s", proj.ToS())
	}

	proj = parseOne(t, "SELECT a[0].b FROM t").Expressions()[0]
	if proj.Kind() != exp.KindDot || exprArg(t, proj, "this").Kind() != exp.KindBracket {
		t.Fatalf("bracket dot mismatch:\n%s", proj.ToS())
	}

	proj = parseOne(t, "SELECT [1, 2]").Expressions()[0]
	if proj.Kind() != exp.KindArray || len(proj.Expressions()) != 2 {
		t.Fatalf("array literal mismatch:\n%s", proj.ToS())
	}
}

func TestParseSpecialFunctions(t *testing.T) {
	cases := []struct {
		sql  string
		kind exp.Kind
	}{
		{"EXTRACT(DAY FROM x)", exp.KindExtract},
		{"SUBSTRING(x FROM 1 FOR 2)", exp.KindSubstring},
		{"TRIM(BOTH ' ' FROM x)", exp.KindTrim},
		{"POSITION(a IN b)", exp.KindStrPosition},
		{"CEIL(x)", exp.KindCeil},
		{"FLOOR(x, 2)", exp.KindFloor},
		{"STRING_AGG(x, ',')", exp.KindGroupConcat},
	}
	for _, tc := range cases {
		expression := parseOne(t, tc.sql)
		if expression.Kind() != tc.kind {
			t.Fatalf("%s kind = %v, want %v:\n%s", tc.sql, expression.Kind(), tc.kind, expression.ToS())
		}
	}
	trim := parseOne(t, "TRIM(BOTH ' ' FROM x)")
	if trim.Arg("position") != "BOTH" {
		t.Fatalf("trim position mismatch:\n%s", trim.ToS())
	}

	// EXTRACT's first arg alternatives must short-circuit (mirror Python `or`): when
	// the function/literal alternative matches, the FROM keyword must survive so the
	// FROM branch is taken (LOCAL fix for eager firstExpression / parseVarOrString).
	for _, sql := range []string{"EXTRACT(foo() FROM x)", "EXTRACT('lit' FROM x)", "EXTRACT(DAY FROM x)"} {
		extract := parseOne(t, sql)
		if extract.Kind() != exp.KindExtract {
			t.Fatalf("%s kind mismatch:\n%s", sql, extract.ToS())
		}
		if expr := exprArg(t, extract, "expression"); expr.Kind() != exp.KindColumn || expr.Name() != "x" {
			t.Fatalf("%s FROM operand mismatch:\n%s", sql, extract.ToS())
		}
	}
}

func TestNestedTypes(t *testing.T) {
	to := exprArg(t, parseOne(t, "CAST(x AS ARRAY<INT>)"), "to")
	if !exp.IsType(to, exp.DTypeArray) || len(to.Expressions()) != 1 || !exp.IsType(to.Expressions()[0], exp.DTypeInt) {
		t.Fatalf("ARRAY type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS STRUCT<a INT, b STRING>)"), "to")
	if !exp.IsType(to, exp.DTypeStruct) || len(to.Expressions()) != 2 || to.Expressions()[0].Kind() != exp.KindColumnDef {
		t.Fatalf("STRUCT type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS MAP<STRING, INT>)"), "to")
	if !exp.IsType(to, exp.DTypeMap) || len(to.Expressions()) != 2 {
		t.Fatalf("MAP type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS MAP[STRING=>INT])"), "to")
	if !exp.IsType(to, exp.DTypeMap) || len(to.Expressions()) != 2 {
		t.Fatalf("MAP bracket type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS ARRAY<STRING COLLATE utf8>)"), "to")
	if !exp.IsType(to, exp.DTypeArray) || len(to.Expressions()) != 1 || to.Expressions()[0].Arg("collate") == nil {
		t.Fatalf("COLLATE nested type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS INT UNSIGNED)"), "to")
	if !exp.IsType(to, exp.DTypeUInt) {
		t.Fatalf("UNSIGNED type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS TIMESTAMP WITH TIME ZONE)"), "to")
	if !exp.IsType(to, exp.DTypeTimestampTz) {
		t.Fatalf("TIMESTAMP WITH TIME ZONE mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS INT[])"), "to")
	if !exp.IsType(to, exp.DTypeArray) || len(to.Expressions()) != 1 || !exp.IsType(to.Expressions()[0], exp.DTypeInt) {
		t.Fatalf("array suffix type mismatch:\n%s", to.ToS())
	}

	to = exprArg(t, parseOne(t, "CAST(x AS NULLABLE(INT))"), "to")
	if !exp.IsType(to, exp.DTypeInt) || to.Arg("nullable") != true {
		t.Fatalf("NULLABLE type mismatch:\n%s", to.ToS())
	}

	// Top-level CAST(... AS <type> COLLATE ...) must parse (with_collation=True),
	// not hard-error on the COLLATE token (LOCAL fix; parser.py:7863).
	to = exprArg(t, parseOne(t, "CAST(x AS VARCHAR COLLATE utf8)"), "to")
	if !exp.IsType(to, exp.DTypeVarchar) || to.Arg("collate") == nil {
		t.Fatalf("top-level COLLATE cast mismatch:\n%s", to.ToS())
	}
}

// A fixed-size array column definition (`col INT[3]`) must parse into a structured
// Create with an ARRAY DataType carrying values, not degrade to a Command. This
// exercises parseTypes' schema=true path via _parse_column_def (LOCAL fix).
func TestFixedSizeArrayColumn(t *testing.T) {
	create := parseOne(t, "CREATE TABLE t (col INT[3])")
	if create.Kind() != exp.KindCreate {
		t.Fatalf("fixed-size array column should parse to Create:\n%s", create.ToS())
	}
	col := exprArg(t, create, "this").Expressions()[0]
	kind := exprArg(t, col, "kind")
	if !exp.IsType(kind, exp.DTypeArray) || len(expressionsForArg(kind, "values")) != 1 {
		t.Fatalf("fixed-size array type mismatch:\n%s", kind.ToS())
	}
	if len(kind.Expressions()) != 1 || !exp.IsType(kind.Expressions()[0], exp.DTypeInt) {
		t.Fatalf("fixed-size array element type mismatch:\n%s", kind.ToS())
	}
}

func TestIntervalType(t *testing.T) {
	to := exprArg(t, parseOne(t, "CAST(x AS INTERVAL DAY)"), "to")
	interval := exprArg(t, to, "this")
	if interval.Kind() != exp.KindInterval {
		t.Fatalf("INTERVAL type inner kind = %v, want Interval:\n%s", interval.Kind(), to.ToS())
	}
	unit := exprArg(t, interval, "unit")
	if unit.Kind() != exp.KindVar || unit.Name() != "DAY" {
		t.Fatalf("INTERVAL unit mismatch:\n%s", interval.ToS())
	}
}

func TestIntervalLiteral(t *testing.T) {
	for _, sql := range []string{"INTERVAL '1' DAY", "INTERVAL 1 DAY", "INTERVAL '1 day'"} {
		interval := parseOne(t, sql)
		if interval.Kind() != exp.KindInterval {
			t.Fatalf("%s: kind = %v, want Interval:\n%s", sql, interval.Kind(), interval.ToS())
		}
		unit := exprArg(t, interval, "unit")
		if unit.Kind() != exp.KindVar {
			t.Fatalf("%s: unit kind = %v, want Var:\n%s", sql, unit.Kind(), interval.ToS())
		}
	}

	interval := parseOne(t, "INTERVAL '1' DAY TO SECOND")
	unit := exprArg(t, interval, "unit")
	if unit.Kind() != exp.KindIntervalSpan {
		t.Fatalf("INTERVAL span unit kind = %v, want IntervalSpan:\n%s", unit.Kind(), interval.ToS())
	}

	// Numeric interval literals canonicalize to a string literal via Python-style str():
	// integers drop no digits, integer-valued decimals keep ".0", and Neg keeps its sign.
	for _, tc := range []struct{ sql, want string }{
		{"INTERVAL 1 DAY", "1"},
		{"INTERVAL 1.0 DAY", "1.0"},
		{"INTERVAL 1.5 DAY", "1.5"},
		{"INTERVAL -1 DAY", "-1"},
	} {
		iv := parseOne(t, tc.sql)
		if got := exprArg(t, iv, "this").Name(); got != tc.want {
			t.Fatalf("%s: interval value = %q, want %q:\n%s", tc.sql, got, tc.want, iv.ToS())
		}
	}
}
