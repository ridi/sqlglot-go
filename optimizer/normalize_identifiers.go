package optimizer

import (
	"github.com/sjincho/sqlglot-go/dialects"
	exp "github.com/sjincho/sqlglot-go/expressions"
)

func NormalizeIdentifiers(expression exp.Expression, dialect string) exp.Expression {
	d, err := dialects.GetOrRaise(dialect)
	if err != nil {
		panic(err)
	}
	if expression == nil {
		return nil
	}
	for _, node := range expression.WalkWithPrune(true, func(n exp.Expression) bool { return false }) {
		// TODO(slice 5): meta_get("case_sensitive") prune/skip + store_original_column_identifiers
		// needs Node.Meta (see schema.go:440).
		if node.Kind() == exp.KindIdentifier {
			d.NormalizeIdentifier(node)
		}
	}
	return expression
}

func NormalizeIdentifiersString(name string, dialect string) exp.Expression {
	return NormalizeIdentifiers(exp.ParseIdentifier(name, dialect), dialect)
}
