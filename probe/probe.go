package probe

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	sqlglot "github.com/sjincho/sqlglot-go"
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/generator"
	"github.com/sjincho/sqlglot-go/optimizer"
	"github.com/sjincho/sqlglot-go/schema"
)

const (
	PREDICATE = "PREDICATE"
	JOIN      = "JOIN"
	GROUP_BY  = "GROUP_BY"
	ORDER_BY  = "ORDER_BY"
	AGGREGATE = "AGGREGATE"
	SUBQUERY  = "SUBQUERY"
	SET_OP    = "SET_OP"
	RECURSIVE = "RECURSIVE"
	DERIVED   = "DERIVED"
	OTHER     = "OTHER"
)

var dangerousFuncs = map[string]bool{
	"dblink": true, "dblink_exec": true, "dblink_open": true, "dblink_fetch": true, "dblink_send_query": true,
	"pg_read_file": true, "pg_read_binary_file": true, "pg_ls_dir": true, "pg_stat_file": true,
	"lo_import": true, "lo_export": true, "load_file": true,
	"query_to_xml": true, "query_to_xml_and_xmlschema": true, "xpath_table": true,
}

type OriginInfo struct {
	Column  string   `json:"column"`
	Origins []string `json:"origins"`
}

type ProbeResult struct {
	Resolved      bool                `json:"resolved"`
	FailedStage   *string             `json:"failedStage"`
	Detail        string              `json:"detail"`
	OutputColumns int                 `json:"outputColumns"`
	TracedColumns int                 `json:"tracedColumns"`
	Origins       []OriginInfo        `json:"origins"`
	References    map[string][]string `json:"references"`
	IsWrite       bool                `json:"isWrite"`
	RewrittenSQL  *string             `json:"rewrittenSql"`
}

type prober struct {
	dialect string

	root         exp.Expression
	qroot        exp.Expression
	analyzeQuery exp.Expression
	writeTarget  exp.Expression
	payloadExprs []exp.Expression
	payloadQuery exp.Expression
	isWrite      bool

	scopes        []*optimizer.Scope
	col2scope     map[exp.Expression]*optimizer.Scope
	scopeOfSelect map[exp.Expression]*optimizer.Scope
	writeScope    *optimizer.Scope

	references    map[string]map[string]bool
	opaqueSelects map[exp.Expression]bool

	loweredMapping *schema.Mapping
	schemaCols     map[string][]string
	schemaCI       map[string]map[string]bool
	schemaTables   map[string]bool
	cteNames       map[string]bool
}

type resolveKey struct {
	scope *optimizer.Scope
	alias string
	name  string
}

type identKey struct {
	scope *optimizer.Scope
	alias string
	name  string
}

type writeSource struct {
	kind  string
	name  string
	query exp.Expression
}

