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
	var begin any
	var properties []exp.Expression

	switch {
	case dbCreatables[ctt]:
		tableParts := p.parseTableParts(true, ctt == tokens.SCHEMA, false, false)
		p.match(tokens.COMMA)
		this = p.parseSchema(tableParts)
		// exp.Properties.Location.POST_SCHEMA/POST_WITH (parser.py:2444): the table-tail
		// property list, e.g. mysql's `DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC`.
		properties = append(properties, p.parseTailProperties()...)
		hasAlias := p.match(tokens.ALIAS)
		expression = p.parseDDLSelect()
		if expression == nil && hasAlias {
			expression = p.tryParse(func() exp.Expression { return p.parseTableParts(false, false, false, false) }, false)
		}
		if ctt == tokens.VIEW && p.matchTextSeq("WITH", "NO", "SCHEMA", "BINDING") {
			noSchemaBinding = true
		}
	case ctt == tokens.FUNCTION || ctt == tokens.PROCEDURE:
		this, expression, properties, begin = p.parseCreateFunction(ctt)
	case ctt == tokens.INDEX:
		var index exp.Expression
		anonymous := false
		if !p.match(tokens.ON) {
			index = p.parseIdVar(true, nil)
		} else {
			anonymous = true
		}
		if index == nil && !anonymous {
			return nil
		}
		this = p.parseIndexBody(index, anonymous)
	case ctt == tokens.TRIGGER || (ctt == tokens.CONSTRAINT && p.match(tokens.TRIGGER)):
		isConstraint := ctt == tokens.CONSTRAINT
		if isConstraint {
			createToken = p.prev
		}
		var triggerProps exp.Expression
		this, triggerProps = p.parseCreateTrigger(isConstraint)
		if this == nil {
			return nil
		}
		properties = append(properties, triggerProps)
	default:
		return nil
	}
	if p.curr.IsValid() && !p.matchSet(rParenCommaSet, false) {
		return nil
	}
	kind := stringsUpper(createToken.Text)
	var propertiesNode exp.Expression
	if len(properties) > 0 {
		propertiesNode = p.expression(exp.Properties(exp.Args{"expressions": properties}), nil, nil)
	}
	return p.expression(exp.Create(exp.Args{
		"this":              this,
		"kind":              kind,
		"replace":           replace,
		"refresh":           refresh,
		"unique":            unique,
		"expression":        expression,
		"exists":            exists,
		"properties":        propertiesNode,
		"no_schema_binding": noSchemaBinding,
		"concurrently":      concurrently,
		"begin":             begin,
	}), nil, nil)
}

// parseUserDefinedFunction ports _parse_user_defined_function (parser.py:7114-7124): a
// CREATE FUNCTION/PROCEDURE signature, `name` or `name(params...)`.
func (p *Parser) parseUserDefinedFunction() exp.Expression {
	this := p.parseTableParts(true, false, false, false)
	if !p.match(tokens.L_PAREN) {
		return this
	}
	expressions := p.parseCsv(p.parseFunctionParameter)
	p.matchRParen(nil)
	return p.expression(exp.UserDefinedFunction(exp.Args{"this": this, "expressions": expressions, "wrapped": true}), nil, nil)
}

// parseFunctionParameter ports the base _parse_function_parameter (parser.py:7111-7112):
// `<id> <type> [constraints...]`. Postgres's own override (parsers/postgres.py:275-292)
// additionally disambiguates a leading IN/OUT/INOUT/VARIADIC parameter-mode keyword via a
// two-token type-lookahead (-> exp.InOutColumnConstraint); that disambiguation isn't ported
// (documented divergence). It's still safe to skip: since this base path's parseIdVar
// (any_token=true) greedily consumes the mode keyword itself as the parameter's `this`,
// leaving its real name/type unconsumed, parseUserDefinedFunction's matchRParen fails (or,
// for the sole "MODE TYPE" two-token shape, happens to render byte-identically regardless of
// which interpretation is taken) - either way the CREATE degrades to a Command exactly as it
// did before this cluster existed, so no in-scope corpus case regresses.
func (p *Parser) parseFunctionParameter() exp.Expression {
	return p.parseColumnDef(p.parseIdVar(true, nil))
}

