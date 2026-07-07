# sqlglot-go — agent guide

A Go port of **[tobymao/sqlglot](https://github.com/tobymao/sqlglot) v30.12.0** (Python).
Goal: a faithful, near-1:1 reimplementation so RIDI's `proxy-monster` lineage probe can run
natively on Go instead of sqlglot-on-GraalPy.

## Source of truth (READ THIS FIRST, always)

- The pinned Python source lives at **`.reference/sqlglot-v30.12.0/`** (gitignored, not part of the
  module). This is the **exact** upstream version proxy-monster pins (`sqlglot==30.12.0`), git SHA in
  `.reference/sqlglot-v30.12.0/GIT_SHA.txt`.
- Port from this reference, file by file, **as 1:1 as possible** — same file layout, same function/
  method names (Go-cased), same structure, same comments where they carry intent. When Go forces a
  divergence (static typing, no metaclasses, error returns), keep it minimal and note *why* in a
  comment.
- **Port the corresponding unit tests too**, 1:1, from `.reference/sqlglot-v30.12.0/tests/`. The
  upstream tests (and `tests/fixtures/*.sql`) are the correctness oracle. Fixture `.sql` files can be
  reused verbatim; loader/assertions get reimplemented in Go test form.

## Milestone 1 — scope (what proxy-monster's `probe.py` needs)

Target file: `~/repos/proxy-monster/engine/src/main/resources/py/probe.py`. It imports exactly:

- `sqlglot.parse(sql, dialect=)` and the `sqlglot.expressions` (`exp`) AST.
- `sqlglot.optimizer.qualify.qualify(root, schema, dialect, qualify_columns,
  validate_qualify_columns, expand_stars, infer_schema)`.
- `sqlglot.optimizer.scope.traverse_scope(root)` + the `Scope` API
  (`.expression / .sources / .parent / .is_union / .union_scopes / .columns`).

**Dialects: MySQL and Postgres only** for M1. proxy-monster only ever passes `"mysql"` or
`"postgres"` (see `proxy-monster/.../probe/Sqlglot.kt`). Do **not** port the other 32 dialects.
Both extend the base classes directly, no fan-out into other dialects.

The transitive closure (base + mysql + postgres) is ~46k LOC of Python across ~54 files:
tokenizer + parser, the `expressions/` package, generator (+ mysql/postgres generators), the probe-path
optimizer passes (`qualify`, `qualify_columns`, `qualify_tables`, `normalize_identifiers`,
`isolate_table_selects`, `annotate_types`, `resolver`, `scope`, `simplify`, `canonicalize`, `normalize`),
`schema`, `helper`, `time`, `errors`, `jsonpath`, `serde`, and the mysql/postgres dialect wiring.
Note: `qualify_columns` **always** runs `TypeAnnotator.annotate_scope` and `quote_identifiers`, so
`annotate_types` and the generator are on the critical path — not optional.

## Central design decision — the AST node model

Upstream `Expression` is dynamically typed: an `args: dict[str, Any]` of children (node | list | str |
bool | None), a per-class `arg_types` map, a metaclass-driven dialect registry, and heavy reflection
(`node.key`, `find_all(*types)` via isinstance). The parser (~10k LOC) and generator (~6k LOC) are
written against that dynamic model. The Go port needs an equivalent: an `Expression` interface + an
embedded base carrying `Args map[string]any` and a node `Kind`, with node types derived from
`arg_types`. Nail this model down before building much on top of it — everything depends on it.

## Conventions

- Go 1.23. Module: `github.com/sjincho/sqlglot-go`.
- Comments in **English**, US spelling (`canceled`, `color`, `catalog`).
- Author/commit attribution: **Seongjin / 성진** (never a title).
- `gofmt` + `go vet` clean; `go test ./...` green before any push.
- Keep package layout mirroring the Python module layout where sensible
  (`expressions/`, `optimizer/`, `dialects/`, etc.).