func Probe(sql, dialect string, sch *schema.Mapping) ProbeResult {
	p := &prober{
		dialect:       strings.ToLower(dialect),
		references:    map[string]map[string]bool{},
		opaqueSelects: map[exp.Expression]bool{},
	}
	p.loweredMapping, p.schemaCols, p.schemaCI, p.schemaTables = lowerSchema(sch)

	var parsed []exp.Expression
	if fail := runStage("PARSE", func() {
		var err error
		parsed, err = sqlglot.Parse(sql, p.dialect)
		if err != nil {
			panic(err)
		}
	}); fail != nil {
		return *fail
	}
	stmts := make([]exp.Expression, 0, len(parsed))
	for _, stmt := range parsed {
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	if len(stmts) != 1 {
		return failResult("PARSE", fmt.Sprintf("expected 1 statement, got %d", len(stmts)))
	}
	p.root = stmts[0]
	if !isKnownRoot(p.root) {
		return failResult("PARSE", fmt.Sprintf("unsupported root %s", exp.ClassName(p.root.Kind())))
	}

	for _, fn := range p.root.FindAll(exp.TraitFunc) {
		if dangerousFuncs[strings.ToLower(fn.Name())] {
			return failResult("VALIDATE", fmt.Sprintf("function '%s' can execute opaque SQL / IO — not allowed", strings.ToLower(fn.Name())))
		}
	}

	foldQuoted := p.dialect == "mysql"
	for _, ident := range p.root.FindAll(exp.KindIdentifier) {
		if foldQuoted || !truthy(ident.Arg("quoted")) {
			if text, ok := ident.Arg("this").(string); ok {
				ident.Set("this", strings.ToLower(text))
			}
		}
	}

	if fail := p.classifyWrite(); fail != nil {
		return *fail
	}

	for _, cte := range p.root.FindAll(exp.KindCTE) {
		body := cte.This()
		if body != nil && (body.Kind() == exp.KindInsert || body.Kind() == exp.KindUpdate || body.Kind() == exp.KindDelete || body.Kind() == exp.KindMerge) {
			return failResult("VALIDATE", "data-modifying CTE not supported")
		}
	}
	if len(p.root.FindAll(exp.KindPivot)) > 0 {
		return failResult("VALIDATE", "PIVOT/UNPIVOT not supported")
	}
	for _, join := range p.root.FindAll(exp.KindJoin) {
		if strings.ToUpper(fmt.Sprint(firstNonNil(join.Arg("method"), ""))) == "NATURAL" || fmt.Sprint(join.Arg("kind")) == "NATURAL" {
			return failResult("VALIDATE", "NATURAL JOIN not supported (shared-column lineage is ambiguous)")
		}
	}
	for _, oc := range p.root.FindAll(exp.KindOnConflict) {
		if truthy(oc.Arg("constraint")) {
			return failResult("VALIDATE", "ON CONFLICT ON CONSTRAINT — cannot map a named constraint to columns")
		}
	}

	if fail := runStage("VALIDATE", func() {
		opts := optimizer.DefaultQualifyOpts()
		opts.Dialect = p.dialect
		opts.Schema = p.loweredMapping
		opts.InferSchema = boolPtr(false)
		opts.ValidateQualifyColumns = !p.isWrite
		p.qroot = optimizer.Qualify(p.root.Copy(), opts)
	}); fail != nil {
		return *fail
	}

	if p.qroot.Kind() == exp.KindDelete && truthy(p.qroot.Arg("tables")) {
		p.qroot.Set("tables", nil)
	}

	for {
		var dead []exp.Expression
		for _, with := range p.qroot.FindAll(exp.KindWith) {
			for _, cte := range with.Expressions() {
				if !p.cteReferenced(cte, p.qroot) {
					dead = append(dead, cte)
				}
			}
		}
		if len(dead) == 0 {
			break
		}
		for _, cte := range dead {
			with := cte.Parent()
			cte.Pop()
			if with != nil && with.Kind() == exp.KindWith && len(with.Expressions()) == 0 {
				with.Pop()
			}
		}
	}

	p.cteNames = map[string]bool{}
	for _, cte := range p.qroot.FindAll(exp.KindCTE) {
		p.cteNames[strings.ToLower(cte.AliasOrName())] = true
	}
	targetNames := map[string]bool{}
	if p.writeTarget != nil {
		for _, target := range p.writeTarget.FindAll(exp.KindTable) {
			targetNames[strings.ToLower(target.Name())] = true
		}
	}
	for _, tbl := range p.qroot.FindAll(exp.KindTable) {
		if this := tbl.This(); this != nil && this.Kind() != exp.KindIdentifier {
			continue
		}
		nm := strings.ToLower(tbl.Name())
		if p.cteNames[nm] || targetNames[nm] {
			continue
		}
		if !p.schemaTables[nm] {
			return failResult("VALIDATE", "unknown table "+tbl.Name())
		}
	}

	if fail := runStage("CONVERT", func() {
		p.buildScopes()
	}); fail != nil {
		return *fail
	}

	if p.isWrite {
		p.synthesizeWriteScope()
	}

	if p.qroot.Kind() == exp.KindInsert && isSelectOrSet(p.qroot.Expr()) {
		p.payloadQuery = p.qroot.Expr()
	} else if p.qroot.Kind() == exp.KindSelect && p.qroot.Arg("into") != nil {
		p.payloadQuery = p.qroot
	}

	var result ProbeResult
	if fail := runStage("LINEAGE", func() {
		result = p.lineage()
	}); fail != nil {
		return *fail
	}
	return result
}

func ProbeJSON(sql, dialect, schemaJSON string) (string, error) {
	sch, err := decodeSchemaJSON(schemaJSON)
	if err != nil {
		return "", err
	}
	result := Probe(sql, dialect, sch)
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (p *prober) classifyWrite() *ProbeResult {
	p.isWrite = false
	p.analyzeQuery = p.root
	p.writeTarget = nil
	p.payloadExprs = nil

	switch p.root.Kind() {
	case exp.KindCreate:
		p.writeTarget = p.root.This()
		if isSelectOrSet(p.root.Expr()) {
			p.isWrite = true
			p.analyzeQuery = p.root.Expr()
		} else {
			fail := failResult("VALIDATE", "CREATE without analyzable query")
			return &fail
		}
	case exp.KindInsert:
		p.isWrite = true
		p.analyzeQuery = nil
		p.writeTarget = p.root.This()
		if !isSelectOrSet(p.root.Expr()) {
			p.payloadExprs = []exp.Expression{p.root.Expr()}
		}
	case exp.KindUpdate:
		p.isWrite = true
		p.analyzeQuery = nil
		p.writeTarget = p.root.This()
		for _, e := range expressionsFor(p.root, "expressions") {
			if e.Kind() == exp.KindEQ {
				p.payloadExprs = append(p.payloadExprs, e.Expr())
			}
		}
	case exp.KindDelete:
		p.isWrite = true
		p.analyzeQuery = nil
		p.writeTarget = p.root.This()
	case exp.KindMerge:
		p.isWrite = true
		p.analyzeQuery = nil
		p.writeTarget = p.root.This()
		whens := asExpression(p.root.Arg("whens"))
		if whens != nil {
			for _, when := range whens.Expressions() {
				then := asExpression(when.Arg("then"))
				if then == nil {
					continue
				}
				if then.Kind() == exp.KindUpdate {
					for _, e := range expressionsFor(then, "expressions") {
						if e.Kind() == exp.KindEQ {
							p.payloadExprs = append(p.payloadExprs, e.Expr())
						}
					}
				} else if then.Kind() == exp.KindInsert && then.Expr() != nil {
					p.payloadExprs = append(p.payloadExprs, then.Expr())
				}
			}
		}
	case exp.KindSelect:
		if p.root.Arg("into") != nil {
			p.isWrite = true
			p.writeTarget = asExpression(p.root.Arg("into"))
			p.analyzeQuery = nil
		}
	}
	return nil
}

type failPanic struct{ result ProbeResult }

func runStage(stage string, fn func()) *ProbeResult {
	var fail *ProbeResult
	func() {
		defer func() {
			if r := recover(); r != nil {
				if f, ok := r.(failPanic); ok {
					fail = &f.result
					return
				}
				f := failResult(stage, panicDetail(r))
				fail = &f
			}
		}()
		fn()
	}()
	return fail
}

func failResult(stage string, detail string) ProbeResult {
	stageCopy := stage
	return ProbeResult{
		Resolved:      false,
		FailedStage:   &stageCopy,
		Detail:        truncateDetail(detail),
		OutputColumns: 0,
		TracedColumns: 0,
		Origins:       []OriginInfo{},
		References:    map[string][]string{},
		IsWrite:       false,
		RewrittenSQL:  nil,
	}
}

func panicDetail(r any) string {
	if err, ok := r.(error); ok {
		return truncateDetail(err.Error())
	}
	return truncateDetail(fmt.Sprint(r))
}

func truncateDetail(detail string) string {
	runes := []rune(detail)
	if len(runes) > 150 {
		return string(runes[:150])
	}
	return detail
}

func isKnownRoot(e exp.Expression) bool {
	if e == nil {
		return false
	}
	if e.Is(exp.TraitSetOperation) {
		return true
	}
	switch e.Kind() {
	case exp.KindSelect, exp.KindInsert, exp.KindUpdate, exp.KindDelete, exp.KindMerge, exp.KindCreate:
		return true
	}
	return false
}

func isSelectOrSet(e exp.Expression) bool {
	return e != nil && (e.Kind() == exp.KindSelect || e.Is(exp.TraitSetOperation))
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func (p *prober) buildScopes() {
	p.scopes = nil
	seenScopeExprs := map[exp.Expression]bool{}
	for _, sc := range optimizer.TraverseScope(p.qroot) {
		p.scopes = append(p.scopes, sc)
		seenScopeExprs[sc.Expression] = true
	}
	for _, sel := range p.qroot.FindAll(exp.KindSelect) {
		if seenScopeExprs[sel] {
			continue
		}
		for _, sc := range optimizer.TraverseScope(sel) {
			if !seenScopeExprs[sc.Expression] {
				p.scopes = append(p.scopes, sc)
				seenScopeExprs[sc.Expression] = true
			}
		}
	}
	p.col2scope = map[exp.Expression]*optimizer.Scope{}
	for _, sc := range p.scopes {
		for _, c := range sc.Columns() {
			p.col2scope[c] = sc
		}
	}
	p.scopeOfSelect = map[exp.Expression]*optimizer.Scope{}
	for _, sc := range p.scopes {
		p.scopeOfSelect[sc.Expression] = sc
	}
}

func (p *prober) synthesizeWriteScope() {
	wsrcs := []exp.Expression{}
	addSrc := func(x exp.Expression) {
		if x == nil {
			return
		}
		if x.Kind() == exp.KindSchema {
			x = x.This()
		}
		if x != nil && (x.Kind() == exp.KindTable || x.Kind() == exp.KindSubquery) {
			wsrcs = append(wsrcs, x.Copy())
		}
	}
	addSrc(p.qroot.This())
	if wfrm := asExpression(firstNonNil(p.qroot.Arg("from"), p.qroot.Arg("from_"))); wfrm != nil && wfrm.Kind() == exp.KindFrom {
		addSrc(wfrm.This())
		for _, e := range expressionsFor(wfrm, "expressions") {
			addSrc(e)
		}
	}
	for _, join := range expressionsFor(p.qroot, "joins") {
		addSrc(join.This())
	}
	wusing := p.qroot.Arg("using")
	if wusing != nil {
		if _, ok := wusing.(bool); !ok {
			switch v := wusing.(type) {
			case []exp.Expression:
				for _, u := range v {
					addSrc(u)
				}
			case exp.Expression:
				addSrc(v)
			case []any:
				for _, item := range v {
					addSrc(asExpression(item))
				}
			}
		}
	}
	if len(wsrcs) == 0 {
		return
	}
	func() {
		defer func() {
			if recover() != nil {
				p.writeScope = nil
			}
		}()
		lit := exp.LiteralNumber(1)
		syn := exp.Select(exp.Args{"expressions": []exp.Expression{lit}})
		syn.Set("from_", exp.From(exp.Args{"this": wsrcs[0]}))
		for _, s := range wsrcs[1:] {
			syn.Append("joins", exp.Join(exp.Args{"this": s, "kind": "CROSS"}))
		}
		if wwith := asExpression(firstNonNil(p.qroot.Arg("with"), p.qroot.Arg("with_"))); wwith != nil {
			// probe.py:736 attaches this under the bare key "with"; in CPython that is
			// still walked (Expression.walk iterates live args key-agnostically), so
			// traverse_scope collects and binds the CTE. Go's walkInScope iterates the
			// DECLARED ArgKeys for the Kind, not live args, so a bare-"with" subtree is
			// never descended and the CTE is silently dropped -> fail-open shadow-CTE
			// leak on writes. Set the declared "with_" key so Go binds the CTE exactly
			// as CPython does. Do NOT revert to bare "with".
			syn.Set("with_", wwith.Copy())
		}
		synOpts := optimizer.DefaultQualifyOpts()
		synOpts.Dialect = p.dialect
		synOpts.Schema = p.loweredMapping
		synOpts.QualifyColumns = false
		synOpts.ValidateQualifyColumns = false
		synOpts.ExpandStars = false
		synOpts.InferSchema = boolPtr(false)
		synq := optimizer.Qualify(syn, synOpts)
		synScopes := optimizer.TraverseScope(synq)
		for _, sc := range synScopes {
			for _, c := range sc.Columns() {
				if _, ok := p.col2scope[c]; !ok {
					p.col2scope[c] = sc
				}
			}
		}
		if len(synScopes) > 0 {
			p.writeScope = synScopes[len(synScopes)-1]
		}
	}()

	cteScopes := map[string]*optimizer.Scope{}
	topWith := asExpression(firstNonNil(p.qroot.Arg("with"), p.qroot.Arg("with_")))
	if topWith != nil {
		for _, cte := range topWith.Expressions() {
			body := cte.This()
			bs := p.scopeOfSelect[body]
			if bs == nil && body != nil && body.Is(exp.TraitSetOperation) {
				func() {
					defer func() { _ = recover() }()
					for _, sc := range optimizer.TraverseScope(body) {
						for _, c := range sc.Columns() {
							if _, ok := p.col2scope[c]; !ok {
								p.col2scope[c] = sc
							}
						}
						if _, ok := p.scopeOfSelect[sc.Expression]; !ok {
							p.scopeOfSelect[sc.Expression] = sc
						}
						if sc.Expression == body {
							bs = sc
						}
					}
				}()
			}
			if bs != nil {
				cteScopes[strings.ToLower(cte.AliasOrName())] = bs
			}
		}
	}
	for _, sc := range p.scopes {
		if p.writeScope != nil && sc.Parent == nil {
			sc.Parent = p.writeScope
		}
		if sc.Expression != nil && sc.Expression.FindAncestor(exp.KindCTE) == nil {
			for nm, src := range sc.Sources {
				if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
					key := strings.ToLower(tbl.Name())
					if cteScope, ok := cteScopes[key]; ok {
						sc.Sources[nm] = cteScope
					}
				}
			}
		}
	}
}

func (p *prober) lineage() ProbeResult {
	p.references = map[string]map[string]bool{}
	aliasMap := p.aliasSources(p.qroot)

	p.opaqueSelects = map[exp.Expression]bool{}
	for _, node := range append(p.qroot.FindAll(exp.KindSelect), p.findAllSetOperations(p.qroot)...) {
		if p.isExpressionSubquery(node) {
			p.opaqueSelects[node] = true
			p.addRef(SUBQUERY, p.bases(node.FindAll(exp.KindColumn)))
		}
	}

	for _, so := range p.findAllSetOperations(p.qroot) {
		distinct := so.Arg("distinct")
		if so.Kind() == exp.KindUnion && !truthy(distinct) {
			continue
		}
		for _, sel := range so.FindAll(exp.KindSelect) {
			if sel.FindAncestor(exp.KindUnion, exp.KindExcept, exp.KindIntersect) == so || sel.Parent() == so {
				p.addRef(SET_OP, p.bases(sel.Selects()))
			}
		}
	}

	for _, with := range p.qroot.FindAll(exp.KindWith) {
		if truthy(with.Arg("recursive")) {
			for _, cte := range with.Expressions() {
				if !p.cteReferenced(cte, p.qroot) {
					continue
				}
				if body := cte.This(); body != nil {
					p.addRef(RECURSIVE, p.bases(body.FindAll(exp.KindColumn)))
				}
			}
		}
	}

	addClause := func(nodes []exp.Expression, ctx string) {
		for _, n := range nodes {
			if n == nil || p.enclosingOpaque(n) != nil {
				continue
			}
			p.addRef(ctx, p.bases([]exp.Expression{n}))
		}
	}
	addOutputRefs := func(sel exp.Expression, terms []exp.Expression, ctx string) {
		projs := sel.Selects()
		for _, t := range terms {
			if t == nil || p.enclosingOpaque(t) != nil {
				continue
			}
			inner := t
			if t.Kind() == exp.KindOrdered {
				inner = t.This()
			}
			if inner != nil && inner.Kind() == exp.KindLiteral && inner.IsInt() {
				idx64, _ := strconv.ParseInt(inner.Name(), 10, 64)
				i := int(idx64) - 1
				if i >= 0 && i < len(projs) {
					p.addRef(ctx, p.bases([]exp.Expression{projs[i]}))
				}
				continue
			}
			for _, col := range inner.FindAll(exp.KindColumn) {
				b := p.resolve(col, map[resolveKey]bool{})
				if len(b) == 0 && col.TableName() == "" {
					for _, proj := range projs {
						if proj.AliasOrName() == col.Name() {
							b = p.bases([]exp.Expression{proj})
							break
						}
					}
				}
				p.addRef(ctx, b)
			}
		}
	}

	for _, sel := range p.qroot.FindAll(exp.KindSelect) {
		if p.opaqueSelects[sel] {
			continue
		}
		if w := asExpression(sel.Arg("where")); w != nil {
			addClause([]exp.Expression{w.This()}, PREDICATE)
		}
		if h := asExpression(sel.Arg("having")); h != nil {
			addClause([]exp.Expression{h.This()}, PREDICATE)
		}
		if g := asExpression(sel.Arg("group")); g != nil {
			gcols := append([]exp.Expression(nil), g.Expressions()...)
			for _, key := range []string{"rollup", "cube", "grouping_sets"} {
				gcols = append(gcols, expressionsFor(g, key)...)
			}
			addOutputRefs(sel, gcols, GROUP_BY)
		}
		if q := asExpression(sel.Arg("qualify")); q != nil {
			addClause([]exp.Expression{q.This()}, PREDICATE)
		}
		for _, j := range expressionsFor(sel, "joins") {
			if on := asExpression(j.Arg("on")); on != nil {
				addClause([]exp.Expression{on}, JOIN)
			}
			addClause(expressionsFor(j, "using"), JOIN)
		}
		if order := asExpression(sel.Arg("order")); order != nil {
			addOutputRefs(sel, order.Expressions(), ORDER_BY)
		}
		if dstn := asExpression(sel.Arg("distinct")); dstn != nil {
			on := asExpression(dstn.Arg("on"))
			if on == nil {
				addClause(sel.Selects(), GROUP_BY)
			} else {
				keys := []exp.Expression{on}
				if on.Kind() == exp.KindTuple {
					keys = on.Expressions()
				}
				addOutputRefs(sel, keys, GROUP_BY)
			}
		}
	}

	for _, so := range p.findAllSetOperations(p.qroot) {
		order := asExpression(so.Arg("order"))
		if order == nil {
			continue
		}
		branches := p.leafSelects(so)
		var left exp.Expression
		if len(branches) > 0 {
			left = branches[0]
		}
		posBases := []map[string]bool{}
		if left != nil {
			for i := range left.Selects() {
				b := map[string]bool{}
				for _, br := range branches {
					selects := br.Selects()
					if i < len(selects) {
						addSet(b, p.bases([]exp.Expression{selects[i]}))
					}
				}
				posBases = append(posBases, b)
			}
		}
		nameOf := map[string]map[string]bool{}
		if left != nil {
			for i, b := range posBases {
				nameOf[left.Selects()[i].AliasOrName()] = b
			}
		}
		for _, t := range order.Expressions() {
			inner := t
			if t.Kind() == exp.KindOrdered {
				inner = t.This()
			}
			if inner != nil && inner.Kind() == exp.KindLiteral && inner.IsInt() {
				idx64, _ := strconv.ParseInt(inner.Name(), 10, 64)
				i := int(idx64) - 1
				if i >= 0 && i < len(posBases) {
					p.addRef(ORDER_BY, posBases[i])
				}
			} else if inner != nil {
				for _, col := range inner.FindAll(exp.KindColumn) {
					b := p.resolve(col, map[resolveKey]bool{})
					if len(b) == 0 {
						b = nameOf[col.Name()]
					}
					p.addRef(ORDER_BY, b)
				}
			}
		}
	}

	for _, over := range p.qroot.FindAll(exp.KindWindow) {
		if p.enclosingOpaque(over) == nil {
			addClause(expressionsFor(over, "partition_by"), PREDICATE)
			if o := asExpression(over.Arg("order")); o != nil {
				addClause(o.Expressions(), ORDER_BY)
			}
		}
	}
	for _, filt := range p.qroot.FindAll(exp.KindFilter) {
		if p.enclosingOpaque(filt) == nil {
			addClause([]exp.Expression{filt.Expr()}, AGGREGATE)
		}
	}

	for _, sel := range p.qroot.FindAll(exp.KindSelect) {
		if p.opaqueSelects[sel] {
			continue
		}
		srcNodes := []exp.Expression{}
		if frm := asExpression(sel.Arg("from")); frm != nil {
			srcNodes = append(srcNodes, frm)
		}
		for _, j := range expressionsFor(sel, "joins") {
			if j.This() != nil {
				srcNodes = append(srcNodes, j.This())
			}
		}
		srcNodes = append(srcNodes, expressionsFor(sel, "laterals")...)
		for _, node := range srcNodes {
			for _, c := range node.FindAll(exp.KindColumn) {
				if c.FindAncestor(exp.KindSelect) == sel {
					p.addRef(OTHER, p.bases([]exp.Expression{c}))
				}
			}
		}
	}

	for _, tc := range p.qroot.FindAll(exp.KindTableColumn) {
		if tc.FindAncestor(exp.KindDot) != nil {
			continue
		}
		if src := p.srcInScope(tc, strings.ToLower(tc.Name())); src != nil {
			p.addRef(OTHER, p.srcAllCols(src))
		}
	}
	for _, dot := range p.qroot.FindAll(exp.KindDot) {
		base := dot.This()
		for base != nil && base.Kind() == exp.KindParen {
			base = base.This()
		}
		baseName := ""
		if base != nil {
			baseName = strings.ToLower(base.Name())
		}
		src := p.srcInScope(dot, baseName)
		if src == nil {
			continue
		}
		fld := ""
		if dot.Expr() != nil {
			fld = strings.ToLower(dot.Expr().Name())
		}
		if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
			cols := p.schemaCI[strings.ToLower(tbl.Name())]
			if fld != "" && cols[fld] {
				p.addRef(OTHER, stringSet(tbl.Name()+"."+fld))
			} else {
				p.addRef(OTHER, p.srcAllCols(src))
			}
		} else if scope, ok := src.(*optimizer.Scope); ok && fld != "" {
			p.addRef(OTHER, p.resolveScopeOutput(scope, fld, map[resolveKey]bool{}))
		} else {
			p.addRef(OTHER, p.srcAllCols(src))
		}
	}
	for _, c := range p.qroot.FindAll(exp.KindColumn) {
		if c.This() != nil && c.This().Kind() == exp.KindStar && c.TableName() != "" {
			if src := p.srcInScope(c, strings.ToLower(c.TableName())); src != nil {
				p.addRef(OTHER, p.srcAllCols(src))
			}
		}
	}

	var rewrittenSQL *string
	if p.analyzeQuery != nil && !p.isWrite {
		for _, sel := range p.qroot.FindAll(exp.KindSelect) {
			for _, proj := range sel.Selects() {
				if p.isStar(proj) {
					return failResult("VALIDATE", "unexpandable `*` (table-function / array / LATERAL source) — cannot bind mask ordinals")
				}
			}
		}
		oBranches := []exp.Expression{p.analyzeQuery}
		if p.analyzeQuery.Is(exp.TraitSetOperation) {
			oBranches = p.leafSelects(p.analyzeQuery)
		}
		qBranches := []exp.Expression{p.qroot}
		if p.qroot.Is(exp.TraitSetOperation) {
			qBranches = p.leafSelects(p.qroot)
		}
		starExpanded := false
		for i, qsel := range qBranches {
			var osel exp.Expression
			if i < len(oBranches) {
				osel = oBranches[i]
			}
			if osel == nil {
				continue
			}
			hasStar := false
			for _, proj := range osel.Selects() {
				if p.isStar(proj) {
					hasStar = true
					break
				}
			}
			if !hasStar {
				continue
			}
			ok, why := p.expandableSources(osel)
			if !ok {
				return failResult("VALIDATE", fmt.Sprintf("`*` over %s — outside the faithful-expansion envelope", why))
			}
			bareStar := false
			starCount := 0
			for _, proj := range osel.Selects() {
				if proj.Kind() == exp.KindStar {
					bareStar = true
				}
				if p.isStar(proj) {
					starCount++
				}
			}
			if bareStar {
				if starCount > 1 {
					return failResult("VALIDATE", "bare `*` mixed with another star — column order can't bind to mask ordinals")
				}
				p.resortBareStarInplace(osel, qsel)
			}
			starExpanded = true
		}
		if starExpanded {
			s, err := sqlglot.Generate(p.qroot, p.dialect, generator.Options{})
			if err != nil {
				panic(err)
			}
			rewrittenSQL = &s
		}
	}

	origins := []OriginInfo{}
	if p.analyzeQuery != nil {
		aq := p.qroot
		if p.root.Kind() == exp.KindCreate {
			aq = p.qroot.Expr()
		}
		if aq != nil && aq.Is(exp.TraitSetOperation) {
			branches := p.leafSelects(aq)
			branchSelects := make([][]exp.Expression, 0, len(branches))
			for _, br := range branches {
				branchSelects = append(branchSelects, br.Selects())
			}
			leftSels := []exp.Expression{}
			if len(branchSelects) > 0 {
				leftSels = branchSelects[0]
			}
			for i := range leftSels {
				srcs := map[string]bool{}
				identity := true
				for _, bs := range branchSelects {
					if i < len(bs) {
						b, sub := p.projIdent(bs[i], map[identKey]bool{})
						addSet(srcs, b)
						identity = identity && sub
					}
				}
				if identity {
					origins = append(origins, OriginInfo{Column: leftSels[i].AliasOrName(), Origins: sortedSet(srcs)})
				} else {
					origins = append(origins, OriginInfo{Column: leftSels[i].AliasOrName(), Origins: []string{}})
					p.addRef(DERIVED, srcs)
				}
			}
		} else if aq != nil {
			for _, proj := range aq.Selects() {
				bases, isID := p.projIdent(proj, map[identKey]bool{})
				if isID {
					origins = append(origins, OriginInfo{Column: proj.AliasOrName(), Origins: sortedSet(bases)})
				} else {
					origins = append(origins, OriginInfo{Column: proj.AliasOrName(), Origins: []string{}})
					p.addRef(DERIVED, bases)
				}
			}
		}
	}

	accounted := map[string]bool{}
	for _, origin := range origins {
		for _, value := range origin.Origins {
			accounted[value] = true
		}
	}
	for _, cols := range p.references {
		addSet(accounted, cols)
	}

	skipArgs := map[string]bool{"expressions": true, "from": true, "from_": true, "joins": true, "with": true, "with_": true, "laterals": true, "pivots": true, "into": true, "hint": true, "kind": true, "operation": true, "operation_modifiers": true}
	for _, sel := range p.qroot.FindAll(exp.KindSelect) {
		if p.opaqueSelects[sel] {
			continue
		}
		for key, node := range exp.ArgsOf(sel) {
			if skipArgs[key] {
				continue
			}
			for _, sub := range expressionsFromAny(node) {
				for _, c := range sub.FindAll(exp.KindColumn) {
					if p.enclosingOpaque(c) == nil {
						p.addRef(OTHER, subtractSet(p.resolve(c, map[resolveKey]bool{}), accounted))
					}
				}
			}
		}
	}

	if p.isWrite {
		destIDs := map[exp.Expression]bool{}
		for _, ins := range p.qroot.FindAll(exp.KindInsert) {
			this := ins.This()
			if this != nil && (this.Kind() == exp.KindSchema || this.Kind() == exp.KindTuple) {
				for _, c := range this.FindAll(exp.KindColumn) {
					destIDs[c] = true
				}
			}
		}
		for _, eq := range p.setAssignments(p.qroot) {
			if lhs := eq.This(); lhs != nil && lhs.Kind() == exp.KindColumn {
				destIDs[lhs] = true
			}
		}
		phys := map[string]bool{}
		for _, t := range p.qroot.FindAll(exp.KindTable) {
			if !p.cteNames[strings.ToLower(t.Name())] {
				phys[t.Name()] = true
			}
		}
		writePhys := map[string]bool{}
		if p.writeScope != nil {
			for _, src := range p.writeScope.Sources {
				if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
					writePhys[strings.ToLower(tbl.Name())] = true
				}
			}
		}
		hasOnConflict := len(p.qroot.FindAll(exp.KindOnConflict)) > 0
		aliasNamesCI := map[string]bool{}
		for key := range aliasMap {
			aliasNamesCI[strings.ToLower(key)] = true
		}

		resolveWS := func(alias, name string) map[string]bool {
			if p.writeScope == nil || alias == "" {
				return map[string]bool{}
			}
			src := p.writeScope.Sources[alias]
			if src == nil {
				return map[string]bool{}
			}
			if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
				return stringSet(tbl.Name() + "." + name)
			}
			if sc, ok := src.(*optimizer.Scope); ok {
				return p.resolveScopeOutput(sc, name, map[resolveKey]bool{})
			}
			return map[string]bool{}
		}
		wbase := func(name, alias string, col exp.Expression) (map[string]bool, bool) {
			bases := map[string]bool{}
			if col != nil {
				bases = p.resolve(col, map[resolveKey]bool{})
			}
			if len(bases) == 0 && alias != "" {
				bases = resolveWS(alias, name)
			}
			if len(bases) == 0 && alias == "" {
				nl := strings.ToLower(name)
				bases = map[string]bool{}
				for t := range phys {
					if p.schemaCI[strings.ToLower(t)][nl] {
						bases[t+"."+name] = true
					}
				}
			}
			if len(bases) == 0 {
				return nil, false
			}
			for base := range bases {
				dot := strings.LastIndex(base, ".")
				if dot < 0 {
					return nil, false
				}
				table, colName := base[:dot], base[dot+1:]
				if !p.schemaCI[strings.ToLower(table)][strings.ToLower(colName)] {
					return nil, false
				}
			}
			return bases, true
		}

		for _, c := range p.qroot.FindAll(exp.KindColumn) {
			if destIDs[c] || p.col2scope[c] != nil {
				continue
			}
			if c.This() != nil && c.This().Kind() == exp.KindStar {
				continue
			}
			if c.TableName() == "" && p.writeScope != nil && c.Name() != "" {
				if wsrc, ok := p.writeScope.Sources[strings.ToLower(c.Name())]; ok {
					hasColumn := false
					for t := range writePhys {
						if p.schemaCI[t][strings.ToLower(c.Name())] {
							hasColumn = true
							break
						}
					}
					if !hasColumn {
						if tbl, ok := wsrc.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
							p.addRef(OTHER, subtractSet(p.srcAllCols(tbl), accounted))
							continue
						}
					}
				}
			}
			if strings.ToLower(c.TableName()) == "excluded" && hasOnConflict && !aliasNamesCI["excluded"] {
				continue
			}
			bases, ok := wbase(c.Name(), c.TableName(), c)
			if !ok {
				return failResult("VALIDATE", fmt.Sprintf("unresolved column '%s' in write", c.Name()))
			}
			p.addRef(OTHER, subtractSet(bases, accounted))
		}
		if p.payloadQuery != nil {
			branches := []exp.Expression{p.payloadQuery}
			if p.payloadQuery.Is(exp.TraitSetOperation) {
				branches = p.leafSelects(p.payloadQuery)
			}
			for _, s := range branches {
				p.addRef(OTHER, subtractSet(p.bases(s.Selects()), accounted))
			}
		}
		for _, j := range p.qroot.FindAll(exp.KindJoin) {
			for _, ident := range expressionsFor(j, "using") {
				bases, ok := wbase(ident.Name(), "", nil)
				if !ok {
					return failResult("VALIDATE", fmt.Sprintf("unresolved USING column '%s' in write", ident.Name()))
				}
				p.addRef(OTHER, subtractSet(bases, accounted))
			}
		}
		if len(p.qroot.FindAll(exp.KindStar)) > 0 {
			for t := range phys {
				for col := range p.schemaCI[strings.ToLower(t)] {
					p.addRef(OTHER, stringSet(t+"."+col))
				}
			}
		}
		for _, fn := range p.qroot.FindAll(exp.TraitFunc) {
			if strings.ToLower(fn.Name()) == "values" {
				for _, a := range fn.FindAll(exp.KindIdentifier) {
					bases, ok := wbase(a.Name(), "", nil)
					if !ok {
						return failResult("VALIDATE", fmt.Sprintf("unresolved VALUES() column '%s' in write", a.Name()))
					}
					p.addRef(OTHER, subtractSet(bases, accounted))
				}
			}
		}
	}

	refsOut := map[string][]string{}
	for key, values := range p.references {
		if len(values) > 0 {
			refsOut[key] = sortedSet(values)
		}
	}
	traced := 0
	for _, origin := range origins {
		if len(origin.Origins) > 0 {
			traced++
		}
	}
	return ProbeResult{
		Resolved:      true,
		FailedStage:   nil,
		Detail:        "ok",
		OutputColumns: len(origins),
		TracedColumns: traced,
		Origins:       origins,
		References:    refsOut,
		IsWrite:       p.isWrite,
		RewrittenSQL:  rewrittenSQL,
	}
}

