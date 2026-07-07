package parser_test

import (
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	exp "github.com/sjincho/sqlglot-go/expressions"
)

func parseOne(t *testing.T, sql string) exp.Expression {
	t.Helper()
	expression, err := sqlglot.ParseOne(sql, "")
	if err != nil {
		t.Fatalf("ParseOne(%q) error: %v", sql, err)
	}
	return expression
}

func expressionsForArg(expression exp.Expression, key string) []exp.Expression {
	value := expression.Arg(key)
	if expressions, ok := value.([]exp.Expression); ok {
		return expressions
	}
	return nil
}
