# Deviations from upstream sqlglot

sqlglot-go is a faithful ~1:1 port of **tobymao/sqlglot v30.12.0**. This file records every place the
port *intentionally* behaves differently from the Python original, so downstream consumers and future
porters know exactly where — and why — the two disagree. It complements the per-site code comments
(grep `divergence`/`Unlike upstream`) and the `ROADMAP.md` "known divergences" + "resolved-findings"
ledgers, which carry the fine detail.

Deviations are grouped by how *observable* they are. Only **§1 changes same-dialect parse→generate
output** vs upstream; everything else is either cross-dialect-only, output-preserving, or a
not-yet-ported scope boundary.

---

## 1. Behavioral deviations (same input → different output than upstream)

### 1.1 ASCII-only identifier case-folding  — *the one that changes same-dialect output*

**What upstream does:** `Dialect.normalize_identifier` folds unquoted identifiers with Python
`str.lower()` / `str.upper()`, and `Dialect.case_sensitive` tests with `str.isupper()`/`str.islower()`
— all **full-Unicode** (`.reference/sqlglot-v30.12.0/sqlglot/dialects/dialect.py:1042-1050,1055-1064`).
So upstream normalizes unquoted `CAFÉ` → `café` (it also lowercases the `É`).

**What sqlglot-go does:** folds **ASCII-only** — `A-Z`↔`a-z` (bytes `0x41-0x5A`/`0x61-0x7A`), leaving
every byte `≥ 0x80` untouched (`dialects/dialect.go` `asciiLower`/`asciiUpper` + `CaseSensitive`). So
unquoted `CAFÉ` → `cafÉ` (the ASCII `C,A,F` fold; `É` is left alone). `CaseSensitive` likewise treats
an identifier that differs only by non-ASCII case (e.g. `cafÉ`) as already-normalized, not
needing-quotes.

**Why we diverge (correctness):** upstream over-folds — it does not match the database it models. Real
engines case-fold identifiers as an **ASCII** operation on multibyte encodings:

- **PostgreSQL** — `downcase_identifier()` in `src/backend/parser/scansup.c`. Per the PG commit
  *"Don't downcase non-ascii identifier chars in multi-byte encoding"*: *"Long-standing code has called
  `tolower()` on identifier character bytes with the high bit set. This is clearly an error and produces
  junk output when the encoding is multi-byte. This patch therefore restricts this activity to cases
  where there is a character with the high bit set AND the encoding is single-byte."* On UTF-8
  (`server_encoding=UTF8`) it degenerates to plain ASCII casing — only `A-Z` fold; multibyte sequences
  pass through unchanged. Empirically verified: `CREATE TABLE t (CAFÉ int)` →
  `information_schema.columns.column_name` = `cafÉ`, not `café`.
  The docs (§4.1.1 *Identifiers and Key Words*) state only that *"unquoted names are always folded to
  lower case"*; the ASCII-only detail lives in the source above.

ASCII-only is **exact for UTF-8** (the dominant server_encoding) and a **safe under-fold** otherwise.
The port has no DB connection and cannot know the actual encoding/locale, so it does not chase
per-encoding/locale rules (e.g. a single-byte-encoding `tolower()` under a specific locale). The same
ASCII-only fold applies to every strategy that folds (Lowercase → PostgreSQL/base; CaseInsensitive →
Presto/Trino/Athena/Hive; and the currently-unused Uppercase/CaseInsensitiveUppercase).

**Why it matters:** a downstream consumer keys column-level security policy off the *normalized*
identifier. If sqlglot-go folds `É` but the backend does not, the normalized key resolves to a different
column than the database binds — a correctness/safety bug. Beyond that consumer, it is simply modeling
the dialect correctly.

**Scope of the change:** `dialects/dialect.go` only — `NormalizeIdentifier` (fold) and `CaseSensitive`
(needs-quoting test). Quoted-identifier handling is unchanged (quoted names are never folded, which is
correct). Regression test: `identifier_casefold_test.go` (root, an original non-ported test). Full
`go test ./...` stayed green — no existing test had encoded the old full-Unicode fold (fixtures are
ASCII), so the blast radius was zero.

