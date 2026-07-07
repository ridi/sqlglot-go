package expressions

func Select(args Args) Expression     { return newNode(KindSelect, args) }
func From(args Args) Expression       { return newNode(KindFrom, args) }
func Join(args Args) Expression       { return newNode(KindJoin, args) }
func Table(args Args) Expression      { return newNode(KindTable, args) }
func TableAlias(args Args) Expression { return newNode(KindTableAlias, args) }
func Where(args Args) Expression      { return newNode(KindWhere, args) }
func Group(args Args) Expression      { return newNode(KindGroup, args) }
func Order(args Args) Expression      { return newNode(KindOrder, args) }
func Limit(args Args) Expression      { return newNode(KindLimit, args) }
func Offset(args Args) Expression     { return newNode(KindOffset, args) }
func Tuple(args Args) Expression      { return newNode(KindTuple, args) }
func Block(args Args) Expression      { return newNode(KindBlock, args) }
