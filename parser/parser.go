package parser

import (
	"fmt"

	"github.com/sjincho/sqlglot-go/dialects"
	sqlerrors "github.com/sjincho/sqlglot-go/errors"
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/tokens"
)

type Parser struct {
	errorLevel          sqlerrors.ErrorLevel
	errorMessageContext int
	maxErrors           int
	maxNodes            int
	dialect             *dialects.Dialect
	sql                 string
	errors              []*sqlerrors.ParseError

	tokens     []tokens.Token
	tokensSize int
	index      int
	curr       tokens.Token
	next       tokens.Token
	prev       tokens.Token

	prevComments []string
	chunks       [][]tokens.Token
	chunkIndex   int
	nodeCount    int
}

func New(d *dialects.Dialect) *Parser {
	return NewWithErrorLevel(d, sqlerrors.IMMEDIATE)
}

func NewWithErrorLevel(d *dialects.Dialect, level sqlerrors.ErrorLevel) *Parser {
	if d == nil {
		d = dialects.Base()
	}
	p := &Parser{
		errorLevel:          level,
		errorMessageContext: 100,
		maxErrors:           3,
		maxNodes:            -1,
		dialect:             d,
		curr:                tokens.SentinelNone,
		next:                tokens.SentinelNone,
		prev:                tokens.SentinelNone,
	}
	return p
}

func (p *Parser) Reset() {
	p.sql = ""
	p.errors = nil
	p.tokens = nil
	p.tokensSize = 0
	p.index = 0
	p.curr = tokens.SentinelNone
	p.next = tokens.SentinelNone
	p.prev = tokens.SentinelNone
	p.prevComments = nil
	p.chunks = nil
	p.chunkIndex = 0
	p.nodeCount = 0
}

func (p *Parser) Errors() []*sqlerrors.ParseError { return p.errors }

func (p *Parser) advance(times ...int) {
	t := 1
	if len(times) > 0 {
		t = times[0]
	}
	index := p.index + t
	p.index = index
	if index >= 0 && index < p.tokensSize {
		p.curr = p.tokens[index]
	} else {
		p.curr = tokens.SentinelNone
	}
	if index+1 >= 0 && index+1 < p.tokensSize {
		p.next = p.tokens[index+1]
	} else {
		p.next = tokens.SentinelNone
	}
	if index > 0 && index-1 < p.tokensSize {
		prev := p.tokens[index-1]
		p.prev = prev
		p.prevComments = prev.Comments
	} else {
		p.prev = tokens.SentinelNone
		p.prevComments = nil
	}
}

func (p *Parser) advanceChunk() {
	p.index = -1
	p.tokens = p.chunks[p.chunkIndex]
	p.tokensSize = len(p.tokens)
	p.chunkIndex++
	p.advance()
}

func (p *Parser) retreat(index int) {
	if index != p.index {
		p.advance(index - p.index)
	}
}

func (p *Parser) addComments(expression exp.Expression) {
	if expression != nil && len(p.prevComments) > 0 {
		expression.AddComments(p.prevComments, false)
		p.prevComments = nil
	}
}

func (p *Parser) match(tokenType tokens.TokenType, advance ...bool) bool {
	shouldAdvance := true
	if len(advance) > 0 {
		shouldAdvance = advance[0]
	}
	if p.curr.TokenType == tokenType {
		if shouldAdvance {
			p.advance()
		}
		return true
	}
	return false
}

func (p *Parser) matchSet(types map[tokens.TokenType]bool, advance ...bool) bool {
	shouldAdvance := true
	if len(advance) > 0 {
		shouldAdvance = advance[0]
	}
	if types[p.curr.TokenType] {
		if shouldAdvance {
			p.advance()
		}
		return true
	}
	return false
}

func (p *Parser) matchPair(a, b tokens.TokenType, advance bool) bool {
	if p.curr.TokenType == a && p.next.TokenType == b {
		if advance {
			p.advance(2)
		}
		return true
	}
	return false
}

func (p *Parser) matchTexts(texts map[string]bool, advance ...bool) bool {
	shouldAdvance := true
	if len(advance) > 0 {
		shouldAdvance = advance[0]
	}
	if p.curr.TokenType != tokens.STRING && texts[stringsUpper(p.curr.Text)] {
		if shouldAdvance {
			p.advance()
		}
		return true
	}
	return false
}

func (p *Parser) matchTextSeq(texts ...string) bool {
	index := p.index
	for _, text := range texts {
		if p.curr.TokenType != tokens.STRING && stringsUpper(p.curr.Text) == text {
			p.advance()
		} else {
			p.retreat(index)
			return false
		}
	}
	return true
}