// parseCreateFunction ports the CREATE FUNCTION/PROCEDURE branch of _parse_create
// (parser.py:2414-2472): the UDF signature, its RETURNS/LANGUAGE/... properties (both
// before and after the body), and the body itself (a string literal, a nested statement, or
// - left unsupported, see parseUserDefinedFunction - a heredoc/BEGIN block).
func (p *Parser) parseCreateFunction(ctt tokens.TokenType) (this, expression exp.Expression, properties []exp.Expression, begin any) {
	this = p.parseUserDefinedFunction()
	// exp.Properties.Location.POST_SCHEMA (parser.py:2417): properties parsed before AS.
	properties = append(properties, p.parseTailProperties()...)

	// _parse_heredoc (parser.py:9368-9370) only matches TokenType.HEREDOC_STRING - a
	// dollar-quoted UDF body (e.g. postgres `AS $$ ... $$`), which this port doesn't model
	// (no exp.Heredoc Kind - deferred, no target-gap SQL needs it). `AS` is still matched
	// here (mirroring upstream's self._match(TokenType.ALIAS) gate); a genuine heredoc body
	// is left unconsumed below (bailing out before the generic parseStatement fallback, which
	// - unlike upstream's own HEREDOC_STRING-only _parse_heredoc - would otherwise happily
	// misparse it as an ordinary string literal), so the caller's trailing-token check
	// degrades the whole CREATE to a Command.
	p.match(tokens.ALIAS)
	if p.curr.TokenType == tokens.HEREDOC_STRING {
		return this, nil, properties, begin
	}

	// upstream's table-overload/MacroOverloads detection (parser.py:2422-2436) is a
	// BigQuery/Snowflake-only feature (`CREATE FUNCTION f(...) AS (expr), (params) AS TABLE
	// body`) that always retreats to a no-op for base/mysql/postgres inputs: it only commits
	// when the parsed body is immediately followed by `, (`, which never occurs in this
	// port's corpus. Omitted (documented divergence; see parser_ddl_tail_test.go).

	// exp.Properties.Location.POST_SCHEMA (parser.py:2445): properties parsed after AS/the
	// overload attempt (e.g. mysql's LANGUAGE/SQL SECURITY tail when there's no AS at all).
	properties = append(properties, p.parseTailProperties()...)

	if p.match(tokens.COMMAND) {
		expression = p.parseAsCommand(p.prev)
		return this, expression, properties, begin
	}
	begin = p.match(tokens.BEGIN)
	returnMatched := p.matchTextSeq("RETURN")
	if p.match(tokens.STRING, false) {
		expression = p.parseString()
		// exp.Properties.Location.POST_SCHEMA (parser.py:2450): properties parsed after a
		// string-literal body, e.g. postgres's trailing LANGUAGE/IMMUTABLE/CALLED ON NULL
		// INPUT.
		properties = append(properties, p.parseTailProperties()...)
	} else if ctt == tokens.FUNCTION {
		// _parse_user_defined_function_expression (parser.py:7108-7109) is just
		// self._parse_statement(); exp.Block (the PROCEDURE body fallback, parser.py:2462)
		// isn't ported (deferred - no target-gap SQL needs it), so a PROCEDURE with a
		// non-string body simply leaves `expression` nil here, degrading to a Command via
		// the caller's trailing-token check.
		expression = p.parseStatement()
	}
	if returnMatched {
		expression = p.expression(exp.Return(exp.Args{"this": expression}), nil, nil)
	}
	return this, expression, properties, begin
}

