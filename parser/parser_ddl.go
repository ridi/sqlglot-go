package parser

import (
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/tokens"
)

var rParenCommaSet = map[tokens.TokenType]bool{tokens.R_PAREN: true, tokens.COMMA: true}

func (p *Parser) parseCreate() exp.Expression {
	start := p.prev
	// Attempt a structured CREATE under error isolation, then degrade to a raw Command
	// whenever it can't be parsed cleanly: an unsupported creatable (FUNCTION/INDEX/...),
	// a deferred column constraint or property (1c), or trailing junk after the body.
	// tryParse runs the body at IMMEDIATE error level, so a partial parse (e.g.
	// parseSchema/matchRParen hitting an unparsed `NOT NULL`) panics-and-retreats instead
	// of leaving a stale "Expecting )" that would later poison checkErrors. This keeps the
	// documented graceful degradation (plan §5 Q#1) clean.
	if structured := p.tryParse(func() exp.Expression { return p.parseCreateStructured(start) }, false); structured != nil {
		return structured
	}
	return p.parseAsCommand(start)
}

// parseCreateStructured parses a CREATE into an exp.Create, or returns nil (signalling the
// caller to degrade to a Command) when the statement isn't a structured creatable we
// support or carries trailing tokens we don't yet parse. Under tryParse it may also raise
// on a malformed/deferred body, which tryParse converts to nil.
func (p *Parser) parseCreateStructured(start tokens.Token) exp.Expression {
	replace := start.TokenType == tokens.REPLACE || p.matchPair(tokens.OR, tokens.REPLACE, true) || p.matchPair(tokens.OR, tokens.ALTER, true)
	refresh := p.matchPair(tokens.OR, tokens.REFRESH, true)
	unique := p.match(tokens.UNIQUE)
	if !p.matchSet(creatables) {
		return nil
	}
	createToken := p.prev
	ctt := createToken.TokenType
	concurrently := p.matchTextSeq("CONCURRENTLY")
	exists := p.parseExists(true)

	var this exp.Expression
	var expression exp.Expression
	var noSchemaBinding any
	if dbCreatables[ctt] {
		tableParts := p.parseTableParts(true, ctt == tokens.SCHEMA, false, false)
		p.match(tokens.COMMA)
		this = p.parseSchema(tableParts)
		hasAlias := p.match(tokens.ALIAS)
		expression = p.parseDDLSelect()
		if expression == nil && hasAlias {
			expression = p.tryParse(func() exp.Expression { return p.parseTableParts(false, false, false, false) }, false)
		}
		if ctt == tokens.VIEW && p.matchTextSeq("WITH", "NO", "SCHEMA", "BINDING") {
			noSchemaBinding = true
		}
	} else {
		return nil
	}
	if p.curr.IsValid() && !p.matchSet(rParenCommaSet, false) {
		return nil
	}
	kind := stringsUpper(createToken.Text)
	return p.expression(exp.Create(exp.Args{
		"this":              this,
		"kind":              kind,
		"replace":           replace,
		"refresh":           refresh,
		"unique":            unique,
		"expression":        expression,
		"exists":            exists,
		"no_schema_binding": noSchemaBinding,
		"concurrently":      concurrently,
	}), nil, nil)
}

func (p *Parser) parseSchema(this exp.Expression) exp.Expression {
	index := p.index
	if !p.match(tokens.L_PAREN) {
		return this
	}
	if p.matchSet(selectStartTokens) {
		p.retreat(index)
		return this
	}
	args := p.parseCsv(func() exp.Expression {
		if c := p.parseConstraint(); c != nil {
			return c
		}
		return p.parseFieldDef()
	})
	p.matchRParen(nil)
	return p.expression(exp.Schema(exp.Args{"this": this, "expressions": args}), nil, nil)
}

func (p *Parser) parseFieldDef() exp.Expression {
	return p.parseColumnDef(p.parseField(true, nil, false))
}

func (p *Parser) parseColumnDef(this exp.Expression) exp.Expression {
	if this != nil && this.Kind() == exp.KindColumn {
		this = this.This()
	}
	// schema=true mirrors upstream _parse_column_def (parser.py:7255): it enables the
	// fixed-size-array form (e.g. `col INT[3]`) in column definitions.
	kind := p.parseTypes(false, true, true, false)
	constraints := []exp.Expression{}
	for {
		constraint := p.parseColumnConstraint()
		if constraint == nil {
			break
		}
		constraints = append(constraints, constraint)
	}
	if kind == nil && len(constraints) == 0 {
		return this
	}
	return p.expression(exp.ColumnDef(exp.Args{"this": this, "kind": kind, "constraints": constraints}), nil, nil)
}

func (p *Parser) parseConstraint() exp.Expression { return nil }

func (p *Parser) parseColumnConstraint() exp.Expression { return nil }

func (p *Parser) parseAsCommand(start tokens.Token) exp.Expression {
	for p.curr.IsValid() {
		p.advance()
	}
	text := p.findSQL(start, p.prev)
	runes := []rune(text)
	size := len([]rune(start.Text))
	return p.expression(exp.Command(exp.Args{"this": string(runes[:size]), "expression": string(runes[size:])}), nil, nil)
}

func (p *Parser) parseDDLSelect() exp.Expression {
	return p.parseQueryModifiers(p.parseSetOperations(p.parseSelect(true, false, false)))
}
