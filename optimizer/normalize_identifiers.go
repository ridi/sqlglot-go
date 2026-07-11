package optimizer

import (
	"github.com/sjincho/sqlglot-go/dialects"
	exp "github.com/sjincho/sqlglot-go/expressions"
)

// NormalizeIdentifiers folds unquoted identifiers per the dialect's normalization strategy.
// dialect is a DialectType-style value (nil | string | *dialects.Dialect), mirroring upstream
// sqlglot's normalize_identifiers(expression, dialect: DialectType).
func NormalizeIdentifiers(expression exp.Expression, dialect any) exp.Expression {
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

func NormalizeIdentifiersString(name string, dialect any) exp.Expression {
	// ParseIdentifier only needs the dialect's tokenizer/quoting (strategy-independent), so
	// resolve any->name for it; NormalizeIdentifiers still applies the full strategy.
	d, err := dialects.GetOrRaise(dialect)
	if err != nil {
		panic(err)
	}
	return NormalizeIdentifiers(exp.ParseIdentifier(name, d.Name), dialect)
}
