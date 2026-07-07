package expressions

type Kind int

const (
	KindExpression Kind = iota
	KindColumn
	KindLiteral
	KindIdentifier
	KindStar
	KindAlias
	KindAliases
	KindDot
	KindNull
	KindBoolean
	KindVar
	KindParen
	KindNeg
	KindNot
	KindAdd
	KindSub
	KindMul
	KindDiv
	KindMod
	KindEQ
	KindNEQ
	KindNullSafeEQ
	KindGT
	KindGTE
	KindLT
	KindLTE
	KindAnd
	KindOr
	KindBitwiseAnd
	KindBitwiseOr
	KindBitwiseXor
	KindDPipe
	KindBetween
	KindIs
	KindLike
	KindILike
	KindIn
	KindSelect
	KindFrom
	KindJoin
	KindTable
	KindTableAlias
	KindWhere
	KindGroup
	KindOrder
	KindLimit
	KindOffset
	KindHint
	KindBlock
	KindPlaceholder
	KindTuple
	KindWith
	KindCTE
	KindRecursiveWithSearch
	KindUnion
	KindExcept
	KindIntersect
	KindSubquery
	KindHaving
	KindQualify
	KindCube
	KindRollup
	KindGroupingSets
	KindOrdered
	KindDistinct
	KindWindow
	KindWindowSpec
	KindFilter
	KindLimitOptions
	KindFetch
	KindCase
	KindIf
	KindExists
	KindAny
	KindAll
	KindNullSafeNEQ
	KindAnonymous
	KindAbs
	KindAvg
	KindSum
	KindSqrt
	KindLn
	KindExp
	KindMin
	KindMax
	KindRound
	KindLog
	KindPow
	KindStddev
	KindStddevPop
	KindStddevSamp
	KindVariance
	KindVariancePop
	KindDay
	KindMonth
	KindYear
	KindQuarter
	KindApproxDistinct
	KindHll
	KindCountIf
	KindQuantile
	KindCount
	KindCoalesce
	KindGreatest
	KindLeast
)

type Trait uint32

const (
	TraitCondition Trait = 1 << iota
	TraitPredicate
	TraitBinary
	TraitConnector
	TraitFunc
	TraitAggFunc
	TraitUnary
	TraitQuery
)

type argSpec struct {
	Key      string
	Required bool
}

var defaultArgTypes = []argSpec{{Key: "this", Required: true}}

var queryModifierArgTypes = []argSpec{
	{"match", false},
	{"laterals", false},
	{"joins", false},
	{"connect", false},
	{"pivots", false},
	{"prewhere", false},
	{"where", false},
	{"group", false},
	{"having", false},
	{"qualify", false},
	{"windows", false},
	{"distribute", false},
	{"sort", false},
	{"cluster", false},
	{"order", false},
	{"limit", false},
	{"offset", false},
	{"locks", false},
	{"sample", false},
	{"settings", false},
	{"format", false},
	{"options", false},
	{"for_", false},
}

func withQueryModifiers(prefix ...argSpec) []argSpec {
	out := append([]argSpec{}, prefix...)
	return append(out, queryModifierArgTypes...)
}

