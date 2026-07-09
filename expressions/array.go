package expressions

func Array(args Args) Expression   { return newNode(KindArray, args) }
func Struct(args Args) Expression  { return newNode(KindStruct, args) }
func Unnest(args Args) Expression  { return newNode(KindUnnest, args) }
func Bracket(args Args) Expression { return newNode(KindBracket, args) }
