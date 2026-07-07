package optimizer_test

import (
	"fmt"
	"strings"
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/generator"
	"github.com/sjincho/sqlglot-go/optimizer"
)

func TestQualifyTablesDirect(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		db      any
		catalog any
		want    string
	}{
		{
			name:    "pivot cte",
			sql:     "WITH cte AS (SELECT * FROM t) SELECT * FROM cte PIVOT(SUM(c) FOR v IN ('x', 'y'))",
			db:      "db",
			catalog: "catalog",
			want:    "WITH cte AS (SELECT * FROM catalog.db.t AS t) SELECT * FROM cte AS cte PIVOT(SUM(c) FOR v IN ('x', 'y')) AS _0",
		},
		{
			name:    "pivot cte explicit alias",
			sql:     "WITH cte AS (SELECT * FROM t) SELECT * FROM cte PIVOT(SUM(c) FOR v IN ('x', 'y')) AS pivot_alias",
			db:      "db",
			catalog: "catalog",
			want:    "WITH cte AS (SELECT * FROM catalog.db.t AS t) SELECT * FROM cte AS cte PIVOT(SUM(c) FOR v IN ('x', 'y')) AS pivot_alias",
		},
		{
			name:    "catalog without db",
			sql:     "select a from b",
			catalog: "catalog",
			want:    "SELECT a FROM b AS b",
		},
		{
			name: "quoted db",
			sql:  "select a from b",
			db:   `"DB"`,
			want: `SELECT a FROM "DB".b AS b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expression, err := sqlglot.ParseOne(tt.sql, "")
			if err != nil {
				t.Fatalf("ParseOne: %v", err)
			}
			result := optimizer.QualifyTables(expression, tt.db, tt.catalog, "", false, nil)
			got, err := sqlglot.Generate(result, "", generator.Options{})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got != tt.want {
				t.Fatalf("QualifyTables() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQualifyTablesFixtures(t *testing.T) {
	for i, pair := range loadSQLFixturePairs(t, "qualify_tables.sql") {
		title := pair.Meta["title"]
		if title == "" {
			title = fmt.Sprintf("%d, %s", i+1, pair.SQL)
		}
		t.Run(title, func(t *testing.T) {
			if !dialectInScope(pair.Meta) {
				t.Skipf("deferred dialect: %s", pair.Meta["dialect"])
			}
			if reason := deferredQualifyTablesFixture(pair); reason != "" {
				t.Skipf("deferred: %s", reason)
			}

			dialect := pair.Meta["dialect"]
			expression, err := sqlglot.ParseOne(pair.SQL, dialect)
			if err != nil {
				t.Fatalf("ParseOne: %v", err)
			}
			result := optimizer.QualifyTables(expression, "db", "c", dialect, stringToBool(pair.Meta["canonicalize_table_aliases"]), nil)
			got, err := sqlglot.Generate(result, dialect, generator.Options{})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got != pair.Expected {
				t.Fatalf("QualifyTables() = %q, want %q", got, pair.Expected)
			}
		})
	}
}

func deferredQualifyTablesFixture(pair sqlFixturePair) string {
	if pair.Meta["dialect"] == "postgres" {
		return "postgres function table-source parser/dialect behavior is slice 5"
	}
	switch pair.Meta["title"] {
	case "nested joins":
		return "parser does not yet accept chained join ON syntax"
	case "table with ordinality":
		return "parser does not yet accept function table sources with ordinality"
	case "alter table":
		return "ALTER TABLE parser support is deferred"
	case "tablesample":
		return "TABLESAMPLE parser support is deferred"
	}
	if strings.HasPrefix(pair.SQL, "COPY INTO ") {
		return "COPY INTO parser support is deferred"
	}
	return ""
}

func TestQualifyTablesCopiesTypedAliasColumns(t *testing.T) {
	original := exp.ColumnDef(exp.Args{
		"this": exp.ToIdentifier("rank", true),
		"kind": exp.DTypeInt.IntoExpr(nil),
	})
	table := exp.Table(exp.Args{
		"this": exp.Anonymous(exp.Args{
			"this":        "JSON_TO_RECORDSET",
			"expressions": []exp.Expression{exp.Column_("z", nil, nil, nil, nil)},
		}),
		"alias": exp.TableAlias(exp.Args{
			"this":    exp.ToIdentifier("y"),
			"columns": []exp.Expression{original},
		}),
	})
	expression := exp.Select(exp.Args{
		"expressions": []exp.Expression{exp.Star(exp.Args{})},
		"from_":       exp.From(exp.Args{"this": table}),
	})

	optimizer.QualifyTables(expression, nil, nil, "", true, nil)

	alias := expression.Find(exp.KindTable).Arg("alias").(exp.Expression)
	newColumn := alias.(*exp.Node).ExpressionsFor("columns")[0]
	if newColumn.Kind() != exp.KindColumnDef {
		t.Fatalf("alias column kind = %v, want ColumnDef", newColumn.Kind())
	}
	if original == newColumn {
		t.Fatalf("alias column was reused, want a copy")
	}
	originalSQL, err := sqlglot.Generate(original, "", generator.Options{})
	if err != nil {
		t.Fatalf("Generate original: %v", err)
	}
	newSQL, err := sqlglot.Generate(newColumn, "", generator.Options{})
	if err != nil {
		t.Fatalf("Generate copied: %v", err)
	}
	if originalSQL != newSQL {
		t.Fatalf("copied alias column SQL = %q, want %q", newSQL, originalSQL)
	}
}
