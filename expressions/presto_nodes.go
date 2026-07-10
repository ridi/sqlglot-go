package expressions

// presto_nodes.go: builders for the Func-trait Kinds referenced by the Presto FUNCTIONS
// overlay (dialects/presto.go), plus the UnicodeString primitive node. One-file-per-slice,
// mirroring residual_tail.go. Each Kind's arg_types/traits/className rows live in kinds.go;
// the builders here mirror the functions.go one-liner style (return newNode(KindX, args)).

// AnyValue ports exp.AnyValue (expressions/aggregate.py:17): ANY_VALUE(x) - returns an
// arbitrary value from the group.
func AnyValue(args Args) Expression { return newNode(KindAnyValue, args) }

// ApproxQuantile ports exp.ApproxQuantile (expressions/aggregate.py:234): Presto's
// APPROX_PERCENTILE(x, [w,] q, [accuracy]).
func ApproxQuantile(args Args) Expression { return newNode(KindApproxQuantile, args) }

// ArrayUniqueAgg ports exp.ArrayUniqueAgg (expressions/aggregate.py:83): Presto's
// SET_AGG(x) - array of distinct aggregated values.
func ArrayUniqueAgg(args Args) Expression { return newNode(KindArrayUniqueAgg, args) }

// DayOfWeekIso ports exp.DayOfWeekIso (expressions/temporal.py:217): Presto's DOW_ISO /
// DAY_OF_WEEK ISO variant.
func DayOfWeekIso(args Args) Expression { return newNode(KindDayOfWeekIso, args) }

// Decode ports exp.Decode (expressions/string.py:285): DECODE(x, charset[, replace]).
func Decode(args Args) Expression { return newNode(KindDecode, args) }

// Encode ports exp.Encode (expressions/string.py:289): ENCODE(x, charset).
func Encode(args Args) Expression { return newNode(KindEncode, args) }

// JSONFormat ports exp.JSONFormat (expressions/json.py:145): JSON_FORMAT(x).
func JSONFormat(args Args) Expression { return newNode(KindJSONFormat, args) }

// Levenshtein ports exp.Levenshtein (expressions/string.py:74): LEVENSHTEIN_DISTANCE(a, b).
func Levenshtein(args Args) Expression { return newNode(KindLevenshtein, args) }

// MD5Digest ports exp.MD5Digest (expressions/string.py:540, is_var_len_args=True): the
// binary-returning MD5(x) used by Presto (distinct from the hex-string exp.MD5).
func MD5Digest(args Args) Expression { return newNode(KindMD5Digest, args) }

// SHA2 ports exp.SHA2 (expressions/string.py:562): SHA256/SHA512 via SHA2(x, length).
func SHA2(args Args) Expression { return newNode(KindSHA2, args) }

// StrToMap ports exp.StrToMap (expressions/string.py:203): SPLIT_TO_MAP(x, kd, vd).
func StrToMap(args Args) Expression { return newNode(KindStrToMap, args) }

// TimeToUnix ports exp.TimeToUnix (expressions/temporal.py:484): TO_UNIXTIME(x).
func TimeToUnix(args Args) Expression { return newNode(KindTimeToUnix, args) }

// UnixToTime ports exp.UnixToTime (expressions/temporal.py:532): FROM_UNIXTIME(x[, ...]).
func UnixToTime(args Args) Expression { return newNode(KindUnixToTime, args) }

// Unhex ports exp.Unhex (expressions/string.py:405): FROM_HEX(x).
func Unhex(args Args) Expression { return newNode(KindUnhex, args) }

// ArraySlice ports exp.ArraySlice (expressions/array.py:85): SLICE(x, start[, end]).
func ArraySlice(args Args) Expression { return newNode(KindArraySlice, args) }

// CurrentTimestamp ports exp.CurrentTimestamp (expressions/temporal.py:37): NOW() /
// CURRENT_TIMESTAMP.
func CurrentTimestamp(args Args) Expression { return newNode(KindCurrentTimestamp, args) }

// UnicodeString ports exp.UnicodeString (expressions/query.py:494, is_primitive=True): a
// `U&'...'` unicode string literal. Unlike the Func Kinds above it is a plain Condition
// (no TraitFunc), so it has no functionFallbackSQL path - the generator supplies a
// dedicated dispatch method (mirrors National's `N'...'`).
func UnicodeString(args Args) Expression { return newNode(KindUnicodeString, args) }
