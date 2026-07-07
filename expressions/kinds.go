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

var argTypes = map[Kind][]argSpec{
	KindExpression:  defaultArgTypes,
	KindColumn:      {{"this", true}, {"table", false}, {"db", false}, {"catalog", false}, {"join_mark", false}},
	KindLiteral:     {{"this", true}, {"is_string", true}},
	KindIdentifier:  {{"this", true}, {"quoted", false}, {"global_", false}, {"temporary", false}},
	KindStar:        {{"except_", false}, {"replace", false}, {"rename", false}, {"ilike", false}},
	KindAlias:       {{"this", true}, {"alias", false}},
	KindAliases:     {{"this", true}, {"expressions", true}},
	KindDot:         {{"this", true}, {"expression", true}},
	KindNull:        {},
	KindBoolean:     defaultArgTypes,
	KindVar:         defaultArgTypes,
	KindParen:       defaultArgTypes,
	KindNeg:         defaultArgTypes,
	KindNot:         defaultArgTypes,
	KindAdd:         {{"this", true}, {"expression", true}},
	KindSub:         {{"this", true}, {"expression", true}},
	KindMul:         {{"this", true}, {"expression", true}},
	KindDiv:         {{"this", true}, {"expression", true}, {"typed", false}, {"safe", false}},
	KindMod:         {{"this", true}, {"expression", true}},
	KindEQ:          {{"this", true}, {"expression", true}},
	KindNEQ:         {{"this", true}, {"expression", true}},
	KindNullSafeEQ:  {{"this", true}, {"expression", true}},
	KindGT:          {{"this", true}, {"expression", true}},
	KindGTE:         {{"this", true}, {"expression", true}},
	KindLT:          {{"this", true}, {"expression", true}},
	KindLTE:         {{"this", true}, {"expression", true}},
	KindAnd:         {{"this", true}, {"expression", true}},
	KindOr:          {{"this", true}, {"expression", true}},
	KindBitwiseAnd:  {{"this", true}, {"expression", true}, {"padside", false}},
	KindBitwiseOr:   {{"this", true}, {"expression", true}, {"padside", false}},
	KindBitwiseXor:  {{"this", true}, {"expression", true}, {"padside", false}},
	KindDPipe:       {{"this", true}, {"expression", true}, {"safe", false}},
	KindBetween:     {{"this", true}, {"low", true}, {"high", true}, {"symmetric", false}},
	KindIs:          {{"this", true}, {"expression", true}},
	KindLike:        {{"this", true}, {"expression", true}, {"negate", false}},
	KindILike:       {{"this", true}, {"expression", true}, {"negate", false}},
	KindIn:          {{"this", true}, {"expressions", false}, {"query", false}, {"unnest", false}, {"field", false}, {"is_global", false}},
	KindSelect:      {{"with_", false}, {"kind", false}, {"expressions", false}, {"hint", false}, {"distinct", false}, {"into", false}, {"from_", false}, {"operation_modifiers", false}, {"exclude", false}, {"match", false}, {"laterals", false}, {"joins", false}, {"connect", false}, {"pivots", false}, {"prewhere", false}, {"where", false}, {"group", false}, {"having", false}, {"qualify", false}, {"windows", false}, {"distribute", false}, {"sort", false}, {"cluster", false}, {"order", false}, {"limit", false}, {"offset", false}, {"locks", false}, {"sample", false}, {"settings", false}, {"format", false}, {"options", false}, {"for_", false}},
	KindFrom:        defaultArgTypes,
	KindJoin:        {{"this", true}, {"on", false}, {"side", false}, {"kind", false}, {"using", false}, {"method", false}, {"global_", false}, {"hint", false}, {"match_condition", false}, {"directed", false}, {"expressions", false}, {"pivots", false}},
	KindTable:       {{"this", false}, {"alias", false}, {"db", false}, {"catalog", false}, {"laterals", false}, {"joins", false}, {"pivots", false}, {"hints", false}, {"system_time", false}, {"version", false}, {"format", false}, {"pattern", false}, {"ordinality", false}, {"when", false}, {"only", false}, {"partition", false}, {"changes", false}, {"rows_from", false}, {"sample", false}, {"indexed", false}},
	KindTableAlias:  {{"this", false}, {"columns", false}},
	KindWhere:       defaultArgTypes,
	KindGroup:       {{"expressions", false}, {"grouping_sets", false}, {"cube", false}, {"rollup", false}, {"totals", false}, {"all", false}},
	KindOrder:       {{"this", false}, {"expressions", true}, {"siblings", false}},
	KindLimit:       {{"this", false}, {"expression", true}, {"offset", false}, {"limit_options", false}, {"expressions", false}},
	KindOffset:      {{"this", false}, {"expression", true}, {"expressions", false}},
	KindHint:        {{"expressions", true}},
	KindBlock:       {{"expressions", true}},
	KindPlaceholder: {{"this", false}, {"kind", false}, {"widget", false}, {"jdbc", false}},
	KindTuple:       {{"expressions", false}},
}