var argTypes = map[Kind][]argSpec{
	KindExpression:          defaultArgTypes,
	KindColumn:              {{"this", true}, {"table", false}, {"db", false}, {"catalog", false}, {"join_mark", false}},
	KindLiteral:             {{"this", true}, {"is_string", true}},
	KindIdentifier:          {{"this", true}, {"quoted", false}, {"global_", false}, {"temporary", false}},
	KindStar:                {{"except_", false}, {"replace", false}, {"rename", false}, {"ilike", false}},
	KindAlias:               {{"this", true}, {"alias", false}},
	KindAliases:             {{"this", true}, {"expressions", true}},
	KindDot:                 {{"this", true}, {"expression", true}},
	KindNull:                {},
	KindBoolean:             defaultArgTypes,
	KindVar:                 defaultArgTypes,
	KindParen:               defaultArgTypes,
	KindNeg:                 defaultArgTypes,
	KindNot:                 defaultArgTypes,
	KindAdd:                 {{"this", true}, {"expression", true}},
	KindSub:                 {{"this", true}, {"expression", true}},
	KindMul:                 {{"this", true}, {"expression", true}},
	KindDiv:                 {{"this", true}, {"expression", true}, {"typed", false}, {"safe", false}},
	KindMod:                 {{"this", true}, {"expression", true}},
	KindEQ:                  {{"this", true}, {"expression", true}},
	KindNEQ:                 {{"this", true}, {"expression", true}},
	KindNullSafeEQ:          {{"this", true}, {"expression", true}},
	KindGT:                  {{"this", true}, {"expression", true}},
	KindGTE:                 {{"this", true}, {"expression", true}},
	KindLT:                  {{"this", true}, {"expression", true}},
	KindLTE:                 {{"this", true}, {"expression", true}},
	KindAnd:                 {{"this", true}, {"expression", true}},
	KindOr:                  {{"this", true}, {"expression", true}},
	KindBitwiseAnd:          {{"this", true}, {"expression", true}, {"padside", false}},
	KindBitwiseOr:           {{"this", true}, {"expression", true}, {"padside", false}},
	KindBitwiseXor:          {{"this", true}, {"expression", true}, {"padside", false}},
	KindDPipe:               {{"this", true}, {"expression", true}, {"safe", false}},
	KindBetween:             {{"this", true}, {"low", true}, {"high", true}, {"symmetric", false}},
	KindIs:                  {{"this", true}, {"expression", true}},
	KindLike:                {{"this", true}, {"expression", true}, {"negate", false}},
	KindILike:               {{"this", true}, {"expression", true}, {"negate", false}},
	KindIn:                  {{"this", true}, {"expressions", false}, {"query", false}, {"unnest", false}, {"field", false}, {"is_global", false}},
	KindSelect:              withQueryModifiers(argSpec{"with_", false}, argSpec{"kind", false}, argSpec{"expressions", false}, argSpec{"hint", false}, argSpec{"distinct", false}, argSpec{"into", false}, argSpec{"from_", false}, argSpec{"operation_modifiers", false}, argSpec{"exclude", false}),
	KindFrom:                defaultArgTypes,
	KindJoin:                {{"this", true}, {"on", false}, {"side", false}, {"kind", false}, {"using", false}, {"method", false}, {"global_", false}, {"hint", false}, {"match_condition", false}, {"directed", false}, {"expressions", false}, {"pivots", false}},
	KindTable:               {{"this", false}, {"alias", false}, {"db", false}, {"catalog", false}, {"laterals", false}, {"joins", false}, {"pivots", false}, {"hints", false}, {"system_time", false}, {"version", false}, {"format", false}, {"pattern", false}, {"ordinality", false}, {"when", false}, {"only", false}, {"partition", false}, {"changes", false}, {"rows_from", false}, {"sample", false}, {"indexed", false}},
	KindTableAlias:          {{"this", false}, {"columns", false}},
	KindWhere:               defaultArgTypes,
	KindGroup:               {{"expressions", false}, {"grouping_sets", false}, {"cube", false}, {"rollup", false}, {"totals", false}, {"all", false}},
	KindOrder:               {{"this", false}, {"expressions", true}, {"siblings", false}},
	KindLimit:               {{"this", false}, {"expression", true}, {"offset", false}, {"limit_options", false}, {"expressions", false}},
	KindOffset:              {{"this", false}, {"expression", true}, {"expressions", false}},
	KindHint:                {{"expressions", true}},
	KindBlock:               {{"expressions", true}},
	KindPlaceholder:         {{"this", false}, {"kind", false}, {"widget", false}, {"jdbc", false}},
	KindTuple:               {{"expressions", false}},
	KindWith:                {{"expressions", true}, {"recursive", false}, {"search", false}},
	KindCTE:                 {{"this", true}, {"alias", true}, {"scalar", false}, {"materialized", false}, {"key_expressions", false}},
	KindRecursiveWithSearch: {{"kind", true}, {"this", true}, {"expression", true}, {"using", false}},
	KindUnion:               withQueryModifiers(argSpec{"with_", false}, argSpec{"this", true}, argSpec{"expression", true}, argSpec{"distinct", false}, argSpec{"by_name", false}, argSpec{"side", false}, argSpec{"kind", false}, argSpec{"on", false}),
	KindExcept:              withQueryModifiers(argSpec{"with_", false}, argSpec{"this", true}, argSpec{"expression", true}, argSpec{"distinct", false}, argSpec{"by_name", false}, argSpec{"side", false}, argSpec{"kind", false}, argSpec{"on", false}),
	KindIntersect:           withQueryModifiers(argSpec{"with_", false}, argSpec{"this", true}, argSpec{"expression", true}, argSpec{"distinct", false}, argSpec{"by_name", false}, argSpec{"side", false}, argSpec{"kind", false}, argSpec{"on", false}),
	KindSubquery:            withQueryModifiers(argSpec{"this", true}, argSpec{"alias", false}, argSpec{"with_", false}),
	KindHaving:              {{"this", true}},
	KindQualify:             {{"this", true}},
	KindCube:                {{"expressions", false}},
	KindRollup:              {{"expressions", false}},
	KindGroupingSets:        {{"expressions", true}},
	KindOrdered:             {{"this", true}, {"desc", false}, {"nulls_first", true}, {"with_fill", false}},
	KindDistinct:            {{"expressions", false}, {"on", false}},
	KindWindow:              {{"this", true}, {"partition_by", false}, {"order", false}, {"spec", false}, {"alias", false}, {"over", false}, {"first", false}},
	KindWindowSpec:          {{"kind", false}, {"start", false}, {"start_side", false}, {"end", false}, {"end_side", false}, {"exclude", false}},
	KindFilter:              {{"this", true}, {"expression", true}},
	KindLimitOptions:        {{"percent", false}, {"rows", false}, {"with_ties", false}},
	KindFetch:               {{"direction", false}, {"count", false}, {"limit_options", false}},
	KindCase:                {{"this", false}, {"ifs", true}, {"default", false}},
	KindIf:                  {{"this", true}, {"true", true}, {"false", false}},
	KindExists:              {{"this", true}, {"expression", false}},
	KindAny:                 {{"this", true}},
	KindAll:                 {{"this", true}},
	KindNullSafeNEQ:         {{"this", true}, {"expression", true}},
	KindAnonymous:           {{"this", true}, {"expressions", false}},
	KindAbs:                 defaultArgTypes,
	KindAvg:                 defaultArgTypes,
	KindSum:                 defaultArgTypes,
	KindSqrt:                defaultArgTypes,
	KindLn:                  defaultArgTypes,
	KindExp:                 defaultArgTypes,
	KindMin:                 {{"this", true}, {"expressions", false}},
	KindMax:                 {{"this", true}, {"expressions", false}},
	KindRound:               {{"this", true}, {"decimals", false}, {"truncate", false}, {"casts_non_integer_decimals", false}},
	KindLog:                 {{"this", true}, {"expression", false}},
	KindPow:                 {{"this", true}, {"expression", true}},
	KindStddev:              defaultArgTypes,
	KindStddevPop:           defaultArgTypes,
	KindStddevSamp:          defaultArgTypes,
	KindVariance:            defaultArgTypes,
	KindVariancePop:         defaultArgTypes,
	KindDay:                 defaultArgTypes,
	KindMonth:               defaultArgTypes,
	KindYear:                defaultArgTypes,
	KindQuarter:             defaultArgTypes,
	KindApproxDistinct:      {{"this", true}, {"accuracy", false}},
	KindHll:                 {{"this", true}, {"expressions", false}},
	KindCountIf:             defaultArgTypes,
	KindQuantile:            {{"this", true}, {"quantile", true}},
	KindCount:               {{"this", false}, {"expressions", false}, {"big_int", false}},
	KindCoalesce:            {{"this", true}, {"expressions", false}, {"is_nvl", false}, {"is_null", false}},
	KindGreatest:            {{"this", true}, {"expressions", false}, {"ignore_nulls", true}},
	KindLeast:               {{"this", true}, {"expressions", false}, {"ignore_nulls", true}},
}