func (p *Parser) isConnected() bool {
	return p.prev.IsValid() && p.curr.IsValid() && p.prev.End+1 == p.curr.Start
}

func (p *Parser) findSQL(start, end tokens.Token) string {
	runes := []rune(p.sql)
	if start.Start < 0 || end.End >= len(runes) || start.Start > end.End {
		return ""
	}
	return string(runes[start.Start : end.End+1])
}

func (p *Parser) raiseError(message string, tok ...tokens.Token) {
	token := tokens.SentinelNone
	if len(tok) > 0 {
		token = tok[0]
	}
	if !token.IsValid() {
		if p.curr.IsValid() {
			token = p.curr
		} else if p.prev.IsValid() {
			token = p.prev
		} else {
			token = tokens.StringToken("")
		}
	}
	formattedSQL, startContext, highlight, endContext := sqlerrors.HighlightSQL(
		p.sql,
		[][2]int{{token.Start, token.End}},
		p.errorMessageContext,
	)
	formattedMessage := fmt.Sprintf("%s. Line %d, Col: %d.\n  %s", message, token.Line, token.Col, formattedSQL)
	err := sqlerrors.NewParseErrorWithLocation(formattedMessage, message, token.Line, token.Col, startContext, highlight, endContext)
	if p.errorLevel == sqlerrors.IMMEDIATE {
		panic(err)
	}
	p.errors = append(p.errors, err)
}

func (p *Parser) validateExpression(expression exp.Expression, args []any) exp.Expression {
	if p.maxNodes > -1 {
		p.nodeCount++
		if p.nodeCount > p.maxNodes {
			p.raiseError(fmt.Sprintf("Maximum number of AST nodes (%d) exceeded", p.maxNodes))
		}
	}
	if p.errorLevel != sqlerrors.IGNORE {
		for _, message := range expression.ErrorMessages(args) {
			p.raiseError(message)
		}
	}
	return expression
}

func (p *Parser) tryParse(parseMethod func() exp.Expression, retreat bool) (out exp.Expression) {
	index := p.index
	errorLevel := p.errorLevel
	p.errorLevel = sqlerrors.IMMEDIATE
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*sqlerrors.ParseError); ok {
				out = nil
			} else {
				panic(r)
			}
		}
		if out == nil || retreat {
			p.retreat(index)
		}
		p.errorLevel = errorLevel
	}()
	out = parseMethod()
	return out
}

func (p *Parser) Parse(rawTokens []tokens.Token, sql string) (expressions []exp.Expression, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case *sqlerrors.ParseError:
				err = e
			case error:
				err = e
			default:
				panic(r)
			}
		}
	}()
	return p.parse(p.parseStatement, rawTokens, sql), nil
}

func (p *Parser) checkErrors() {
	if p.errorLevel == sqlerrors.RAISE && len(p.errors) > 0 {
		errs := make([]error, len(p.errors))
		for i, err := range p.errors {
			errs[i] = err
		}
		panic(&sqlerrors.ParseError{Msg: sqlerrors.ConcatMessages(errs, p.maxErrors), Errors: sqlerrors.MergeErrors(p.errors)})
	}
}

func (p *Parser) Expression(instance exp.Expression) exp.Expression {
	return p.expression(instance, nil, nil)
}

func (p *Parser) expression(instance exp.Expression, token *tokens.Token, comments []string) exp.Expression {
	if instance == nil {
		return nil
	}
	if len(comments) > 0 {
		instance.AddComments(comments, false)
	} else {
		p.addComments(instance)
	}
	if !instance.IsPrimitive() {
		instance = p.validateExpression(instance, nil)
	}
	return instance
}

func (p *Parser) parseBatchStatements(parseMethod func() exp.Expression, sepFirstStatement bool) []exp.Expression {
	expressions := []exp.Expression{}
	if sepFirstStatement {
		p.match(tokens.BEGIN)
		expressions = append(expressions, parseMethod())
	}
	chunksLength := len(p.chunks)
	for p.chunkIndex < chunksLength {
		p.advanceChunk()
		expressions = append(expressions, parseMethod())
		if p.index < p.tokensSize {
			p.raiseError("Invalid expression / Unexpected token")
		}
		p.checkErrors()
	}
	return expressions
}