var traitsOf = map[Kind]Trait{
	KindColumn:      TraitCondition,
	KindLiteral:     TraitCondition,
	KindNull:        TraitCondition,
	KindBoolean:     TraitCondition,
	KindPlaceholder: TraitCondition,
	KindDot:         TraitCondition | TraitBinary,
	KindAdd:         TraitCondition | TraitBinary,
	KindSub:         TraitCondition | TraitBinary,
	KindMul:         TraitCondition | TraitBinary,
	KindDiv:         TraitCondition | TraitBinary,
	KindMod:         TraitCondition | TraitBinary,
	KindEQ:          TraitCondition | TraitBinary | TraitPredicate,
	KindNEQ:         TraitCondition | TraitBinary | TraitPredicate,
	KindNullSafeEQ:  TraitCondition | TraitBinary | TraitPredicate,
	KindGT:          TraitCondition | TraitBinary | TraitPredicate,
	KindGTE:         TraitCondition | TraitBinary | TraitPredicate,
	KindLT:          TraitCondition | TraitBinary | TraitPredicate,
	KindLTE:         TraitCondition | TraitBinary | TraitPredicate,
	KindLike:        TraitCondition | TraitBinary | TraitPredicate,
	KindILike:       TraitCondition | TraitBinary | TraitPredicate,
	KindIs:          TraitCondition | TraitBinary | TraitPredicate,
	KindBetween:     TraitPredicate,
	KindIn:          TraitPredicate,
	KindAnd:         TraitCondition | TraitBinary | TraitConnector | TraitFunc,
	KindOr:          TraitCondition | TraitBinary | TraitConnector | TraitFunc,
	KindBitwiseAnd:  TraitCondition | TraitBinary,
	KindBitwiseOr:   TraitCondition | TraitBinary,
	KindBitwiseXor:  TraitCondition | TraitBinary,
	KindDPipe:       TraitCondition | TraitBinary,
	KindParen:       TraitCondition | TraitUnary,
	KindNeg:         TraitCondition | TraitUnary,
	KindNot:         TraitCondition | TraitUnary,
	KindSelect:      TraitQuery,
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
	KindExpression:  "Expression",
	KindColumn:      "Column",
	KindLiteral:     "Literal",
	KindIdentifier:  "Identifier",
	KindStar:        "Star",
	KindAlias:       "Alias",
	KindAliases:     "Aliases",
	KindDot:         "Dot",
	KindNull:        "Null",
	KindBoolean:     "Boolean",
	KindVar:         "Var",
	KindParen:       "Paren",
	KindNeg:         "Neg",
	KindNot:         "Not",
	KindAdd:         "Add",
	KindSub:         "Sub",
	KindMul:         "Mul",
	KindDiv:         "Div",
	KindMod:         "Mod",
	KindEQ:          "EQ",
	KindNEQ:         "NEQ",
	KindNullSafeEQ:  "NullSafeEQ",
	KindGT:          "GT",
	KindGTE:         "GTE",
	KindLT:          "LT",
	KindLTE:         "LTE",
	KindAnd:         "And",
	KindOr:          "Or",
	KindBitwiseAnd:  "BitwiseAnd",
	KindBitwiseOr:   "BitwiseOr",
	KindBitwiseXor:  "BitwiseXor",
	KindDPipe:       "DPipe",
	KindBetween:     "Between",
	KindIs:          "Is",
	KindLike:        "Like",
	KindILike:       "ILike",
	KindIn:          "In",
	KindSelect:      "Select",
	KindFrom:        "From",
	KindJoin:        "Join",
	KindTable:       "Table",
	KindTableAlias:  "TableAlias",
	KindWhere:       "Where",
	KindGroup:       "Group",
	KindOrder:       "Order",
	KindLimit:       "Limit",
	KindOffset:      "Offset",
	KindHint:        "Hint",
	KindBlock:       "Block",
	KindPlaceholder: "Placeholder",
	KindTuple:       "Tuple",
}

var modifiableKinds = map[Kind]bool{
	KindSelect: true,
	KindTable:  true,
}