var traitsOf = map[Kind]Trait{
	KindColumn:         TraitCondition,
	KindLiteral:        TraitCondition,
	KindNull:           TraitCondition,
	KindBoolean:        TraitCondition,
	KindPlaceholder:    TraitCondition,
	KindDot:            TraitCondition | TraitBinary,
	KindAdd:            TraitCondition | TraitBinary,
	KindSub:            TraitCondition | TraitBinary,
	KindMul:            TraitCondition | TraitBinary,
	KindDiv:            TraitCondition | TraitBinary,
	KindMod:            TraitCondition | TraitBinary,
	KindEQ:             TraitCondition | TraitBinary | TraitPredicate,
	KindNEQ:            TraitCondition | TraitBinary | TraitPredicate,
	KindNullSafeEQ:     TraitCondition | TraitBinary | TraitPredicate,
	KindGT:             TraitCondition | TraitBinary | TraitPredicate,
	KindGTE:            TraitCondition | TraitBinary | TraitPredicate,
	KindLT:             TraitCondition | TraitBinary | TraitPredicate,
	KindLTE:            TraitCondition | TraitBinary | TraitPredicate,
	KindLike:           TraitCondition | TraitBinary | TraitPredicate,
	KindILike:          TraitCondition | TraitBinary | TraitPredicate,
	KindIs:             TraitCondition | TraitBinary | TraitPredicate,
	KindBetween:        TraitPredicate,
	KindIn:             TraitPredicate,
	KindAnd:            TraitCondition | TraitBinary | TraitConnector | TraitFunc,
	KindOr:             TraitCondition | TraitBinary | TraitConnector | TraitFunc,
	KindBitwiseAnd:     TraitCondition | TraitBinary,
	KindBitwiseOr:      TraitCondition | TraitBinary,
	KindBitwiseXor:     TraitCondition | TraitBinary,
	KindDPipe:          TraitCondition | TraitBinary,
	KindParen:          TraitCondition | TraitUnary,
	KindNeg:            TraitCondition | TraitUnary,
	KindNot:            TraitCondition | TraitUnary,
	KindSelect:         TraitQuery,
	KindUnion:          TraitQuery,
	KindExcept:         TraitQuery,
	KindIntersect:      TraitQuery,
	KindSubquery:       TraitQuery,
	KindNullSafeNEQ:    TraitCondition | TraitBinary | TraitPredicate,
	KindCase:           TraitCondition | TraitFunc,
	KindIf:             TraitCondition | TraitFunc,
	KindExists:         TraitCondition | TraitPredicate | TraitFunc,
	KindAny:            TraitCondition | TraitPredicate,
	KindAll:            TraitCondition | TraitPredicate,
	KindAnonymous:      TraitCondition | TraitFunc,
	KindAbs:            TraitCondition | TraitFunc,
	KindSqrt:           TraitCondition | TraitFunc,
	KindLn:             TraitCondition | TraitFunc,
	KindExp:            TraitCondition | TraitFunc,
	KindRound:          TraitCondition | TraitFunc,
	KindLog:            TraitCondition | TraitFunc,
	KindPow:            TraitCondition | TraitFunc,
	KindDay:            TraitCondition | TraitFunc,
	KindMonth:          TraitCondition | TraitFunc,
	KindYear:           TraitCondition | TraitFunc,
	KindQuarter:        TraitCondition | TraitFunc,
	KindCoalesce:       TraitCondition | TraitFunc,
	KindGreatest:       TraitCondition | TraitFunc,
	KindLeast:          TraitCondition | TraitFunc,
	KindWindow:         TraitCondition,
	KindAvg:            TraitCondition | TraitFunc | TraitAggFunc,
	KindSum:            TraitCondition | TraitFunc | TraitAggFunc,
	KindMin:            TraitCondition | TraitFunc | TraitAggFunc,
	KindMax:            TraitCondition | TraitFunc | TraitAggFunc,
	KindStddev:         TraitCondition | TraitFunc | TraitAggFunc,
	KindStddevPop:      TraitCondition | TraitFunc | TraitAggFunc,
	KindStddevSamp:     TraitCondition | TraitFunc | TraitAggFunc,
	KindVariance:       TraitCondition | TraitFunc | TraitAggFunc,
	KindVariancePop:    TraitCondition | TraitFunc | TraitAggFunc,
	KindApproxDistinct: TraitCondition | TraitFunc | TraitAggFunc,
	KindHll:            TraitCondition | TraitFunc | TraitAggFunc,
	KindCountIf:        TraitCondition | TraitFunc | TraitAggFunc,
	KindQuantile:       TraitCondition | TraitFunc | TraitAggFunc,
	KindCount:          TraitCondition | TraitFunc | TraitAggFunc,
}

