package dialects_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	sqlglot "github.com/sjincho/sqlglot-go"
	"github.com/sjincho/sqlglot-go/generator"
)

type identityCase struct {
	name           string
	dialect        string
	sql            string
	want           string
	deferredReason string
	category       string
}

func validateIdentity(t *testing.T, dialect, sql string, want ...string) {
	t.Helper()
	expression, err := sqlglot.ParseOne(sql, dialect)
	if err != nil {
		t.Fatalf("ParseOne(%q, %q): %v", sql, dialect, err)
	}
	got, err := sqlglot.Generate(expression, dialect, generator.Options{})
	if err != nil {
		t.Fatalf("Generate(%q, %q): %v", sql, dialect, err)
	}
	expected := sql
	if len(want) > 0 {
		expected = want[0]
	}
	if got != expected {
		t.Fatalf("identity(%q, %q) = %q, want %q", dialect, sql, got, expected)
	}
}

func runIdentityCases(t *testing.T, label string, cases []identityCase) (int, map[string]int) {
	t.Helper()
	pass := 0
	deferred := map[string]int{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.deferredReason != "" {
				category := tc.category
				if category == "" {
					category = "deferred"
				}
				deferred[category]++
				t.Skipf("deferred: %s", tc.deferredReason)
				return
			}
			if tc.want == "" {
				validateIdentity(t, tc.dialect, tc.sql)
			} else {
				validateIdentity(t, tc.dialect, tc.sql, tc.want)
			}
			pass++
		})
	}
	t.Logf("%s: pass=%d deferred=%s", label, pass, formatDeferredCounts(deferred))
	return pass, deferred
}

func formatDeferredCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func boolPtr(value bool) *bool { return &value }