// parseIndexBody ports the `if index or anonymous:` half of _parse_index
// (parser.py:4606-4634) - the only half reachable from CREATE INDEX (the other half, used
// only by a bare _parse_index() call from CREATE TABLE's own post-schema index-collection
// loop, isn't ported: this slice doesn't populate exp.Create's "indexes" list).
func (p *Parser) parseIndexBody(index exp.Expression, anonymous bool) exp.Expression {
	p.match(tokens.ON)
	p.match(tokens.TABLE) // hive
	table := p.parseTableParts(true, false, false, false)
	params := p.parseIndexParams()
	return p.expression(exp.Index(exp.Args{
		"this": index, "table": table, "params": params,
	}), nil, nil)
}

// triggerEventTokens mirrors TRIGGER_EVENTS (parser.py:650-655).
var triggerEventTokens = map[tokens.TokenType]bool{
	tokens.INSERT: true, tokens.UPDATE: true, tokens.DELETE: true, tokens.TRUNCATE: true,
}

// triggerTimingOptions mirrors TRIGGER_TIMING (parser.py:1588-1592).
var triggerTimingOptions = optionsType{
	"INSTEAD": {{"OF"}},
	"BEFORE":  nil,
	"AFTER":   nil,
}

// triggerDeferrableOptions mirrors TRIGGER_DEFERRABLE (parser.py:1594-1597).
var triggerDeferrableOptions = optionsType{
	"NOT":        {{"DEFERRABLE"}},
	"DEFERRABLE": nil,
}

// parseCreateTrigger ports the (CONSTRAINT )?TRIGGER branch of _parse_create
// (parser.py:2483-2532): trigger name, timing, events, ON table, [FROM referenced_table],
// [DEFERRABLE ...], [REFERENCING ...], [FOR EACH ROW|STATEMENT], [WHEN (...)], EXECUTE
// FUNCTION|PROCEDURE <call>. Returns (nil, nil) at any of upstream's own
// `return self._parse_as_command(start)` bail points, which the caller (case
// ctt==TRIGGER... in parseCreateStructured) turns into an overall nil (-> Command).
func (p *Parser) parseCreateTrigger(isConstraint bool) (exp.Expression, exp.Expression) {
	triggerName := p.parseIdVar(true, nil)
	if triggerName == nil {
		return nil, nil
	}
	timingVar := p.parseVarFromOptions(triggerTimingOptions, false)
	if timingVar == nil {
		return nil, nil
	}
	timing := timingVar.Name()
	events := p.parseTriggerEvents()
	if !p.match(tokens.ON) {
		p.raiseError("Expected ON in trigger definition")
	}
	table := p.parseTableParts(false, false, false, false)
	var referencedTable exp.Expression
	if p.match(tokens.FROM) {
		referencedTable = p.parseTableParts(false, false, false, false)
	}
	deferrable, initially := p.parseTriggerDeferrable()
	referencing := p.parseTriggerReferencing()
	forEach := p.parseTriggerForEach()
	var when exp.Expression
	if p.matchTextSeq("WHEN") {
		when = p.parseWrapped(p.parseDisjunction, true)
	}
	execute := p.parseTriggerExecute()
	if execute == nil {
		return nil, nil
	}
	triggerProps := p.expression(exp.TriggerProperties(exp.Args{
		"table":            table,
		"timing":           timing,
		"events":           events,
		"execute":          execute,
		"constraint":       isConstraint,
		"referenced_table": referencedTable,
		"deferrable":       deferrable,
		"initially":        initially,
		"referencing":      referencing,
		"for_each":         forEach,
		"when":             when,
	}), nil, nil)
	return triggerName, triggerProps
}

// parseTriggerEvents ports _parse_trigger_events (parser.py:2681-2701).
func (p *Parser) parseTriggerEvents() []exp.Expression {
	var events []exp.Expression
	for {
		if !p.matchSet(triggerEventTokens) {
			p.raiseError("Expected trigger event (INSERT, UPDATE, DELETE, TRUNCATE)")
		}
		eventType := stringsUpper(p.prev.Text)
		var columns []exp.Expression
		if eventType == "UPDATE" && p.matchTextSeq("OF") {
			columns = p.parseCsv(p.parseColumn)
		}
		events = append(events, p.expression(exp.TriggerEvent(exp.Args{"this": eventType, "columns": columns}), nil, nil))
		if !p.match(tokens.OR) {
			break
		}
	}
	return events
}

