package parser

import "regexp"

// timeZoneRE ports TIME_ZONE_RE (parser.py:41 `re.compile(r":.*?[a-zA-Z\+\-]")`): a naive
// check for a time-zone suffix on a timestamp literal - a colon followed later by a letter
// or a sign, e.g. `... 05:00 Europe/Prague` or `... +05:00`. Presto's
// ZONE_AWARE_TIMESTAMP_CONSTRUCTOR path (parseType in parser.go) uses it to promote
// `TIMESTAMP '<zoned literal>'` to TIMESTAMPTZ.
var timeZoneRE = regexp.MustCompile(`:.*?[a-zA-Z+\-]`)

// Presto builds FUNCTION_PARSERS as the base table minus TRIM (parsers/presto.py:137:
// `FUNCTION_PARSERS = {k: v for k, v in parser.Parser.FUNCTION_PARSERS.items() if k != "TRIM"}`).
// Disabling TRIM here makes `TRIM(x)` fall through to the Anonymous-function path
// (functionParser resolution, dialect_parser_overrides.go:58-90) instead of building an
// exp.Trim node, matching upstream Presto.
func init() {
	registerDialectParserOverrides("presto", dialectParserOverrideSet{
		DisabledFunctionParsers: map[string]bool{"TRIM": true},
	})
}
