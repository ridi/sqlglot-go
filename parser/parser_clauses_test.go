package parser_test

import (
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	"github.com/sjincho/sqlglot-go/dialects"
	sqlerrors "github.com/sjincho/sqlglot-go/errors"
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/parser"
)

func TestGroupByVariants(t *testing.T) {
	expression := parseOne(t, "SELECT a GROUP BY a")
	group := expression.Arg("group").(exp.Expression)
	if group.Kind() != exp.KindGroup || len(group.Expressions()) != 1 {
		t.Fatalf("GROUP BY shape = %s, want one expression", group.ToS())
	}

	expression = parseOne(t, "SELECT a GROUP BY CUBE(a)")
	group = expression.Arg("group").(exp.Expression)
	if cube := expressionsForArg(group, "cube"); len(cube) != 1 || cube[0].Kind() != exp.KindCube {
		t.Fatalf("GROUP BY CUBE shape = %s, want one Cube", group.ToS())
	}

	expression = parseOne(t, "SELECT a GROUP BY ROLLUP(a)")
	group = expression.Arg("group").(exp.Expression)
	if rollup := expressionsForArg(group, "rollup"); len(rollup) != 1 || rollup[0].Kind() != exp.KindRollup {
		t.Fatalf("GROUP BY ROLLUP shape = %s, want one Rollup", group.ToS())
	}

	expression = parseOne(t, "SELECT a GROUP BY GROUPING SETS((a),(b))")
	group = expression.Arg("group").(exp.Expression)
	sets := expressionsForArg(group, "grouping_sets")
	if len(sets) != 1 || sets[0].Kind() != exp.KindGroupingSets || len(sets[0].Expressions()) != 2 {
		t.Fatalf("GROUPING SETS shape = %s, want one GroupingSets with two entries", group.ToS())
	}
}

func TestQueryClauses(t *testing.T) {
	expression := parseOne(t, "SELECT a HAVING a > 1")
	if having, ok := expression.Arg("having").(exp.Expression); !ok || having.Kind() != exp.KindHaving {
		t.Fatalf("having = %#v, want Having", expression.Arg("having"))
	}

	expression = parseOne(t, "SELECT a QUALIFY a > 1")
	if qualify, ok := expression.Arg("qualify").(exp.Expression); !ok || qualify.Kind() != exp.KindQualify {
		t.Fatalf("qualify = %#v, want Qualify", expression.Arg("qualify"))
	}

	expression = parseOne(t, "SELECT a ORDER BY a DESC")
	order := expression.Arg("order").(exp.Expression)
	ordered := order.Expressions()[0]
	if ordered.Kind() != exp.KindOrdered || ordered.Arg("desc") != true || ordered.Arg("nulls_first") != false {
		t.Fatalf("ordered = %s, want desc nulls_last", ordered.ToS())
	}

	expression = parseOne(t, "SELECT a LIMIT 5 OFFSET 2")
	if limit, ok := expression.Arg("limit").(exp.Expression); !ok || limit.Kind() != exp.KindLimit {
		t.Fatalf("limit = %#v, want Limit", expression.Arg("limit"))
	}
	if offset, ok := expression.Arg("offset").(exp.Expression); !ok || offset.Kind() != exp.KindOffset {
		t.Fatalf("offset = %#v, want Offset", expression.Arg("offset"))
	}

	expression = parseOne(t, "SELECT DISTINCT ON (a) a FROM t")
	distinct := expression.Arg("distinct").(exp.Expression)
	if distinct.Kind() != exp.KindDistinct || distinct.Arg("on") == nil {
		t.Fatalf("distinct = %s, want DISTINCT ON", distinct.ToS())
	}
}

// TestLimitPercentModRetreat locks in _parse_limit's `exp.Mod` retreat (parser.py:5576-5579):
// `LIMIT x%` must not parse the count as a Mod term. Real forms are unaffected; the
// never-valid `LIMIT 10 % 3` must error on the trailing operand rather than build Mod(10, 3).
func TestLimitPercentModRetreat(t *testing.T) {
	for _, sql := range []string{"SELECT a LIMIT 10", "SELECT a LIMIT 10 PERCENT", "SELECT a LIMIT 10%"} {
		limit, ok := parseOne(t, sql).Arg("limit").(exp.Expression)
		if !ok || limit.Kind() != exp.KindLimit {
			t.Fatalf("%q: limit = %#v, want Limit", sql, limit)
		}
		if expr := limit.Expr(); expr != nil && expr.Kind() == exp.KindMod {
			t.Fatalf("%q: limit count parsed as Mod, want a plain count:\n%s", sql, limit.ToS())
		}
	}
	if _, err := sqlglot.ParseOne("SELECT a LIMIT 10 % 3", ""); err == nil {
		t.Fatal("LIMIT 10 % 3 should error on the trailing operand (upstream parity)")
	}
}

func TestWindowAndFilterClauses(t *testing.T) {
	expression := parseOne(t, "SELECT SUM(x) OVER (PARTITION BY a ORDER BY b)")
	window := expression.Expressions()[0]
	if window.Kind() != exp.KindWindow {
		t.Fatalf("projection kind = %v, want Window", window.Kind())
	}
	if partitionBy := expressionsForArg(window, "partition_by"); len(partitionBy) != 1 {
		t.Fatalf("partition_by count = %d, want 1", len(partitionBy))
	}
	if order, ok := window.Arg("order").(exp.Expression); !ok || order.Kind() != exp.KindOrder {
		t.Fatalf("window order = %#v, want Order", window.Arg("order"))
	}

	expression = parseOne(t, "SELECT SUM(x) FILTER (WHERE a > 0)")
	filter := expression.Expressions()[0]
	if filter.Kind() != exp.KindFilter {
		t.Fatalf("projection kind = %v, want Filter", filter.Kind())
	}

	expression = parseOne(t, "SELECT a WINDOW w AS (PARTITION BY x)")
	windows := expressionsForArg(expression, "windows")
	if len(windows) != 1 || windows[0].Kind() != exp.KindWindow {
		t.Fatalf("windows = %#v, want one Window", windows)
	}
}

func TestDuplicateWhereIgnoreKeepsLast(t *testing.T) {
	d := dialects.Base()
	toks, err := d.NewTokenizer().Tokenize("SELECT a WHERE x = 1 WHERE y = 2")
	if err != nil {
		t.Fatalf("Tokenize error: %v", err)
	}
	p := parser.NewWithErrorLevel(d, sqlerrors.IGNORE)
	expressions, err := p.Parse(toks, "SELECT a WHERE x = 1 WHERE y = 2")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(expressions) != 1 || expressions[0] == nil {
		t.Fatalf("expressions = %#v, want one expression", expressions)
	}
	where := expressions[0].Arg("where").(exp.Expression)
	if where.Find(exp.KindColumn).Name() != "y" {
		t.Fatalf("WHERE kept %s, want last duplicate y", where.ToS())
	}
}
