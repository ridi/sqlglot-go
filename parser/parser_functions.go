package parser

import (
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/tokens"
)

func seqGet[T any](s []T, i int) T {
	var zero T
	if i < 0 || i >= len(s) {
		return zero
	}
	return s[i]
}

func (p *Parser) parseConvert(strict bool, safe any) exp.Expression {
	this := p.parseBitwise()
	var to exp.Expression
	if p.match(tokens.USING) {
		// TODO(1d): CONVERT charset form.
	} else if p.match(tokens.COMMA) {
		to = p.parseTypes(false, false, true, false)
	}
	args := exp.Args{"this": this, "to": to}
	if safe != nil {
		args["safe"] = safe
	}
	return p.buildCast(strict, args)
}

func (p *Parser) parseCeilFloor(kind exp.Kind) exp.Expression {
	args := p.parseCsv(func() exp.Expression { return p.parseLambda(false) })
	this := seqGet(args, 0)
	decimals := seqGet(args, 1)
	var to exp.Expression
	if p.matchTextSeq("TO") {
		to = p.parseVar(false, nil, false)
	}
	return p.expression(exp.New(kind, exp.Args{"this": this, "decimals": decimals, "to": to}), nil, nil)
}

func (p *Parser) parseExtract() exp.Expression {
	// Mirror upstream `self._parse_function() or self._parse_var_or_string(...)`:
	// short-circuit so the second alternative never runs (and never advances past
	// FROM via parseVar's advanceAny) once the first succeeds.
	this := p.parseFunction(nil, false, true, false)
	if this == nil {
		this = p.parseVarOrString(true)
	}
	if p.match(tokens.FROM) {
		return p.expression(exp.Extract(exp.Args{"this": this, "expression": p.parseBitwise()}), nil, nil)
	}
	if !p.match(tokens.COMMA) {
		p.raiseError("Expected FROM or comma after EXTRACT", p.prev)
	}
	return p.expression(exp.Extract(exp.Args{"this": this, "expression": p.parseBitwise()}), nil, nil)
}

func (p *Parser) parsePosition() exp.Expression {
	args := p.parseCsv(p.parseBitwise)
	if p.match(tokens.IN) {
		return p.expression(exp.StrPosition(exp.Args{"this": p.parseBitwise(), "substr": seqGet(args, 0)}), nil, nil)
	}
	return p.expression(exp.StrPosition(exp.Args{"this": seqGet(args, 1), "substr": seqGet(args, 0), "position": seqGet(args, 2)}), nil, nil)
}

func (p *Parser) parseSubstring() exp.Expression {
	args := p.parseCsv(p.parseBitwise)
	var start exp.Expression
	var length exp.Expression
	for p.curr.IsValid() {
		if p.match(tokens.FROM) {
			start = p.parseBitwise()
		} else if p.match(tokens.FOR) {
			if start == nil {
				start = exp.LiteralNumber(1)
			}
			length = p.parseBitwise()
		} else {
			break
		}
	}
	if start != nil {
		args = append(args, start)
	}
	if length != nil {
		args = append(args, length)
	}
	return p.validateExpression(exp.FromArgList(exp.KindSubstring, args), exprArgs(args))
}

func (p *Parser) parseTrim() exp.Expression {
	var position any
	var collation exp.Expression
	var expression exp.Expression
	if p.matchTexts(trimTypes) {
		position = stringsUpper(p.prev.Text)
	}
	this := p.parseBitwise()
	if p.matchSet(map[tokens.TokenType]bool{tokens.FROM: true, tokens.COMMA: true}) {
		invert := p.prev.TokenType == tokens.FROM
		expression = p.parseBitwise()
		if invert {
			this, expression = expression, this
		}
	}
	if p.match(tokens.COLLATE) {
		collation = p.parseBitwise()
	}
	return p.expression(exp.Trim(exp.Args{"this": this, "position": position, "expression": expression, "collation": collation}), nil, nil)
}

func (p *Parser) parseStringAgg() exp.Expression {
	var args []exp.Expression
	if p.match(tokens.DISTINCT) {
		args = []exp.Expression{p.expression(exp.Distinct(exp.Args{"expressions": []exp.Expression{p.parseDisjunction()}}), nil, nil)}
		if p.match(tokens.COMMA) {
			args = append(args, p.parseCsv(p.parseDisjunction)...)
		}
	} else {
		args = p.parseCsv(p.parseDisjunction)
	}
	index := p.index
	if !p.match(tokens.R_PAREN) && len(args) > 0 {
		ordered := p.parseOrder(seqGet(args, 0), false)
		args[0] = p.parseLimit(ordered, false, false)
		return p.expression(exp.GroupConcat(exp.Args{"this": args[0], "separator": seqGet(args, 1)}), nil, nil)
	}
	if !p.matchTextSeq("WITHIN", "GROUP") {
		p.retreat(index)
		return p.validateExpression(exp.FromArgList(exp.KindGroupConcat, args), exprArgs(args))
	}
	p.matchLParen(nil)
	return p.expression(exp.GroupConcat(exp.Args{"this": p.parseOrder(seqGet(args, 0), false), "separator": seqGet(args, 1)}), nil, nil)
}