// parseTriggerDeferrable ports _parse_trigger_deferrable (parser.py:2703-2717).
func (p *Parser) parseTriggerDeferrable() (any, any) {
	var deferrable any
	if deferrableVar := p.parseVarFromOptions(triggerDeferrableOptions, false); deferrableVar != nil {
		deferrable = deferrableVar.Name()
	}
	var initially any
	if deferrable != nil && p.matchTextSeq("INITIALLY") {
		if p.matchTexts(map[string]bool{"IMMEDIATE": true, "DEFERRED": true}) {
			initially = stringsUpper(p.prev.Text)
		}
	}
	return deferrable, initially
}

// parseTriggerReferencingClause ports _parse_trigger_referencing_clause (parser.py:2719-2725).
func (p *Parser) parseTriggerReferencingClause(keyword string) exp.Expression {
	if !p.matchTextSeq(keyword) {
		return nil
	}
	if !p.matchTextSeq("TABLE") {
		p.raiseError("Expected TABLE after " + keyword + " in REFERENCING clause")
	}
	p.matchTextSeq("AS")
	return p.parseIdVar(true, nil)
}

// parseTriggerReferencing ports _parse_trigger_referencing (parser.py:2727-2749).
func (p *Parser) parseTriggerReferencing() exp.Expression {
	if !p.matchTextSeq("REFERENCING") {
		return nil
	}
	var oldAlias, newAlias exp.Expression
	for {
		if alias := p.parseTriggerReferencingClause("OLD"); alias != nil {
			if oldAlias != nil {
				p.raiseError("Duplicate OLD clause in REFERENCING")
			}
			oldAlias = alias
		} else if alias := p.parseTriggerReferencingClause("NEW"); alias != nil {
			if newAlias != nil {
				p.raiseError("Duplicate NEW clause in REFERENCING")
			}
			newAlias = alias
		} else {
			break
		}
	}
	if oldAlias == nil && newAlias == nil {
		p.raiseError("REFERENCING clause requires at least OLD TABLE or NEW TABLE")
	}
	return p.expression(exp.TriggerReferencing(exp.Args{"old": oldAlias, "new": newAlias}), nil, nil)
}

// parseTriggerForEach ports _parse_trigger_for_each (parser.py:2751-2755).
func (p *Parser) parseTriggerForEach() any {
	if !p.matchTextSeq("FOR", "EACH") {
		return nil
	}
	if p.matchTexts(map[string]bool{"ROW": true, "STATEMENT": true}) {
		return stringsUpper(p.prev.Text)
	}
	return nil
}