func expressionsFromAny(value any) []exp.Expression {
	switch v := value.(type) {
	case nil:
		return nil
	case exp.Expression:
		return []exp.Expression{v}
	case []exp.Expression:
		return v
	case []any:
		out := []exp.Expression{}
		for _, item := range v {
			if expr := asExpression(item); expr != nil {
				out = append(out, expr)
			}
		}
		return out
	default:
		return nil
	}
}

func (p *prober) addRef(ctx string, bases map[string]bool) {
	if len(bases) == 0 {
		return
	}
	if p.references[ctx] == nil {
		p.references[ctx] = map[string]bool{}
	}
	addSet(p.references[ctx], bases)
}

func (p *prober) findAllSetOperations(root exp.Expression) []exp.Expression {
	if root == nil {
		return nil
	}
	return root.FindAll(exp.TraitSetOperation)
}

func (p *prober) cteColNames(cte exp.Expression) []string {
	if cte == nil {
		return nil
	}
	cols := expressionsFor(cte, "columns")
	if len(cols) == 0 {
		al := asExpression(cte.Arg("alias"))
		cols = expressionsFor(al, "columns")
	}
	if len(cols) == 0 {
		return nil
	}
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		out = append(out, c.Name())
	}
	return out
}

func (p *prober) resolve(col exp.Expression, seen map[resolveKey]bool) map[string]bool {
	if col == nil {
		return map[string]bool{}
	}
	scope := p.col2scope[col]
	if scope == nil {
		return map[string]bool{}
	}
	return p.resolveIn(col.Name(), col.TableName(), scope, seen)
}

