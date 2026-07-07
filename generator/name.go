package generator

import (
	"strings"

	"github.com/sjincho/sqlglot-go/expressions"
)

var sqlNameOverrides = map[expressions.Kind]string{
	expressions.KindPow:                "POWER",
	expressions.KindDateDiff:           "DATEDIFF",
	expressions.KindMD5:                "MD5",
	expressions.KindJSONExtract:        "JSON_EXTRACT",
	expressions.KindJSONExtractScalar:  "JSON_EXTRACT_SCALAR",
	expressions.KindJSONBExtract:       "JSONB_EXTRACT",
	expressions.KindJSONBExtractScalar: "JSONB_EXTRACT_SCALAR",
}

func (g *Generator) sqlName(kind expressions.Kind) string {
	if name, ok := sqlNameOverrides[kind]; ok {
		return name
	}
	return camelToSnake(expressions.ClassName(kind))
}

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}
