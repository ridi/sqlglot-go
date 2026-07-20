package optimizer_test

import (
	"testing"

	sqlglot "github.com/ridi-oss/sqlglot-go"
	"github.com/ridi-oss/sqlglot-go/dialects"
	exp "github.com/ridi-oss/sqlglot-go/expressions"
	"github.com/ridi-oss/sqlglot-go/optimizer"
	"github.com/ridi-oss/sqlglot-go/schema"
)

// Under MySQL lower_case_table_names=0 a search-path schema name is case-SENSITIVE and must be stamped
// in its original case (relation role), not folded like a column. Before Fix D the search-path
// identifier was normalized detached (no parent) and misread as a column, folding `App` to `app` — so a
// caller had to pre-fold the search path itself. Now sqlglot-go folds it role-aware, so a raw
// mixed-case search path resolves and stamps correctly, and a wrong-cased one fails closed.
func TestQualifyTablesSearchPathMySQLLctn0(t *testing.T) {
	d := dialects.MySQL()
	d.NormalizationStrategy = dialects.MySQLCaseSensitiveTableNames
	mapping, err := schema.NewMappingSchema(schema.M(
		"def", schema.M(
			"App", schema.M(
				"Users", schema.M("ID", "BIGINT"),
			),
		),
	), d, true)
	if err != nil {
		t.Fatalf("NewMappingSchema: %v", err)
	}

	qualify := func(sql string, searchPath []string) exp.Expression {
		t.Helper()
		expression, err := sqlglot.ParseOne(sql, "mysql")
		if err != nil {
			t.Fatalf("ParseOne: %v", err)
		}
		return optimizer.Qualify(expression, optimizer.QualifyOpts{
			Dialect:    d,
			Schema:     mapping,
			SearchPath: searchPath,
		})
	}
	soleTable := func(expression exp.Expression) exp.Expression {
		t.Helper()
		tables := expression.FindAll(exp.KindTable)
		if len(tables) != 1 {
			t.Fatalf("table count = %d, want 1", len(tables))
		}
		return tables[0]
	}

	// A RAW mixed-case search path resolves and stamps the schema in its original case.
	tbl := soleTable(qualify("SELECT ID FROM Users", []string{"App"}))
	if got := tbl.SchemaName(); got != "App" {
		t.Fatalf("db = %q, want \"App\" (lctn=0 preserves the search-path schema case)", got)
	}
	if got := tbl.Name(); got != "Users" {
		t.Fatalf("table = %q, want \"Users\"", got)
	}

	// A lowercased search path must NOT resolve the case-sensitive schema (fail closed, no stamp).
	miss := soleTable(qualify("SELECT ID FROM Users", []string{"app"}))
	if miss.SchemaName() != "" {
		t.Fatalf("lowercased search path \"app\" wrongly stamped db=%q (schema \"App\" is case-sensitive)", miss.SchemaName())
	}
}

// INFORMATION_SCHEMA is case-insensitive even when reached via the search path: an unqualified
// `FROM Tables` (search path includes information_schema) resolves to information_schema.tables under
// lctn=0, because the probe folds the table name in the candidate-schema context and the stamped
// identity is re-folded to match the schema key. (Regression for the search-path/info_schema probe miss.)
func TestQualifyTablesSearchPathInformationSchemaLctn0(t *testing.T) {
	d := dialects.MySQL()
	d.NormalizationStrategy = dialects.MySQLCaseSensitiveTableNames
	mapping, err := schema.NewMappingSchema(schema.M(
		"def", schema.M("information_schema", schema.M("TABLES", schema.M("TABLE_NAME", "VARCHAR"))),
	), d, true)
	if err != nil {
		t.Fatalf("NewMappingSchema: %v", err)
	}
	expr, err := sqlglot.ParseOne("SELECT TABLE_NAME FROM TABLES", "mysql")
	if err != nil {
		t.Fatalf("ParseOne: %v", err)
	}
	q := optimizer.Qualify(expr, optimizer.QualifyOpts{
		Dialect:    d,
		Schema:     mapping,
		SearchPath: []string{"Information_Schema"},
	})
	tables := q.FindAll(exp.KindTable)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}
	if got, gotDB := tables[0].Name(), tables[0].SchemaName(); got != "tables" || gotDB != "information_schema" {
		t.Fatalf("resolved %q.%q, want information_schema.tables (info_schema is case-insensitive via search path)", gotDB, got)
	}
}

// Passing an existing Identifier node as DB must not reparent or mutate the caller's node (upstream
// leaves it untouched). Regression for the throwaway-table reparenting.
func TestQualifyTablesDBIdentifierNotMutated(t *testing.T) {
	owner := exp.Table(exp.Args{"schema": exp.ToIdentifier("Owned")})
	dbID, _ := owner.Arg("schema").(exp.Expression)
	if dbID == nil {
		t.Fatal("setup: owner.db is not an identifier")
	}
	expr, err := sqlglot.ParseOne("SELECT * FROM t", "mysql")
	if err != nil {
		t.Fatalf("ParseOne: %v", err)
	}
	optimizer.QualifyTables(expr, dbID, nil, "mysql", false, nil, nil, nil)
	if dbID.Parent() != owner {
		t.Fatalf("db identifier was reparented away from its owner table")
	}
	if dbID.Name() != "Owned" {
		t.Fatalf("db identifier name was mutated to %q, want Owned", dbID.Name())
	}
}

// Under lctn=1/2 (mysql_case_insensitive) a search-path schema name folds, so a mixed-case raw search
// path still resolves the folded schema.
func TestQualifyTablesSearchPathMySQLLctn1(t *testing.T) {
	d := dialects.MySQL()
	d.NormalizationStrategy = dialects.MySQLCaseInsensitive
	mapping, err := schema.NewMappingSchema(schema.M(
		"def", schema.M("app", schema.M("users", schema.M("id", "BIGINT"))),
	), d, true)
	if err != nil {
		t.Fatalf("NewMappingSchema: %v", err)
	}
	expression, err := sqlglot.ParseOne("SELECT id FROM users", "mysql")
	if err != nil {
		t.Fatalf("ParseOne: %v", err)
	}
	qualified := optimizer.Qualify(expression, optimizer.QualifyOpts{
		Dialect:    d,
		Schema:     mapping,
		SearchPath: []string{"APP"}, // raw upper-case; folds to "app"
	})
	tables := qualified.FindAll(exp.KindTable)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}
	if got := tables[0].SchemaName(); got != "app" {
		t.Fatalf("db = %q, want \"app\" (lctn=1 folds the search-path schema)", got)
	}
}