func (p *Parser) parse(parseMethod func() exp.Expression, rawTokens []tokens.Token, sql string) []exp.Expression {
	p.Reset()
	p.sql = sql
	total := len(rawTokens)
	chunks := [][]tokens.Token{{}}
	for i, token := range rawTokens {
		if token.TokenType == tokens.SEMICOLON {
			if len(token.Comments) > 0 {
				chunks = append(chunks, []tokens.Token{token})
			}
			if i < total-1 {
				chunks = append(chunks, []tokens.Token{})
			}
		} else {
			chunks[len(chunks)-1] = append(chunks[len(chunks)-1], token)
		}
	}
	p.chunks = chunks
	return p.parseBatchStatements(parseMethod, false)
}

func (p *Parser) parseStatement() exp.Expression {
	if !p.curr.IsValid() {
		return nil
	}
	expression := p.parseExpression()
	if expression != nil {
		expression = p.parseSetOperations(expression)
	} else {
		expression = p.parseSelect()
	}
	return p.parseQueryModifiers(expression)
}

func (p *Parser) parseSetOperations(expression exp.Expression) exp.Expression { return expression }

func (p *Parser) parseSelect() exp.Expression {
	if !p.match(tokens.SELECT) {
		return nil
	}
	comments := p.prevComments
	projections := p.parseExpressions()
	this := p.expression(exp.Select(exp.Args{"expressions": projections}), nil, nil)
	if len(comments) > 0 {
		this.AddComments(comments, true)
	}
	if from := p.parseFrom(false, false, false); from != nil {
		this.Set("from_", from)
	}
	return p.parseQueryModifiers(this)
}

func (p *Parser) parseQueryModifiers(this exp.Expression) exp.Expression {
	if this != nil && modifiableKindsLocal(this.Kind()) {
		for {
			join := p.parseJoin(false, false, nil)
			if join == nil {
				break
			}
			this.Append("joins", join)
		}
		for {
			if p.curr.TokenType == tokens.WHERE {
				where := p.parseWhere(false)
				if where != nil {
					if this.Arg("where") != nil {
						p.raiseError("Found multiple 'WHERE' clauses", p.curr)
					}
					this.Set("where", where)
					continue
				}
			}
			break
		}
	}
	return this
}

func modifiableKindsLocal(k exp.Kind) bool {
	return k == exp.KindSelect || k == exp.KindTable
}

func (p *Parser) parseFrom(joins bool, skipFromToken bool, consumePipe bool) exp.Expression {
	if !skipFromToken && !p.match(tokens.FROM) {
		return nil
	}
	comments := p.prevComments
	return p.expression(exp.From(exp.Args{"this": p.parseTable(false, joins, nil, false, false, false, consumePipe)}), nil, comments)
}

func (p *Parser) parseTable(schema bool, joins bool, aliasTokens map[tokens.TokenType]bool, parseBracket bool, isDBReference bool, parsePartition bool, consumePipe bool) exp.Expression {
	this := p.parseTableParts(schema, isDBReference, false, false)
	if this == nil {
		return nil
	}
	if alias := p.parseTableAlias(aliasTokens); alias != nil {
		this.Set("alias", alias)
	}
	if joins {
		for {
			join := p.parseJoin(false, false, aliasTokens)
			if join == nil {
				break
			}
			this.Append("joins", join)
		}
	}
	return this
}

func (p *Parser) parseTableParts(schema bool, isDBReference bool, wildcard bool, fast bool) exp.Expression {
	var catalog exp.Expression
	var db exp.Expression
	table := p.parseTablePart(schema)
	for p.match(tokens.DOT) {
		if catalog != nil {
			table = p.expression(exp.Dot(exp.Args{"this": table, "expression": p.parseTablePart(schema)}), nil, nil)
		} else {
			catalog = db
			db = table
			table = p.parseTablePart(schema)
			if table == nil {
				table = exp.Identifier(exp.Args{"this": "", "quoted": false})
			}
		}
	}
	if isDBReference {
		catalog = db
		db = table
		table = nil
	}
	if table == nil && !isDBReference {
		p.raiseError(fmt.Sprintf("Expected table name but got %s", p.curr.String()))
	}
	if db == nil && isDBReference {
		p.raiseError(fmt.Sprintf("Expected database name but got %s", p.curr.String()))
	}
	return p.expression(exp.Table(exp.Args{"this": table, "db": db, "catalog": catalog}), nil, nil)
}

func (p *Parser) parseTablePart(schema bool) exp.Expression {
	if expression := p.parseIdVar(false, nil); expression != nil {
		return expression
	}
	if expression := p.parseStringAsIdentifier(); expression != nil {
		return expression
	}
	return p.parsePlaceholder()
}