func (p *prober) resolveIn(name, alias string, scope *optimizer.Scope, seen map[resolveKey]bool) map[string]bool {
	key := resolveKey{scope: scope, alias: alias, name: name}
	if seen[key] {
		return map[string]bool{}
	}
	seen[key] = true
	var src any
	for sc := scope; sc != nil; sc = sc.Parent {
		if alias != "" {
			if candidate, ok := sc.Sources[alias]; ok {
				src = candidate
				break
			}
		} else if len(sc.Sources) == 1 {
			for _, candidate := range sc.Sources {
				src = candidate
			}
			break
		}
	}
	if src == nil {
		return map[string]bool{}
	}
	if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
		return stringSet(tbl.Name() + "." + name)
	}
	if sc, ok := src.(*optimizer.Scope); ok {
		return p.resolveScopeOutput(sc, name, seen)
	}
	return map[string]bool{}
}

func (p *prober) resolveScopeOutput(scope *optimizer.Scope, outName string, seen map[resolveKey]bool) map[string]bool {
	out := map[string]bool{}
	if scope == nil || scope.Expression == nil {
		return out
	}
	expr := scope.Expression
	if expr.Is(exp.TraitSetOperation) {
		branches := p.leafSelects(expr)
		var names []string
		if cte := expr.Parent(); cte != nil && cte.Kind() == exp.KindCTE {
			names = p.cteColNames(cte)
		}
		if len(names) == 0 && len(branches) > 0 {
			for _, proj := range branches[0].Selects() {
				names = append(names, proj.AliasOrName())
			}
		}
		for i, nm := range names {
			if nm == outName {
				for _, br := range branches {
					selects := br.Selects()
					if i < len(selects) {
						addSet(out, p.bases([]exp.Expression{selects[i]}))
					}
				}
			}
		}
		return out
	}
	if scope.IsUnion() {
		for _, branch := range scope.UnionScopes {
			addSet(out, p.resolveScopeOutput(branch, outName, seen))
		}
		return out
	}
	if expr.Kind() != exp.KindSelect {
		return out
	}
	matched := []exp.Expression{}
	for _, proj := range expr.Selects() {
		if proj.AliasOrName() == outName {
			matched = append(matched, proj)
		}
	}
	if len(matched) == 0 {
		if cte := expr.Parent(); cte != nil && cte.Kind() == exp.KindCTE {
			names := p.cteColNames(cte)
			for i, nm := range names {
				if nm == outName && i < len(expr.Selects()) {
					matched = []exp.Expression{expr.Selects()[i]}
					break
				}
			}
		}
	}
	for _, proj := range matched {
		for _, c := range proj.FindAll(exp.KindColumn) {
			addSet(out, p.resolveIn(c.Name(), c.TableName(), scope, seen))
		}
	}
	return out
}