// parseTriggerExecute ports _parse_trigger_execute (parser.py:2757-2765).
func (p *Parser) parseTriggerExecute() exp.Expression {
	if !p.match(tokens.EXECUTE) {
		return nil
	}
	if !p.matchSet(map[tokens.TokenType]bool{tokens.FUNCTION: true, tokens.PROCEDURE: true}) {
		p.raiseError("Expected FUNCTION or PROCEDURE after EXECUTE")
	}
	funcCall := p.parseColumn()
	return p.expression(exp.TriggerExecute(exp.Args{"this": funcCall}), nil, nil)
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

	// `<name> FOR ORDINALITY` (parser.py:7257): an XMLTABLE/column-def ordinality marker. The
	// flag isn't rendered by columndef_sql (generator.py:1169), so it round-trips lossily to just
	// `<name>`, matching upstream.
	if p.matchPair(tokens.FOR, tokens.ORDINALITY, true) {
		return p.expression(exp.ColumnDef(exp.Args{"this": this, "ordinality": true}), nil, nil)
	}

	constraints := []exp.Expression{}

	// `<kind> AS (<expr>) [STORED|VIRTUAL]`: the computed-column branch at parser.py:
	// 7283-7302. Gated on WRAPPED_TRANSFORM_COLUMN_CONSTRAINT (=True for base/mysql/
	// postgres; only RisingWave, out of scope, overrides False), which requires the AS to
	// be immediately followed by "(" - so `col INT AS some_expr` (no parens) is left for
	// parseColumnConstraint's own dispatch instead. The two upstream sibling branches
	// (ALIAS/MATERIALIZED-prefixed and IN/OUT parameter constraints, parser.py:7262-7281)
	// are Oracle/T-SQL/procedure-only and aren't exercised by the base/mysql/postgres
	// corpus, so they're omitted here (documented divergence).
	if kind != nil && p.matchPair(tokens.ALIAS, tokens.L_PAREN, false) {
		p.advance() // consume AS, leaving `(` as the current token for parseDisjunction below
		// `this` MUST be parsed before the STORED/VIRTUAL match: upstream (parser.py:7295-7299)
		// evaluates this=self._parse_disjunction() before persisted=self._match_texts(...), and
		// parseDisjunction consumes the `(<expr>)` so the storage keyword (if present) becomes the
		// current token. Computing `persisted` first (Go arg-map evaluation order) would test the
		// still-current `(` and leave STORED/VIRTUAL unconsumed, degrading the CREATE to a Command.
		this := p.parseDisjunction()
		persisted := p.matchTexts(map[string]bool{"STORED": true, "VIRTUAL": true}) && stringsUpper(p.prev.Text) == "STORED"
		constraints = append(constraints, p.expression(exp.ColumnConstraint(exp.Args{
			"kind": p.expression(exp.ComputedColumnConstraint(exp.Args{
				"this":      this,
				"persisted": persisted,
			}), nil, nil),
		}), nil, nil))
	}

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

	// Trailing FIRST/AFTER <col> position (parser.py:7313-7316), e.g. `ADD COLUMN k INT
	// FIRST` / `ADD COLUMN k INT AFTER m`.
	var position exp.Expression
	if p.matchTexts(map[string]bool{"FIRST": true, "AFTER": true}) {
		// pos must be captured before parseColumn() below, which advances p.prev past the
		// FIRST/AFTER keyword itself (mirrors upstream's `pos = self._prev.text` capture
		// preceding its own _parse_column() call, parser.py:7314-7316).
		pos := p.prev.Text
		position = p.expression(exp.ColumnPosition(exp.Args{"this": p.parseColumn(), "position": pos}), nil, nil)
	}

	return p.expression(exp.ColumnDef(exp.Args{"this": this, "kind": kind, "constraints": constraints, "position": position}), nil, nil)
}

// parseConstraint ports _parse_constraint (parser.py:7462-7468): a named `CONSTRAINT <id>
// (<unnamed constraints>)` clause, or (when CONSTRAINT isn't present) an unnamed
// schema-level constraint drawn from schemaUnnamedConstraints (CHECK/FOREIGN KEY/PRIMARY
// KEY/UNIQUE, +mysql's FULLTEXT/INDEX/KEY/SPATIAL).
func (p *Parser) parseConstraint() exp.Expression {
	if !p.match(tokens.CONSTRAINT) {
		return p.parseUnnamedConstraint(p.schemaUnnamedConstraints())
	}
	return p.expression(exp.Constraint(exp.Args{
		"this":        p.parseIdVar(true, nil),
		"expressions": p.parseUnnamedConstraints(),
	}), nil, nil)
}

// parseUnnamedConstraints ports _parse_unnamed_constraints (parser.py:7470-7478): the body
// of a named CONSTRAINT, a CSV of unnamed constraints (matched against every registered
// CONSTRAINT_PARSERS key, not just the schema-level subset) or plain function calls.
func (p *Parser) parseUnnamedConstraints() []exp.Expression {
	var constraints []exp.Expression
	for {
		constraint := p.parseUnnamedConstraint(nil)
		if constraint == nil {
			constraint = p.parseFunction(nil, false, true, false)
		}
		if constraint == nil {
			break
		}
		constraints = append(constraints, constraint)
	}
	return constraints
}