func (p *Parser) parseTableAlias(aliasTokensArg map[tokens.TokenType]bool) exp.Expression {
	anyToken := p.match(tokens.ALIAS)
	toks := aliasTokensArg
	if toks == nil {
		toks = tableAliasTokens
	}
	alias := p.parseIdVar(anyToken, toks)
	if alias == nil {
		alias = p.parseStringAsIdentifier()
	}
	if alias == nil {
		return nil
	}
	tableAlias := p.expression(exp.TableAlias(exp.Args{"this": alias}), nil, nil)
	if alias.Kind() == exp.KindIdentifier {
		tableAlias.AddComments(alias.PopComments(), false)
	}
	return tableAlias
}

var joinMethods = map[tokens.TokenType]bool{tokens.ASOF: true, tokens.NATURAL: true, tokens.POSITIONAL: true}
var joinSides = map[tokens.TokenType]bool{tokens.FULL: true, tokens.LEFT: true, tokens.RIGHT: true}
var joinKinds = map[tokens.TokenType]bool{tokens.ANTI: true, tokens.CROSS: true, tokens.INNER: true, tokens.OUTER: true, tokens.SEMI: true, tokens.STRAIGHT_JOIN: true}

func (p *Parser) parseJoinParts() (method, side, kind *tokens.Token) {
	if p.matchSet(joinMethods) {
		prev := p.prev
		method = &prev
	}
	if p.matchSet(joinSides) {
		prev := p.prev
		side = &prev
	}
	if p.matchSet(joinKinds) {
		prev := p.prev
		kind = &prev
	}
	return method, side, kind
}

func (p *Parser) parseJoin(skipJoinToken bool, parseBracket bool, aliasTokensArg map[tokens.TokenType]bool) exp.Expression {
	if p.match(tokens.COMMA) {
		table := p.tryParse(func() exp.Expression { return p.parseTable(false, false, aliasTokensArg, false, false, false, false) }, false)
		if table != nil {
			return p.expression(exp.Join(exp.Args{"this": table}), nil, nil)
		}
		return nil
	}
	index := p.index
	method, side, kind := p.parseJoinParts()
	join := p.match(tokens.JOIN) || (kind != nil && kind.TokenType == tokens.STRAIGHT_JOIN)
	joinComments := p.prevComments
	if !skipJoinToken && !join {
		p.retreat(index)
		method, side, kind = nil, nil, nil
	}
	if !skipJoinToken && !join {
		return nil
	}
	kwargs := exp.Args{"this": p.parseTable(false, false, aliasTokensArg, parseBracket, false, false, false)}
	if method != nil {
		kwargs["method"] = stringsUpper(method.Text)
	}
	if side != nil {
		kwargs["side"] = stringsUpper(side.Text)
	}
	if kind != nil {
		kwargs["kind"] = stringsUpper(kind.Text)
	}
	if p.match(tokens.ON) {
		kwargs["on"] = p.parseDisjunction()
	}
	comments := append([]string(nil), joinComments...)
	for _, token := range []*tokens.Token{method, side, kind} {
		if token != nil {
			comments = append(comments, token.Comments...)
		}
	}
	return p.expression(exp.Join(kwargs), nil, comments)
}

func (p *Parser) parseWhere(skipWhereToken bool) exp.Expression {
	if !skipWhereToken && !p.match(tokens.WHERE) {
		return nil
	}
	comments := p.prevComments
	return p.expression(exp.Where(exp.Args{"this": p.parseDisjunction()}), nil, comments)
}

func (p *Parser) parseExpression() exp.Expression { return p.parseAlias(p.parseAssignment(), false) }

func (p *Parser) parseAssignment() exp.Expression { return p.parseDisjunction() }

func (p *Parser) parseDisjunction() exp.Expression {
	this := p.parseConjunction()
	for p.match(tokens.OR) {
		comments := p.prevComments
		this = p.expression(exp.Or(exp.Args{"this": this, "expression": p.parseConjunction()}), nil, comments)
	}
	return this
}

func (p *Parser) parseConjunction() exp.Expression {
	this := p.parseEquality()
	for p.match(tokens.AND) {
		comments := p.prevComments
		this = p.expression(exp.And(exp.Args{"this": this, "expression": p.parseEquality()}), nil, comments)
	}
	return this
}