func (p *prober) resolveIdent(col exp.Expression, seen map[identKey]bool) (map[string]bool, bool) {
	if col == nil {
		return map[string]bool{}, true
	}
	scope := p.col2scope[col]
	if scope == nil {
		return map[string]bool{}, true
	}
	key := identKey{scope: scope, alias: col.TableName(), name: col.Name()}
	if seen[key] {
		return map[string]bool{}, true
	}
	seen2 := copyIdentSeen(seen)
	seen2[key] = true
	var src any
	for sc := scope; sc != nil; sc = sc.Parent {
		if col.TableName() != "" {
			if candidate, ok := sc.Sources[col.TableName()]; ok {
				src = candidate
				break
			}
		} else if len(sc.Sources) == 1 {
			for _, candidate := range sc.Sources {
				src = candidate
			}
			break
		}
	}
	if src == nil {
		return map[string]bool{}, true
	}
	if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
		return stringSet(tbl.Name() + "." + col.Name()), true
	}
	if sc, ok := src.(*optimizer.Scope); ok {
		return p.scopeOutIdent(sc, col.Name(), seen2)
	}
	return map[string]bool{}, true
}

func copyIdentSeen(in map[identKey]bool) map[identKey]bool {
	out := make(map[identKey]bool, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (p *prober) scopeOutIdent(scope *optimizer.Scope, outName string, seen map[identKey]bool) (map[string]bool, bool) {
	if scope == nil || scope.Expression == nil {
		return map[string]bool{}, true
	}
	expr := scope.Expression
	if expr.Is(exp.TraitSetOperation) {
		branches := p.leafSelects(expr)
		var names []string
		if cte := expr.Parent(); cte != nil && cte.Kind() == exp.KindCTE {
			names = p.cteColNames(cte)
		}
		if len(names) == 0 && len(branches) > 0 {
			for _, proj := range branches[0].Selects() {
				names = append(names, proj.AliasOrName())
			}
		}
		out, ident := map[string]bool{}, true
		for i, nm := range names {
			if nm == outName {
				for _, br := range branches {
					selects := br.Selects()
					if i < len(selects) {
						b, sub := p.projIdent(selects[i], seen)
						addSet(out, b)
						ident = ident && sub
					}
				}
			}
		}
		return out, ident
	}
	if scope.IsUnion() {
		out, ident := map[string]bool{}, true
		for _, branch := range scope.UnionScopes {
			b, sub := p.scopeOutIdent(branch, outName, seen)
			addSet(out, b)
			ident = ident && sub
		}
		return out, ident
	}
	if expr.Kind() != exp.KindSelect {
		return map[string]bool{}, true
	}
	matched := []exp.Expression{}
	for _, proj := range expr.Selects() {
		if proj.AliasOrName() == outName {
			matched = append(matched, proj)
		}
	}
	if len(matched) == 0 {
		if cte := expr.Parent(); cte != nil && cte.Kind() == exp.KindCTE {
			names := p.cteColNames(cte)
			for i, nm := range names {
				if nm == outName && i < len(expr.Selects()) {
					matched = []exp.Expression{expr.Selects()[i]}
					break
				}
			}
		}
	}
	out, ident := map[string]bool{}, true
	for _, proj := range matched {
		b, sub := p.projIdent(proj, seen)
		addSet(out, b)
		ident = ident && sub
	}
	return out, ident
}

func (p *prober) projIdent(proj exp.Expression, seen map[identKey]bool) (map[string]bool, bool) {
	ic := p.identityCol(proj)
	if ic == nil {
		out := map[string]bool{}
		if proj != nil {
			for _, c := range proj.FindAll(exp.KindColumn) {
				b, _ := p.resolveIdent(c, seen)
				addSet(out, b)
			}
		}
		return out, false
	}
	return p.resolveIdent(ic, seen)
}

func (p *prober) identityCol(proj exp.Expression) exp.Expression {
	for proj != nil && proj.Kind() == exp.KindAlias {
		proj = proj.This()
	}
	if proj != nil && proj.Kind() == exp.KindColumn {
		return proj
	}
	return nil
}

func (p *prober) bases(nodes []exp.Expression) map[string]bool {
	out := map[string]bool{}
	for _, n := range nodes {
		if n == nil {
			continue
		}
		cols := []exp.Expression{n}
		if n.Kind() != exp.KindColumn {
			cols = n.FindAll(exp.KindColumn)
		}
		for _, c := range cols {
			addSet(out, p.resolve(c, map[resolveKey]bool{}))
		}
	}
	return out
}

func (p *prober) leafSelects(setop exp.Expression) []exp.Expression {
	out := []exp.Expression{}
	if setop == nil {
		return out
	}
	for _, side := range []exp.Expression{setop.Left(), setop.Right()} {
		if side == nil {
			continue
		}
		if side.Is(exp.TraitSetOperation) {
			out = append(out, p.leafSelects(side)...)
		} else if side.Kind() == exp.KindSelect {
			out = append(out, side)
		} else if side.Kind() == exp.KindSubquery {
			inner := side.This()
			if inner != nil && inner.Is(exp.TraitSetOperation) {
				out = append(out, p.leafSelects(inner)...)
			} else if inner != nil && inner.Kind() == exp.KindSelect {
				out = append(out, inner)
			}
		}
	}
	return out
}

func (p *prober) fromSourceOrder(sel exp.Expression) []string {
	aliasOf := func(src exp.Expression) string {
		if src == nil {
			return ""
		}
		if src.Kind() == exp.KindSubquery {
			return src.Alias()
		}
		if src.Kind() == exp.KindTable {
			if alias := src.Alias(); alias != "" {
				return alias
			}
			return src.Name()
		}
		return ""
	}
	order := []string{}
	if frm := asExpression(sel.Arg("from_")); frm != nil {
		order = append(order, aliasOf(frm.This()))
		for _, e := range expressionsFor(frm, "expressions") {
			order = append(order, aliasOf(e))
		}
	}
	for _, j := range expressionsFor(sel, "joins") {
		if j.This() != nil {
			order = append(order, aliasOf(j.This()))
		}
	}
	out := []string{}
	for _, item := range order {
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (p *prober) isStar(proj exp.Expression) bool {
	return proj != nil && (proj.Kind() == exp.KindStar || (proj.Kind() == exp.KindColumn && proj.This() != nil && proj.This().Kind() == exp.KindStar))
}

func (p *prober) expandableSources(sel exp.Expression) (bool, string) {
	srcs := []exp.Expression{}
	if frm := asExpression(sel.Arg("from_")); frm != nil {
		srcs = append(srcs, frm.This())
	}
	for _, j := range expressionsFor(sel, "joins") {
		if strings.ToUpper(fmt.Sprint(firstNonNil(j.Arg("method"), ""))) == "NATURAL" || j.Arg("kind") == "NATURAL" {
			return false, "a NATURAL join"
		}
		srcs = append(srcs, j.This())
	}
	for _, s := range srcs {
		if s == nil {
			return false, "a <nil> source (table-function / VALUES / LATERAL)"
		}
		if s.Kind() == exp.KindSubquery {
			continue
		}
		if s.Kind() == exp.KindTable && s.This() != nil && s.This().Kind() == exp.KindIdentifier {
			continue
		}
		kind := exp.ClassName(s.Kind())
		if s.Kind() == exp.KindTable && s.This() != nil {
			kind = exp.ClassName(s.This().Kind())
		}
		return false, fmt.Sprintf("a %s source (table-function / VALUES / LATERAL)", strings.ToLower(kind))
	}
	return true, ""
}

func (p *prober) resortBareStarInplace(osel, qsel exp.Expression) {
	starIdx := -1
	for i, proj := range osel.Selects() {
		if proj.Kind() == exp.KindStar {
			starIdx = i
			break
		}
	}
	if starIdx < 0 {
		return
	}
	qs := qsel.Selects()
	nb, na := starIdx, len(osel.Selects())-starIdx-1
	fo := p.fromSourceOrder(qsel)
	pos := map[string]int{}
	for i, alias := range fo {
		pos[alias] = i
	}
	block := append([]exp.Expression(nil), qs[nb:len(qs)-na]...)
	position := func(proj exp.Expression) int {
		if col := proj.Find(exp.KindColumn); col != nil {
			if value, ok := pos[col.TableName()]; ok {
				return value
			}
		}
		return len(fo)
	}
	sort.SliceStable(block, func(i, j int) bool { return position(block[i]) < position(block[j]) })
	newSelects := append([]exp.Expression(nil), qs[:nb]...)
	newSelects = append(newSelects, block...)
	newSelects = append(newSelects, qs[len(qs)-na:]...)
	qsel.Set("expressions", newSelects)
}

func (p *prober) cteReferenced(cte exp.Expression, root exp.Expression) bool {
	if cte == nil || root == nil {
		return false
	}
	nm := strings.ToLower(cte.AliasOrName())
	for _, t := range root.FindAll(exp.KindTable) {
		if strings.ToLower(t.Name()) == nm && t.FindAncestor(exp.KindCTE) != cte {
			return true
		}
	}
	return false
}

func (p *prober) aliasSources(node exp.Expression) map[string]writeSource {
	m := map[string]writeSource{}
	setDefault := func(key string, value writeSource) {
		if _, ok := m[key]; !ok {
			m[key] = value
		}
	}
	for _, t := range node.FindAll(exp.KindTable) {
		setDefault(firstString(t.Alias(), t.Name()), writeSource{kind: "table", name: t.Name()})
		setDefault(t.Name(), writeSource{kind: "table", name: t.Name()})
	}
	for _, sq := range node.FindAll(exp.KindSubquery) {
		if sq.Alias() != "" && isSelectOrSet(sq.This()) {
			m[sq.Alias()] = writeSource{kind: "subq", query: sq.This()}
		}
	}
	return m
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (p *prober) resolveWrite(col exp.Expression, aliasMap map[string]writeSource, defaultTable string, seen map[[2]string]bool) map[string]bool {
	alias, name := col.TableName(), col.Name()
	key := [2]string{alias, name}
	if seen[key] {
		return map[string]bool{}
	}
	seen[key] = true
	if alias == "" {
		if defaultTable != "" {
			return stringSet(defaultTable + "." + name)
		}
		tbls := map[string]bool{}
		for _, src := range aliasMap {
			if src.kind == "table" {
				tbls[src.name] = true
			}
		}
		if len(tbls) == 1 {
			for t := range tbls {
				return stringSet(t + "." + name)
			}
		}
		return map[string]bool{}
	}
	src, ok := aliasMap[alias]
	if !ok {
		return map[string]bool{}
	}
	if src.kind == "table" {
		return stringSet(src.name + "." + name)
	}
	subSelects := []exp.Expression{src.query}
	if src.query != nil && src.query.Is(exp.TraitSetOperation) {
		subSelects = p.leafSelects(src.query)
	}
	subAlias := p.aliasSources(src.query)
	out := map[string]bool{}
	for _, s := range subSelects {
		for _, proj := range s.Selects() {
			if proj.AliasOrName() == name {
				for _, c := range proj.FindAll(exp.KindColumn) {
					addSet(out, p.resolveWrite(c, subAlias, "", seen))
				}
			}
		}
	}
	return out
}

func (p *prober) targetName(writeTarget exp.Expression) string {
	if writeTarget == nil {
		return ""
	}
	if writeTarget.Kind() == exp.KindTable {
		return writeTarget.Name()
	}
	if t := writeTarget.Find(exp.KindTable); t != nil {
		return t.Name()
	}
	return ""
}

func (p *prober) setAssignments(node exp.Expression) []exp.Expression {
	out := []exp.Expression{}
	for _, upd := range node.FindAll(exp.KindUpdate) {
		for _, e := range expressionsFor(upd, "expressions") {
			if e.Kind() == exp.KindEQ {
				out = append(out, e)
			}
		}
	}
	for _, oc := range node.FindAll(exp.KindOnConflict) {
		for _, e := range expressionsFor(oc, "expressions") {
			if e.Kind() == exp.KindEQ {
				out = append(out, e)
			}
		}
	}
	return out
}

func (p *prober) isExpressionSubquery(node exp.Expression) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Kind() {
		case exp.KindExists, exp.KindIn, exp.KindAny, exp.KindAll:
			return true
		}
		if parent.Is(exp.TraitFunc) || parent.Kind() == exp.KindArray || parent.Kind() == exp.KindUnnest {
			return true
		}
		if parent.Kind() == exp.KindSubquery {
			gp := parent.Parent()
			if gp != nil && (gp.Kind() == exp.KindFrom || gp.Kind() == exp.KindJoin) {
				return false
			}
			if gp != nil && gp.Kind() == exp.KindLateral {
				return true
			}
			if gp == nil || !gp.Is(exp.TraitSetOperation) {
				return true
			}
		}
		if parent.Kind() == exp.KindLateral {
			return true
		}
		if parent.Kind() == exp.KindFrom || parent.Kind() == exp.KindJoin {
			return false
		}
		if parent.Kind() == exp.KindSelect || parent.Kind() == exp.KindInsert || parent.Kind() == exp.KindUpdate || parent.Kind() == exp.KindMerge || parent.Kind() == exp.KindDelete || parent.Kind() == exp.KindCreate {
			return false
		}
	}
	return false
}

func (p *prober) enclosingOpaque(node exp.Expression) exp.Expression {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if p.opaqueSelects[parent] {
			return parent
		}
	}
	return nil
}

func (p *prober) srcInScope(node exp.Expression, nm string) any {
	sel := node.FindAncestor(exp.KindSelect)
	var sc *optimizer.Scope
	if sel != nil {
		sc = p.scopeOfSelect[sel]
	}
	for sc != nil {
		if src, ok := sc.Sources[nm]; ok {
			return src
		}
		sc = sc.Parent
	}
	if p.writeScope != nil {
		if src, ok := p.writeScope.Sources[nm]; ok {
			return src
		}
	}
	return nil
}

func (p *prober) srcAllCols(src any) map[string]bool {
	if tbl, ok := src.(exp.Expression); ok && tbl != nil && tbl.Kind() == exp.KindTable {
		out := map[string]bool{}
		for _, c := range p.schemaCols[strings.ToLower(tbl.Name())] {
			out[tbl.Name()+"."+c] = true
		}
		return out
	}
	if sc, ok := src.(*optimizer.Scope); ok && sc != nil {
		expr := sc.Expression
		if expr != nil && expr.Is(exp.TraitSetOperation) {
			out := map[string]bool{}
			for _, br := range p.leafSelects(expr) {
				addSet(out, p.bases(br.Selects()))
			}
			return out
		}
		if expr != nil && expr.Kind() == exp.KindSelect {
			return p.bases(expr.Selects())
		}
	}
	return map[string]bool{}
}