// parseUnnamedConstraint ports _parse_unnamed_constraint (parser.py:7480-7498). constraints
// filters which texts are eligible (nil means "any CONSTRAINT_PARSERS key", mirroring
// `constraints or self.CONSTRAINT_PARSERS`).
func (p *Parser) parseUnnamedConstraint(constraints map[string]bool) exp.Expression {
	index := p.index
	keys := constraints
	if keys == nil {
		keys = p.constraintParserKeys()
	}
	if p.match(tokens.IDENTIFIER, false) || !p.matchTexts(keys) {
		return nil
	}
	key := stringsUpper(p.prev.Text)
	fn, ok := p.constraintParsers()[key]
	if !ok {
		p.raiseError("No parser found for schema constraint " + key + ".")
		return nil
	}
	result := fn(p)
	if result == nil {
		p.retreat(index)
	}
	return result
}

// parseColumnConstraint ports _parse_column_constraint (parser.py:7443-7460): an optional
// `CONSTRAINT <id>` name, followed by a CONSTRAINT_PARSERS-dispatched kind.
// PROCEDURE_OPTIONS (parser.py:1636) is empty for base/mysql/postgres (only T-SQL overrides
// it), so procedure_option_follows is always false and is omitted here.
func (p *Parser) parseColumnConstraint() exp.Expression {
	var this exp.Expression
	if p.match(tokens.CONSTRAINT) {
		this = p.parseIdVar(true, nil)
	}
	if p.matchTexts(p.constraintParserKeys()) {
		key := stringsUpper(p.prev.Text)
		var constraint exp.Expression
		if fn := p.constraintParsers()[key]; fn != nil {
			constraint = fn(p)
		}
		if constraint == nil {
			p.retreat(p.index - 1)
			return nil
		}
		return p.expression(exp.ColumnConstraint(exp.Args{"this": this, "kind": constraint}), nil, nil)
	}
	return this
}

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

// propertyParserFunc mirrors one PROPERTY_PARSERS entry (parser.py:1227-1341): `default`
// carries the `_match(DEFAULT) and ...` prefix upstream passes as a `default=True` kwarg
// (only the CHARACTER SET/CHARSET entries consult it - parser.py:1239-1240).
type propertyParserFunc func(p *Parser, isDefault bool) exp.Expression

// propertyParsers/postgresPropertyParsers are a restricted PROPERTY_PARSERS
// (parser.py:1227-1341): only the keys needed to close this slice's DDL-tail parity gaps
// (RETURNS/LANGUAGE/SECURITY/SQL SECURITY/CALLED/STRICT/IMMUTABLE/STABLE/DETERMINISTIC/
// CHARSET/CHARACTER SET/ROW_FORMAT, +postgres's SET override). The remaining ~90 upstream
// entries (ENGINE/AUTO_INCREMENT/COLLATE/COMMENT/PARTITION BY/WITH/TBLPROPERTIES/... and
// every dialect-specific storage/model property) are exotic and out of scope for this port
// (documented divergence, mirroring constraintParsers' analogous omission list in
// parser_constraints.go); a property list this restricted parser can't recognize simply
// stops - it never consumes a token it shouldn't, so an unsupported trailing property always
// degrades the enclosing CREATE to a Command exactly as it did before this cluster existed.
var (
	propertyParsers         map[string]propertyParserFunc
	propertyParserKeySet    map[string]bool
	postgresPropertyParsers map[string]propertyParserFunc
	postgresPropertyKeySet  map[string]bool
)