var equalityTokens = map[tokens.TokenType]func(exp.Args) exp.Expression{
	tokens.EQ:          exp.EQ,
	tokens.NEQ:         exp.NEQ,
	tokens.NULLSAFE_EQ: exp.NullSafeEQ,
}

func (p *Parser) parseEquality() exp.Expression {
	this := p.parseComparison()
	for constructor, ok := equalityTokens[p.curr.TokenType]; ok; constructor, ok = equalityTokens[p.curr.TokenType] {
		p.advance()
		comments := p.prevComments
		this = p.expression(constructor(exp.Args{"this": this, "expression": p.parseComparison()}), nil, comments)
	}
	return this
}

var comparisonTokens = map[tokens.TokenType]func(exp.Args) exp.Expression{
	tokens.GT:  exp.GT,
	tokens.GTE: exp.GTE,
	tokens.LT:  exp.LT,
	tokens.LTE: exp.LTE,
}

func (p *Parser) parseComparison() exp.Expression {
	this := p.parseRange(nil)
	for constructor, ok := comparisonTokens[p.curr.TokenType]; ok; constructor, ok = comparisonTokens[p.curr.TokenType] {
		p.advance()
		comments := p.prevComments
		this = p.expression(constructor(exp.Args{"this": this, "expression": p.parseRange(nil)}), nil, comments)
	}
	return this
}

func (p *Parser) parseRange(this exp.Expression) exp.Expression {
	if this == nil {
		this = p.parseBitwise()
	}
	negate := p.match(tokens.NOT)
	switch p.curr.TokenType {
	case tokens.BETWEEN:
		p.advance()
		low := p.parseBitwise()
		p.match(tokens.AND)
		high := p.parseBitwise()
		this = p.expression(exp.Between(exp.Args{"this": this, "low": low, "high": high}), nil, nil)
	case tokens.IN:
		p.advance()
		if p.match(tokens.L_PAREN) {
			expressions := p.parseExpressions()
			p.matchRParen(this)
			this = p.expression(exp.In(exp.Args{"this": this, "expressions": expressions}), nil, nil)
		} else {
			this = p.expression(exp.In(exp.Args{"this": this, "field": p.parseColumn()}), nil, nil)
		}
	case tokens.LIKE:
		p.advance()
		// Mirror _negate_range (parser.py): a plain LIKE has no negate key; only
		// NOT LIKE sets negate=True on the Like node (rather than wrapping in Not).
		args := exp.Args{"this": this, "expression": p.parseBitwise()}
		if negate {
			args["negate"] = true
		}
		this = p.expression(exp.Like(args), nil, nil)
		negate = false
	case tokens.ILIKE:
		p.advance()
		args := exp.Args{"this": this, "expression": p.parseBitwise()}
		if negate {
			args["negate"] = true
		}
		this = p.expression(exp.ILike(args), nil, nil)
		negate = false
	case tokens.ISNULL:
		p.advance()
		this = p.expression(exp.Is(exp.Args{"this": this, "expression": exp.Null()}), nil, nil)
	}
	if negate && p.match(tokens.NULL) {
		this = p.expression(exp.Is(exp.Args{"this": this, "expression": exp.Null()}), nil, nil)
		negate = false
	}
	if p.match(tokens.IS) {
		this = p.parseIs(this)
	}
	if negate && this != nil {
		this = p.expression(exp.Not(exp.Args{"this": this}), nil, nil)
	}
	return this
}

func (p *Parser) parseIs(this exp.Expression) exp.Expression {
	index := p.index - 1
	negate := p.match(tokens.NOT)
	expression := p.parseNull()
	if expression == nil {
		expression = p.parseBoolean()
	}
	if expression == nil {
		expression = p.parseBitwise()
	}
	if expression == nil {
		p.retreat(index)
		return nil
	}
	this = p.expression(exp.Is(exp.Args{"this": this, "expression": expression}), nil, nil)
	if negate {
		this = p.expression(exp.Not(exp.Args{"this": this}), nil, nil)
	}
	return p.parseColumnOps(this)
}

var bitwiseTokens = map[tokens.TokenType]func(exp.Args) exp.Expression{
	tokens.AMP:   exp.BitwiseAnd,
	tokens.PIPE:  exp.BitwiseOr,
	tokens.CARET: exp.BitwiseXor,
}