**Upstream status:** this is a real upstream bug (the Go port faithfully mirrored it before this fix).
Worth an upstream issue/PR to sqlglot proposing ASCII-restricted folding for the LOWERCASE/UPPERCASE
strategies; until/unless upstream changes, this stays a deliberate divergence.

---

## 2. Cross-dialect-only deviations (never affect same-dialect round-trip)

The port's verified goal is **same-dialect round-trip** (read X → write X). Cross-dialect
transpilation (read X → write Y) is explicitly out of scope and only partially correct. These differ
from upstream only on the cross-dialect path:

- **Generator `TRANSFORMS` / `TYPE_MAPPING` not ported** for presto/trino/hive/athena (the
  parser+tokenizer side is ported for lineage; generators are skipped). Same-dialect functions
  round-trip via the base generator + `Anonymous` fallback. See `ROADMAP.md` "Athena support".
- **Functions left `Anonymous` instead of structured** where structuring would only matter for
  canonicalization/transpile: Presto/Trino `DATE_FORMAT`/`DATE_PARSE`/`TO_CHAR`/`DATE_TRUNC`/`REGEXP_*`/
  `LOCALTIME[STAMP]`, and `CONCAT_WS` (`parser/parser_functions.go`, `dialects/presto.go`). Lineage
  still sees their column args; same-dialect `.sql()` echoes them verbatim.

---

## 3. Output-preserving, Go-necessitated divergences (same output, internal difference)

Go's static typing / lack of metaclasses forces these; none change `.sql()` output:

- **Arg ordering:** `newNode` orders args by the per-Kind `argTypes` declaration order, not caller
  insertion order (Python dict preserves insertion). Equality/hash/find/walk are unaffected; the
  generator iterates `argTypes` order too. (`expressions/core.go`; `ROADMAP.md` known-divergences.)
- **`parseTable` fast-path skip** — a parse-order optimization divergence, same result
  (`parser/parser.go`).
- **`IsWrapper` uses the `truthy` helper** rather than Python's `v is None` (the port doesn't store nil
  args); equivalent for stored args.
- **`matchTextSeq` retreat has no logger** vs upstream's debug log (`parser/parser_stmt_common.go`).

---

## 4. Cosmetic AST-shape divergences (round-trip output identical, `.ToS()`/repr differs)

- **Comment bubbling:** a trailing comment can attach to a slightly different node than upstream; the
  comment still renders in the same textual position, so round-trip output matches (`parser` — see
  `ROADMAP.md`).
- **MySQL `PARTITION BY RANGE (YEAR|MONTH(col))`:** upstream wraps the 11 MySQL date functions in
  `TsOrDsToDate` in the parser and removes it in the generator; the port elides both consistently.
  Round-trips to identical SQL; only the incidental partition-expression arg shape differs
  (`generator/dialect_funcs.go`; capped in `fidelity_test.go` `maxASTDivergences`).

---

## 5. Deferred / fail-closed (parse to `exp.Command` where a future slice would structure)

These currently produce a raw-text `Command` (round-trips verbatim, fails closed) pending future work:

- **Postgres `CREATE FUNCTION ... AS $$...$$`** dollar-quoted (heredoc) bodies — `exp.Heredoc` unmodeled
  (`parser/parser_ddl.go`).
- **Hive CREATE-DDL property callbacks** (`CLUSTERED`/`EXTERNAL`/`LOCATION`/`ROW`/`STORED`/
  `TBLPROPERTIES`/`USING`) live in Hive's `PropertyParsers` overlay, deliberately kept out of the shared
  base registry until a paired parser+generator slice, so base/mysql/postgres/presto keep failing them
  closed (`parser/dialect_hive_overrides.go`).

---

## Not deviations (called out to avoid confusion)

- Where a reviewer flagged an "upstream bug," the port generally **keeps upstream's behavior 1:1** (e.g.
  a qualify_columns edge case, `optimizer/qualify_columns.go`) rather than silently "fixing" it — that
  is *faithfulness*, the opposite of a deviation. §1.1 is the deliberate exception, made only because it
  is a genuine correctness/safety issue against the modeled engine.