func init() {
	propertyParsers = map[string]propertyParserFunc{
		"RETURNS": func(p *Parser, _ bool) exp.Expression { return p.parseReturnsProperty() },
		"LANGUAGE": func(p *Parser, _ bool) exp.Expression {
			return p.parsePropertyAssignment(func(this exp.Expression) exp.Expression {
				return p.expression(exp.LanguageProperty(exp.Args{"this": this}), nil, nil)
			})
		},
		"SECURITY":     func(p *Parser, _ bool) exp.Expression { return p.parseSQLSecurity() },
		"SQL SECURITY": func(p *Parser, _ bool) exp.Expression { return p.parseSQLSecurity() },
		"CALLED":       func(p *Parser, _ bool) exp.Expression { return p.parseCalledOnNullInputProperty() },
		"STRICT": func(p *Parser, _ bool) exp.Expression {
			return p.expression(exp.StrictProperty(nil), nil, nil)
		},
		"IMMUTABLE":     func(p *Parser, _ bool) exp.Expression { return p.parseStabilityProperty("IMMUTABLE") },
		"STABLE":        func(p *Parser, _ bool) exp.Expression { return p.parseStabilityProperty("STABLE") },
		"DETERMINISTIC": func(p *Parser, _ bool) exp.Expression { return p.parseStabilityProperty("IMMUTABLE") },
		"CHARSET":       func(p *Parser, isDefault bool) exp.Expression { return p.parseCharacterSet(isDefault) },
		"CHARACTER SET": func(p *Parser, isDefault bool) exp.Expression { return p.parseCharacterSet(isDefault) },
		"ROW_FORMAT": func(p *Parser, _ bool) exp.Expression {
			return p.parsePropertyAssignment(func(this exp.Expression) exp.Expression {
				return p.expression(exp.RowFormatProperty(exp.Args{"this": this}), nil, nil)
			})
		},
	}
	propertyParserKeySet = funcMapKeys2(propertyParsers)

	// postgresPropertyParsers ports parsers/postgres.py:89-91 PostgresParser.PROPERTY_PARSERS:
	// `**parser.Parser.PROPERTY_PARSERS` (minus INPUT, not ported anyway) plus a "SET"
	// override -> exp.SetConfigProperty (`SET <config_param> {TO|=} <value>`). This reuses
	// the top-level parseSet (parser/stmt_set.go) rather than a bespoke CSV-of-parseSetItem
	// helper: upstream's own `_parse_set()` (parser.py:9265-9275, called here with its
	// defaults) degrades to a raw Command whenever anything follows the SET-item list that
	// isn't itself a further SET item - swallowing the *rest of the whole statement*
	// (parser.py's own `self._parse_as_command(start)` has no notion of "just this
	// property"'s scope). That's not a bug specific to being invoked mid-CREATE-FUNCTION: it's
	// upstream's real behavior (verified against the pinned oracle for
	// `SET search_path TO 'public' AS 'select $1 + $2;' LANGUAGE SQL IMMUTABLE`, which
	// produces SetConfigProperty(this=Command(this=SET, expression="search_path TO 'public'
	// AS ... IMMUTABLE"))), so parseSet's identical "SET " + Command(rest) fallback is exactly
	// the 1:1 port, not an approximation.
	postgresPropertyParsers = make(map[string]propertyParserFunc, len(propertyParsers)+1)
	for k, v := range propertyParsers {
		postgresPropertyParsers[k] = v
	}
	postgresPropertyParsers["SET"] = func(p *Parser, _ bool) exp.Expression {
		return p.expression(exp.SetConfigProperty(exp.Args{"this": p.parseSet()}), nil, nil)
	}
	postgresPropertyKeySet = funcMapKeys2(postgresPropertyParsers)
}

func funcMapKeys2(m map[string]propertyParserFunc) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func (p *Parser) propertyParsersFor() map[string]propertyParserFunc {
	if p.dialect.Name == "postgres" {
		return postgresPropertyParsers
	}
	return propertyParsers
}

func (p *Parser) propertyParserKeys() map[string]bool {
	if p.dialect.Name == "postgres" {
		return postgresPropertyKeySet
	}
	return propertyParserKeySet
}

// parseTailProperty ports _parse_property (parser.py:2793-2811) minus its generic key=value
// fallback (_parse_key_value_property) and sequence-properties branch: neither is reachable
// through this restricted propertyParsers set, so a non-matching leading token simply yields
// nil (mirrors _parse_function_properties' analogous restriction, parser.py:7091-7095).
func (p *Parser) parseTailProperty() exp.Expression {
	if p.matchTexts(p.propertyParserKeys()) {
		return p.propertyParsersFor()[stringsUpper(p.prev.Text)](p, false)
	}
	if p.match(tokens.DEFAULT) && p.matchTexts(p.propertyParserKeys()) {
		return p.propertyParsersFor()[stringsUpper(p.prev.Text)](p, true)
	}
	return nil
}