var primitive = map[Kind]bool{
	KindLiteral:    true,
	KindIdentifier: true,
	KindVar:        true,
	KindBoolean:    true,
}

var hashRaw = map[Kind]bool{
	KindLiteral:    true,
	KindIdentifier: true,
}

var className = map[Kind]string{
	KindExpression:          "Expression",
	KindColumn:              "Column",
	KindLiteral:             "Literal",
	KindIdentifier:          "Identifier",
	KindStar:                "Star",
	KindAlias:               "Alias",
	KindAliases:             "Aliases",
	KindDot:                 "Dot",
	KindNull:                "Null",
	KindBoolean:             "Boolean",
	KindVar:                 "Var",
	KindParen:               "Paren",
	KindNeg:                 "Neg",
	KindNot:                 "Not",
	KindAdd:                 "Add",
	KindSub:                 "Sub",
	KindMul:                 "Mul",
	KindDiv:                 "Div",
	KindMod:                 "Mod",
	KindEQ:                  "EQ",
	KindNEQ:                 "NEQ",
	KindNullSafeEQ:          "NullSafeEQ",
	KindGT:                  "GT",
	KindGTE:                 "GTE",
	KindLT:                  "LT",
	KindLTE:                 "LTE",
	KindAnd:                 "And",
	KindOr:                  "Or",
	KindBitwiseAnd:          "BitwiseAnd",
	KindBitwiseOr:           "BitwiseOr",
	KindBitwiseXor:          "BitwiseXor",
	KindDPipe:               "DPipe",
	KindBetween:             "Between",
	KindIs:                  "Is",
	KindLike:                "Like",
	KindILike:               "ILike",
	KindIn:                  "In",
	KindSelect:              "Select",
	KindFrom:                "From",
	KindJoin:                "Join",
	KindTable:               "Table",
	KindTableAlias:          "TableAlias",
	KindWhere:               "Where",
	KindGroup:               "Group",
	KindOrder:               "Order",
	KindLimit:               "Limit",
	KindOffset:              "Offset",
	KindHint:                "Hint",
	KindBlock:               "Block",
	KindPlaceholder:         "Placeholder",
	KindTuple:               "Tuple",
	KindWith:                "With",
	KindCTE:                 "CTE",
	KindRecursiveWithSearch: "RecursiveWithSearch",
	KindUnion:               "Union",
	KindExcept:              "Except",
	KindIntersect:           "Intersect",
	KindSubquery:            "Subquery",
	KindHaving:              "Having",
	KindQualify:             "Qualify",
	KindCube:                "Cube",
	KindRollup:              "Rollup",
	KindGroupingSets:        "GroupingSets",
	KindOrdered:             "Ordered",
	KindDistinct:            "Distinct",
	KindWindow:              "Window",
	KindWindowSpec:          "WindowSpec",
	KindFilter:              "Filter",
	KindLimitOptions:        "LimitOptions",
	KindFetch:               "Fetch",
	KindCase:                "Case",
	KindIf:                  "If",
	KindExists:              "Exists",
	KindAny:                 "Any",
	KindAll:                 "All",
	KindNullSafeNEQ:         "NullSafeNEQ",
	KindAnonymous:           "Anonymous",
	KindAbs:                 "Abs",
	KindAvg:                 "Avg",
	KindSum:                 "Sum",
	KindSqrt:                "Sqrt",
	KindLn:                  "Ln",
	KindExp:                 "Exp",
	KindMin:                 "Min",
	KindMax:                 "Max",
	KindRound:               "Round",
	KindLog:                 "Log",
	KindPow:                 "Pow",
	KindStddev:              "Stddev",
	KindStddevPop:           "StddevPop",
	KindStddevSamp:          "StddevSamp",
	KindVariance:            "Variance",
	KindVariancePop:         "VariancePop",
	KindDay:                 "Day",
	KindMonth:               "Month",
	KindYear:                "Year",
	KindQuarter:             "Quarter",
	KindApproxDistinct:      "ApproxDistinct",
	KindHll:                 "Hll",
	KindCountIf:             "CountIf",
	KindQuantile:            "Quantile",
	KindCount:               "Count",
	KindCoalesce:            "Coalesce",
	KindGreatest:            "Greatest",
	KindLeast:               "Least",
}

// varLenArgs is the authoritative is_var_len_args=True set (mirroring the upstream Func
// class attribute): the final arg_type is a variadic list. It drives both FromArgList's
// var-len packing and the error_messages arg-count check (Node.ErrorMessages). Count/
// Coalesce/Greatest/Least set it upstream too; they build via custom builders (never
// FromArgList) but must still be recognized here so the over-arity check skips them.
var varLenArgs = map[Kind]bool{
	KindMax:       true,
	KindMin:       true,
	KindHll:       true,
	KindAnonymous: true,
	KindCount:     true,
	KindCoalesce:  true,
	KindGreatest:  true,
	KindLeast:     true,
}
