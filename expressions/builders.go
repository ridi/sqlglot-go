package expressions

import (
	"fmt"
	"math"
	"regexp"
)

var safeIdentifierRE = regexp.MustCompile(`^[_a-zA-Z]\w*$`)

var MaybeParseFunc func(sql string, dialect string) (Expression, error)

func ToIdentifier(name any, quoted ...bool) Expression {
	if name == nil {
		return nil
	}
	if expr, ok := name.(Expression); ok {
		if expr.Kind() == KindIdentifier {
			return expr
		}
	}
	text, ok := name.(string)
	if !ok {
		return nil
	}
	quote := !safeIdentifierRE.MatchString(text)
	if len(quoted) > 0 {
		quote = quoted[0]
	}
	return Identifier(Args{"this": text, "quoted": quote})
}

func AliasExpr(expression any, alias any, table bool, quoted ...bool) Expression {
	expr := MaybeParse(expression, "", false)
	aliasExpr := ToIdentifier(alias, quoted...)
	if table {
		expr.Set("alias", TableAlias(Args{"this": aliasExpr}))
		return expr
	}
	if supportsArg(expr.Kind(), "alias") && expr.Kind() != KindExpression {
		expr.Set("alias", aliasExpr)
		return expr
	}
	return AliasNode(Args{"this": expr, "alias": aliasExpr})
}

func Column_(col any, table any, db any, catalog any, fields []any, quoted ...bool) Expression {
	var colExpr Expression
	if expr, ok := col.(Expression); ok && expr.Kind() == KindStar {
		colExpr = expr
	} else {
		colExpr = ToIdentifier(col, quoted...)
	}
	this := Column(Args{
		"this":    colExpr,
		"table":   ToIdentifier(table, quoted...),
		"db":      ToIdentifier(db, quoted...),
		"catalog": ToIdentifier(catalog, quoted...),
	})
	if len(fields) > 0 {
		parts := []Expression{this}
		for _, field := range fields {
			parts = append(parts, ToIdentifier(field, quoted...))
		}
		return DotBuild(parts)
	}
	return this
}

func Convert(value any, copyValue bool) Expression {
	if expr, ok := value.(Expression); ok {
		if copyValue {
			return expr.Copy()
		}
		return expr
	}
	switch v := value.(type) {
	case string:
		return LiteralString(v)
	case bool:
		return Boolean(Args{"this": v})
	case nil:
		return Null()
	case int:
		return LiteralNumber(v)
	case int64:
		return LiteralNumber(v)
	case float64:
		if math.IsNaN(v) {
			return Null()
		}
		return LiteralNumber(v)
	case float32:
		return LiteralNumber(v)
	}
	panic(fmt.Sprintf("Cannot convert %v", value))
}

func MaybeParse(sqlOrExpression any, dialect string, copyValue bool) Expression {
	if expr, ok := sqlOrExpression.(Expression); ok {
		if copyValue {
			return expr.Copy()
		}
		return expr
	}
	if sqlOrExpression == nil {
		panic("SQL cannot be None")
	}
	if MaybeParseFunc == nil {
		return Convert(sqlOrExpression, copyValue)
	}
	expr, err := MaybeParseFunc(fmt.Sprint(sqlOrExpression), dialect)
	if err != nil {
		panic(err)
	}
	return expr
}

func Condition(expression any, dialect string, copyValue bool) Expression {
	return MaybeParse(expression, dialect, copyValue)
}

func And_(expressions ...Expression) Expression {
	return combine(expressions, KindAnd, true)
}

func Or_(expressions ...Expression) Expression {
	return combine(expressions, KindOr, true)
}

func Not_(expression Expression, copyValue bool) Expression {
	if copyValue && expression != nil {
		expression = expression.Copy()
	}
	return Not(Args{"this": wrap(expression, TraitConnector)})
}

func combine(expressions []Expression, kind Kind, wrapOperands bool) Expression {
	filtered := make([]Expression, 0, len(expressions))
	for _, expression := range expressions {
		if expression != nil {
			filtered = append(filtered, expression)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	this := filtered[0]
	if len(filtered) > 1 && wrapOperands {
		this = wrap(this, TraitConnector)
	}
	for _, expression := range filtered[1:] {
		if wrapOperands {
			expression = wrap(expression, TraitConnector)
		}
		args := Args{"this": this, "expression": expression}
		if kind == KindAnd {
			this = And(args)
		} else {
			this = Or(args)
		}
	}
	return this
}

func wrap(expression Expression, trait Trait) Expression {
	if expression != nil && expression.Is(trait) {
		return Paren(Args{"this": expression})
	}
	return expression
}

func supportsArg(kind Kind, key string) bool {
	for _, spec := range argTypesFor(kind) {
		if spec.Key == key {
			return true
		}
	}
	return false
}