func (p *Parser) parseBitwise() exp.Expression {
	this := p.parseTerm()
	for {
		if constructor, ok := bitwiseTokens[p.curr.TokenType]; ok {
			p.advance()
			this = p.expression(constructor(exp.Args{"this": this, "expression": p.parseTerm()}), nil, nil)
		} else if p.dialect.DPipeIsStringConcat && p.match(tokens.DPIPE) {
			this = p.expression(exp.DPipe(exp.Args{"this": this, "expression": p.parseTerm(), "safe": !p.dialect.StrictStringConcat}), nil, nil)
		} else {
			break
		}
	}
	return this
}

var termTokens = map[tokens.TokenType]func(exp.Args) exp.Expression{
	tokens.DASH: exp.Sub,
	tokens.PLUS: exp.Add,
	tokens.MOD:  exp.Mod,
}

func (p *Parser) parseTerm() exp.Expression {
	this := p.parseFactor()
	for constructor, ok := termTokens[p.curr.TokenType]; ok; constructor, ok = termTokens[p.curr.TokenType] {
		p.advance()
		comments := p.prevComments
		this = p.expression(constructor(exp.Args{"this": this, "expression": p.parseFactor()}), nil, comments)
	}
	return this
}

var factorTokens = map[tokens.TokenType]func(exp.Args) exp.Expression{
	tokens.SLASH: exp.Div,
	tokens.STAR:  exp.Mul,
	tokens.DIV:   exp.Div,
}

func (p *Parser) parseFactor() exp.Expression {
	this := p.parseUnary()
	for constructor, ok := factorTokens[p.curr.TokenType]; ok; constructor, ok = factorTokens[p.curr.TokenType] {
		tokenType := p.curr.TokenType
		p.advance()
		comments := p.prevComments
		args := exp.Args{"this": this, "expression": p.parseUnary()}
		if tokenType == tokens.SLASH || tokenType == tokens.DIV {
			args["typed"] = p.dialect.TypedDivision
			args["safe"] = p.dialect.SafeDivision
		}
		this = p.expression(constructor(args), nil, comments)
	}
	return this
}

func (p *Parser) parseUnary() exp.Expression {
	if p.match(tokens.DASH) {
		return p.expression(exp.Neg(exp.Args{"this": p.parseUnary()}), nil, p.prevComments)
	}
	if p.match(tokens.PLUS) {
		return p.parseUnary()
	}
	if p.match(tokens.NOT) {
		// Upstream UNARY_PARSERS binds NOT to _parse_equality, not _parse_unary
		// (parser.py:1115), so NOT is lower-precedence than comparison/range ops:
		// `NOT a = b` -> Not(EQ(a, b)), `NOT a IN (..)` -> Not(In(a, ..)).
		return p.expression(exp.Not(exp.Args{"this": p.parseEquality()}), nil, p.prevComments)
	}
	return p.parseType()
}

func (p *Parser) parseType() exp.Expression {
	if atom := p.parseAtom(); atom != nil {
		return atom
	}
	return p.parseColumn()
}

func (p *Parser) parseAtom() exp.Expression {
	if identifierTokens[p.curr.TokenType] {
		if column := p.parseColumn(); column != nil {
			return column
		}
	}
	token := p.curr
	switch token.TokenType {
	case tokens.STRING, tokens.NUMBER, tokens.NULL, tokens.TRUE, tokens.FALSE, tokens.STAR:
		p.advance()
		return p.primaryFromToken(token)
	}
	return nil
}

func (p *Parser) parseColumn() exp.Expression {
	column := p.parseColumnPartsFast()
	if column == nil {
		this := p.parseColumnReference()
		if this != nil {
			column = p.parseColumnOps(this)
		}
	}
	return column
}

func (p *Parser) parseColumnPartsFast() exp.Expression {
	index := p.index
	var parts []exp.Expression
	var allComments []string
	for p.matchSet(identifierTokens) {
		token := p.prev
		comments := p.prevComments
		hasDot := p.match(tokens.DOT)
		currTT := p.curr.TokenType
		if hasDot && !identifierTokens[currTT] {
			p.retreat(index)
			return nil
		}
		if len(comments) > 0 {
			allComments = append(allComments, comments...)
			p.prevComments = nil
		}
		parts = append(parts, p.expression(exp.Identifier(exp.Args{"this": token.Text, "quoted": token.TokenType == tokens.IDENTIFIER}), &token, nil))
		if !hasDot {
			break
		}
	}
	if len(parts) == 0 {
		return nil
	}
	var column exp.Expression
	switch len(parts) {
	case 1:
		column = exp.Column(exp.Args{"this": parts[0]})
	case 2:
		column = exp.Column(exp.Args{"this": parts[1], "table": parts[0]})
	case 3:
		column = exp.Column(exp.Args{"this": parts[2], "table": parts[1], "db": parts[0]})
	default:
		column = exp.Column(exp.Args{"this": parts[3], "table": parts[2], "db": parts[1], "catalog": parts[0]})
		for _, part := range parts[4:] {
			column = exp.Dot(exp.Args{"this": column, "expression": part})
		}
	}
	if len(allComments) > 0 {
		column.AddComments(allComments, false)
	}
	return column
}

