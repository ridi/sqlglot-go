package parser_test

import (
	"testing"

	exp "github.com/sjincho/sqlglot-go/expressions"
)

func TestJSONOperators(t *testing.T) {
	cases := []struct {
		sql  string
		kind exp.Kind
	}{
		{"SELECT a -> 'b'", exp.KindJSONExtract},
		{"SELECT a ->> 'b'", exp.KindJSONExtractScalar},
		{"SELECT a #> '{b}'", exp.KindJSONBExtract},
		{"SELECT a #>> '{b}'", exp.KindJSONBExtractScalar},
	}
	for _, tc := range cases {
		projection := parseOne(t, tc.sql).Expressions()[0]
		if projection.Kind() != tc.kind {
			t.Fatalf("%s: projection kind = %v, want %v:\n%s", tc.sql, projection.Kind(), tc.kind, projection.ToS())
		}
		if rhs := exprArg(t, projection, "expression"); rhs.Kind() != exp.KindLiteral {
			t.Fatalf("%s: RHS kind = %v, want Literal:\n%s", tc.sql, rhs.Kind(), projection.ToS())
		}
	}

	projection := parseOne(t, "SELECT a -> 'b' -> 'c'").Expressions()[0]
	if projection.Kind() != exp.KindJSONExtract || projection.This() == nil || projection.This().Kind() != exp.KindJSONExtract {
		t.Fatalf("nested JSON extract mismatch:\n%s", projection.ToS())
	}

	projection = parseOne(t, "SELECT x::INT").Expressions()[0]
	if projection.Kind() != exp.KindCast {
		t.Fatalf("dcolon cast regression: kind = %v, want Cast:\n%s", projection.Kind(), projection.ToS())
	}
}
