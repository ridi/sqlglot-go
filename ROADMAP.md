# sqlglot-go — Milestone 1 roadmap

M1 goal: the API surface proxy-monster/probe.py uses — sqlglot.parse(sql, dialect=),
the exp AST, optimizer.qualify.qualify(...), optimizer.scope.traverse_scope + Scope.
Dialects: MySQL + Postgres ONLY. Closure ≈ 46k LOC / ~54 Python files. Port 1:1 from
.reference/sqlglot-v30.12.0/ file-by-file; port the matching tests as the oracle.

Slices (ordered; each must land `go test ./...` green before the next):

0. FOUNDATION (this run) — DONE when green:
   errors, trie, tokens (TokenType/Token/TokenizerCore/Tokenizer), expressions core
   (Node model + core/query nodes needed for SELECT), minimal SELECT parser, base Dialect.
   Tests: test_tokens.py (all but test_jinja), subset of test_expressions.py/test_parser.py.

1. PARSER CORE — split into 1a/1b so each lands green:
   - 1a: DONE (committed on branch sjcho/sqlglot-go/parser-core; 52 tests green). Grammar
     green, function long-tail + CAST deferred. Includes set operations,
     Subquery/derived tables/scalar subqueries, CTE/WITH, GROUP/HAVING/ORDER/LIMIT/
     OFFSET/FETCH/QUALIFY/DISTINCT/WINDOW/FILTER, predicates (IN/EXISTS/ANY/ALL/
     BETWEEN/IS DISTINCT FROM), CASE/IF/Paren/Tuple, function-call dispatch,
     FUNCTION_BY_NAME, Anonymous fallback, and a curated common-function set.
   - 1b: DONE when green. Landed DML/DDL statement roots, minimal CREATE/Command,
     CAST/`::`/DataType coordination, specialized FUNCTION_PARSERS, bracket
     literals/indexing, LATERAL/UNNEST/VALUES/PIVOT, and the M1 root probes.
   - 1c: DONE (committed on branch; 73 tests green). Landed LOCK/FOR, CLUSTER/DISTRIBUTE/
     SORT BY, PREWHERE, window extras (WITHIN GROUP / IGNORE-RESPECT NULLS / frame EXCLUDE),
     full parseTypes (nested/UDT/parameterized/MAP/STRUCT/ARRAY/enum/NULLABLE/COLLATE),
     INTERVAL literals, full PIVOT/UNPIVOT (Any/Alias), JSON column operators
     (-> ->> #> #>> → JSONExtract*/JSONBExtract*), SELECT TOP wiring, and a function batch.
   - 1d: defer CREATE detail (properties, column + table CONSTRAINT_PARSERS, indexes,
     clone/sequence/materialized), remaining STATEMENT_PARSERS + parse_into(into=),
     CONNECT BY / START WITH, angle-bracket inline STRUCT constructor, and the remaining
     long function tail. (None probe-critical; probe's KNOWN_ROOTS already parse.)

2. GENERATOR CORE — generator.py: Generator + generate(); un-skip .sql()-dependent tests;
   enables tests/fixtures/identity.sql round-trips (test_transpile subset). Required by
   qualify (quote_identifiers) — on the critical path, not optional.

3. SCHEMA — schema.py (MappingSchema), datatypes.py (DataType/DType build), time.py.

4. OPTIMIZER PASSES — in probe order:
   qualify_tables → normalize_identifiers → isolate_table_selects → qualify_columns
   (which always runs annotate_types + quote_identifiers) → annotate_types → resolver →
   scope (traverse_scope + Scope API) → simplify → canonicalize/normalize → qualify (driver).
   Port tests/fixtures/optimizer/*.sql per pass.

5. MYSQL + POSTGRES WIRING — dialects/{mysql,postgres}.py, parsers/{mysql,postgres}.py,
   typing/{mysql,postgres}.py, generator overrides. Both extend base directly (no fan-out).

6. PROBE END-TO-END — jsonpath, serde, lineage bits probe touches; run probe.py’s path
   (parse → qualify → traverse_scope) against real queries; parity vs Python sqlglot 30.12.0.

Cross-cutting deferred from foundation (tracked as TODOs in code):
- Expr→SQL (generator) — blocks all .sql() asserts.
- Reflection registries EXPR_CLASSES / FUNCTION_BY_NAME (expressions/__init__.py:47-51) →
  explicit Go registration tables (slice 1).
- Full schema/type annotation hierarchy beyond the parser's minimal DataType/DType nodes (slice 3).
- highlight_sql-rich parse errors already ported in foundation; parse_into(into=) deferred.

Known divergences from the r1–r3 adversarial review (differential-tested vs Python 30.12.0;
non-blocking for the foundation, must be resolved by the noted slice):
- arg ordering: newNode orders args by argTypes declaration order, not caller insertion
  order (expressions/core.go newNode). Cosmetic now — HashKey sorts keys, and Expression-
  valued children traverse in the same relative order, so equality/find/walk are unaffected.
  MUST fix before serde (slice 6) and review for the generator (slice 2), which depend on
  faithful arg order. (Upstream preserves insertion order via a dict.)
- parser-level comment bubbling: `SELECT a FROM t /* after */` attaches the trailing comment
  to the inner Identifier(t) rather than the Table node; and `_parse_alias` does not yet move
  a mid-expression comment next to the alias (upstream parser.py:8499-8501). Tokenizer-level
  attachment is correct. Affects generator round-trip fidelity — resolve in slice 2.
- deferred-feature parse divergences (expected, un-skip as features land): adjacent string
  literals `'a' 'b'` parse as Alias, not Concat (slice 1); `/*+ HINT */` errors instead of
  being ignored (slice 1); int64 overflow in ToPy/IsInt (latent until slice 4).
- Slice 1a intentionally drops `_parse_table`'s fast path so subquery detection runs before
  table-part parsing. This is a pure optimization divergence; revisit if parser profiling
  shows it matters.
- `IsWrapper` uses the Go AST's `truthy` helper rather than Python's `v is None` check because
  `newNode` does not store nil args. The wrapper semantics are equivalent for stored args.
- Full DataType semantics remain deferred to schema/DataType slice 3. Slice 1b only adds the
  parser-visible DType enum and DataType nodes needed for CAST/`::` and column definitions;
  generator `.sql()` and rich `.type` assertions stay deferred.

Resolved in the foundation review pass (were latent, now fixed + regression-tested):
- Replace()/Pop() silently no-op'd on single-value (non-list) args — the core tree-rewrite
  primitive every optimizer pass depends on. Fixed in expressions/core.go Replace (route
  index<0 through Set, the index-nil path). Tests: TestReplaceSingleValueArg, TestPopSingleValueArg.
- _parse_alias built an invalid exp.Tuple{this:...} (Tuple has no `this` arg) → ArgError.
  Added exp.Aliases (this+expressions) and use it. Test: TestParseAliases.

Resolved in the slice-1c review pass:
- parseWindow parsed the frame EXCLUDE option with raise_unmatched=false; upstream
  _parse_window (parser.py:8405) uses the default True, so a malformed EXCLUDE option must
  raise "Unknown option". Fixed in parser/parser.go. Test: TestWindowExtras (malformed case).

Slice-1b review disposition:
- Reviewer flagged parseValue ignoring its `values` param, claiming upstream has an
  `if not values and self._curr: return None` guard. VERIFIED against the pinned source:
  v30.12.0 `_parse_value` (parser.py:3783) declares `values=True` but never references it —
  the Go port is faithful; that guard exists in a different sqlglot version. No change.
- Genuine minor gap (deferred to dialect slice 5): parseValue does not yet honor
  SUPPORTS_VALUES_DEFAULT (`VALUES (DEFAULT)` → exp.var), a dialect flag; base is unaffected.

Resolved in the slice-1a review pass:
- parseLimit dropped upstream's `isinstance(expression, exp.Mod)` retreat (parser.py:5576-5579),
  so `LIMIT 10 % 3` built Mod(10,3) instead of erroring on the trailing operand. Restored the
  retreat in parser/parser.go parseLimit. Test: TestLimitPercentModRetreat.