func (p *Parser) parseColumnReference() exp.Expression {
	this := p.parseField(false, nil, false)
	if this != nil && this.Kind() == exp.KindIdentifier {
		this = p.expression(exp.Column(exp.Args{"this": this}), nil, this.PopComments())
	}
	return this
}

func (p *Parser) parseColumnOps(this exp.Expression) exp.Expression {
	for p.curr.TokenType == tokens.DOT {
		p.advance()
		// Upstream _parse_column_ops takes the plain-DOT (op is None) branch here,
		// which uses _parse_field(any_token=True, anonymous_func=True). That returns a
		// bare Identifier (or Star via _parse_primary), NOT a Column-wrapped Identifier.
		// Using parseColumnReference() would wrap the field in a Column, nesting a Column
		// inside this.this/this.table and corrupting Name()/Text("table") accessors.
		field := p.parseField(true, nil, false)
		if this != nil && this.Kind() == exp.KindColumn && this.Arg("catalog") == nil {
			this = p.expression(exp.Column(exp.Args{"this": field, "table": this.This(), "db": this.Arg("table"), "catalog": this.Arg("db")}), nil, this.Comments())
		} else {
			this = p.expression(exp.Dot(exp.Args{"this": this, "expression": field}), nil, nil)
		}
		if field != nil && len(field.Comments()) > 0 {
			this.AddComments(field.PopComments(), false)
		}
	}
	return this
}

func (p *Parser) parseParen() exp.Expression {
	if !p.match(tokens.L_PAREN) {
		return nil
	}
	comments := p.prevComments
	query := p.parseSelect()
	var expressions []exp.Expression
	if query != nil {
		expressions = []exp.Expression{query}
	} else {
		expressions = p.parseExpressions()
	}
	var this exp.Expression
	if len(expressions) > 0 {
		this = expressions[0]
	}
	if this == nil && p.curr.TokenType == tokens.R_PAREN {
		this = p.expression(exp.Tuple(nil), nil, nil)
	} else if len(expressions) > 1 || p.prev.TokenType == tokens.COMMA {
		this = p.expression(exp.Tuple(exp.Args{"expressions": expressions}), nil, nil)
	} else {
		this = p.expression(exp.Paren(exp.Args{"this": this}), nil, nil)
	}
	if this != nil {
		this.AddComments(comments, false)
	}
	p.matchRParen(this)
	return this
}

func (p *Parser) parsePrimary() exp.Expression {
	token := p.curr
	switch token.TokenType {
	case tokens.STRING, tokens.NUMBER, tokens.NULL, tokens.TRUE, tokens.FALSE, tokens.STAR:
		p.advance()
		return p.primaryFromToken(token)
	}
	if p.matchPair(tokens.DOT, tokens.NUMBER, true) {
		return exp.LiteralNumber("0." + p.prev.Text)
	}
	return p.parseParen()
}

func (p *Parser) primaryFromToken(token tokens.Token) exp.Expression {
	switch token.TokenType {
	case tokens.STRING:
		return p.expression(exp.LiteralString(token.Text), &token, nil)
	case tokens.NUMBER:
		return p.expression(exp.LiteralNumber(token.Text), &token, nil)
	case tokens.NULL:
		return p.expression(exp.Null(), &token, nil)
	case tokens.TRUE:
		return p.expression(exp.Boolean(exp.Args{"this": true}), &token, nil)
	case tokens.FALSE:
		return p.expression(exp.Boolean(exp.Args{"this": false}), &token, nil)
	case tokens.STAR:
		return p.expression(exp.Star(nil), &token, nil)
	}
	return nil
}

func (p *Parser) parseField(anyToken bool, toks map[tokens.TokenType]bool, anonymousFunc bool) exp.Expression {
	field := p.parsePrimary()
	if field == nil {
		field = p.parseIdVar(anyToken, toks)
	}
	return field
}

