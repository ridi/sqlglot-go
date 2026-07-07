package sqlglot

import (
	fmtstd "fmt"

	"github.com/sjincho/sqlglot-go/dialects"
	sqlerrors "github.com/sjincho/sqlglot-go/errors"
	"github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/parser"
	"github.com/sjincho/sqlglot-go/tokens"
)

func Tokenize(sql string, dialect string) ([]tokens.Token, error) {
	d, err := dialects.GetOrRaise(dialect)
	if err != nil {
		return nil, err
	}
	return d.NewTokenizer().Tokenize(sql)
}

func Parse(sql string, dialect string) ([]expressions.Expression, error) {
	d, err := dialects.GetOrRaise(dialect)
	if err != nil {
		return nil, err
	}
	tokenizer := d.NewTokenizer()
	toks, err := tokenizer.Tokenize(sql)
	if err != nil {
		return nil, err
	}
	p := parser.New(d)
	return p.Parse(toks, sql)
}

func ParseOne(sql string, dialect string) (expressions.Expression, error) {
	res, err := Parse(sql, dialect)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 || res[0] == nil {
		return nil, sqlerrors.NewParseError(fmtstd.Sprintf("No expression was parsed from '%s'", sql))
	}
	if len(res) > 1 {
		return expressions.Block(expressions.Args{"expressions": res}), nil
	}
	return res[0], nil
}

func init() {
	expressions.MaybeParseFunc = func(sql string, dialect string) (expressions.Expression, error) {
		return ParseOne(sql, dialect)
	}
}