// parseTailProperties ports _parse_properties (parser.py:2767-2781) with `before` always false
// (the `before=True` POST_NAME variant, teradata-only, is out of scope). Returns the raw
// property list rather than wrapping it in exp.Properties, so callers can accumulate
// multiple passes (before/after AS, ...) before building one Properties node.
func (p *Parser) parseTailProperties() []exp.Expression {
	var properties []exp.Expression
	for {
		prop := p.parseTailProperty()
		if prop == nil {
			break
		}
		properties = append(properties, prop)
	}
	return properties
}

// parsePropertyAssignment ports _parse_property_assignment (parser.py:2873-2877): `[=] [AS]
// <unquoted field>`.
func (p *Parser) parsePropertyAssignment(build func(this exp.Expression) exp.Expression) exp.Expression {
	p.match(tokens.EQ)
	p.match(tokens.ALIAS)
	return build(p.parseUnquotedField())
}

// parseCharacterSet ports _parse_character_set (parser.py:3382-3386).
func (p *Parser) parseCharacterSet(isDefault bool) exp.Expression {
	p.match(tokens.EQ)
	return p.expression(exp.CharacterSetProperty(exp.Args{"this": p.parseVarOrString(false), "default": isDefault}), nil, nil)
}

// parseSQLSecurity ports _parse_sql_security (parser.py:2901-2905).
func (p *Parser) parseSQLSecurity() exp.Expression {
	var this any
	if p.matchTexts(securityPropertyKeywords) {
		this = stringsUpper(p.prev.Text)
	}
	return p.expression(exp.SqlSecurityProperty(exp.Args{"this": this}), nil, nil)
}

// securityPropertyKeywords mirrors SECURITY_PROPERTY_KEYWORDS (parser.py:1751).
var securityPropertyKeywords = map[string]bool{"DEFINER": true, "INVOKER": true, "NONE": true}

// parseCalledOnNullInputProperty ports _parse_called_on_null_input_property
// (parser.py:2913-2918). On failure it retreats past the already-consumed "CALLED" keyword
// too (self._retreat(self._index - 1)), matching upstream exactly.
func (p *Parser) parseCalledOnNullInputProperty() exp.Expression {
	if !p.matchTextSeq("ON", "NULL", "INPUT") {
		p.retreat(p.index - 1)
		return nil
	}
	return p.expression(exp.CalledOnNullInputProperty(nil), nil, nil)
}

// parseStabilityProperty builds the exp.StabilityProperty(this=Literal.string(name)) node
// shared by the IMMUTABLE/STABLE/DETERMINISTIC PROPERTY_PARSERS entries (parser.py:1253-1255,
// 1275-1277,1323-1325).
func (p *Parser) parseStabilityProperty(name string) exp.Expression {
	return p.expression(exp.StabilityProperty(exp.Args{"this": exp.LiteralString(name)}), nil, nil)
}

// parseReturnsProperty ports _parse_returns (parser.py:3394-3414), minus its `RETURNS
// TABLE(...)` branch (parser.py:3397-3407, a BigQuery/DuckDB-oriented table-valued-function
// signature - deferred, no target-gap SQL uses it: a bare TABLE token here is simply left
// unconsumed, degrading the enclosing CREATE to a Command).
func (p *Parser) parseReturnsProperty() exp.Expression {
	var value exp.Expression
	var null any
	if p.matchTextSeq("NULL", "ON", "NULL", "INPUT") {
		null = true
	} else {
		value = p.parseTypes(false, false, true, false)
	}
	return p.expression(exp.ReturnsProperty(exp.Args{"this": value, "is_table": false, "null": null}), nil, nil)
}