func (p *Parser) parseAlias(this exp.Expression, explicit bool) exp.Expression {
	anyToken := p.match(tokens.ALIAS)
	comments := p.prevComments
	if explicit && !anyToken {
		return this
	}
	if p.match(tokens.L_PAREN) {
		aliases := p.expression(exp.Aliases(exp.Args{"this": this, "expressions": p.parseCsv(func() exp.Expression { return p.parseIdVar(anyToken, nil) })}), nil, comments)
		p.matchRParen(aliases)
		return aliases
	}
	alias := p.parseIdVar(anyToken, aliasTokens)
	if alias == nil {
		alias = p.parseStringAsIdentifier()
	}
	if alias != nil {
		comments = append(comments, alias.PopComments()...)
		this = p.expression(exp.AliasNode(exp.Args{"this": this, "alias": alias}), nil, comments)
	}
	return this
}

func (p *Parser) parseIdVar(anyToken bool, toks map[tokens.TokenType]bool) exp.Expression {
	expression := p.parseIdentifier()
	if expression == nil {
		if (anyToken && p.advanceAny(false) != nil) || p.matchSet(tokensOrDefault(toks, idVarTokens)) {
			quoted := p.prev.TokenType == tokens.STRING
			expression = p.identifierExpression(quoted)
		}
	}
	return expression
}

func (p *Parser) parseStringAsIdentifier() exp.Expression {
	if !p.match(tokens.STRING) {
		return nil
	}
	return exp.ToIdentifier(p.prev.Text, true)
}

func (p *Parser) parseIdentifier() exp.Expression {
	if p.match(tokens.IDENTIFIER) {
		return p.identifierExpression(true)
	}
	return p.parsePlaceholder()
}

func (p *Parser) identifierExpression(quoted bool) exp.Expression {
	return p.expression(exp.Identifier(exp.Args{"this": p.prev.Text, "quoted": quoted}), &p.prev, nil)
}

func (p *Parser) advanceAny(ignoreReserved bool) *tokens.Token {
	if p.curr.IsValid() && (ignoreReserved || !reservedTokens[p.curr.TokenType]) {
		p.advance()
		return &p.prev
	}
	return nil
}

func (p *Parser) parseNull() exp.Expression {
	if p.curr.TokenType == tokens.NULL || p.curr.TokenType == tokens.UNKNOWN {
		token := p.curr
		p.advance()
		return p.primaryFromToken(token)
	}
	return p.parsePlaceholder()
}

func (p *Parser) parseBoolean() exp.Expression {
	if p.curr.TokenType == tokens.TRUE || p.curr.TokenType == tokens.FALSE {
		token := p.curr
		p.advance()
		return p.primaryFromToken(token)
	}
	return p.parsePlaceholder()
}

func (p *Parser) parseStar() exp.Expression {
	if p.curr.TokenType == tokens.STAR {
		token := p.curr
		p.advance()
		return p.primaryFromToken(token)
	}
	return p.parsePlaceholder()
}

func (p *Parser) parsePlaceholder() exp.Expression {
	if p.match(tokens.PLACEHOLDER) {
		return p.expression(exp.Placeholder(nil), &p.prev, nil)
	}
	return nil
}

func (p *Parser) parseCsv(parseMethod func() exp.Expression, sep ...tokens.TokenType) []exp.Expression {
	separator := tokens.COMMA
	if len(sep) > 0 {
		separator = sep[0]
	}
	parseResult := parseMethod()
	items := []exp.Expression{}
	if parseResult != nil {
		items = append(items, parseResult)
	}
	for p.match(separator) {
		if parseResult != nil {
			p.addComments(parseResult)
		}
		parseResult = parseMethod()
		if parseResult != nil {
			items = append(items, parseResult)
		}
	}
	return items
}

func (p *Parser) parseExpressions() []exp.Expression { return p.parseCsv(p.parseExpression) }

func (p *Parser) matchLParen(expression exp.Expression) {
	if !p.match(tokens.L_PAREN) {
		p.raiseError("Expecting (")
	}
}

func (p *Parser) matchRParen(expression exp.Expression) {
	if !p.match(tokens.R_PAREN) {
		p.raiseError("Expecting )")
	}
}

func firstExpression(expressions ...exp.Expression) exp.Expression {
	for _, expression := range expressions {
		if expression != nil {
			return expression
		}
	}
	return nil
}

func tokensOrDefault(value map[tokens.TokenType]bool, defaultValue map[tokens.TokenType]bool) map[tokens.TokenType]bool {
	if value != nil {
		return value
	}
	return defaultValue
}

func stringsUpper(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'a' && r <= 'z' {
			out[i] = r - ('a' - 'A')
		}
	}
	return string(out)
}
