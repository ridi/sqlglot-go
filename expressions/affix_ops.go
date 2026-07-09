package expressions

// Builders for the affix/connector-operator cluster (see the KindCollate block comment in
// kinds.go for exact upstream line numbers): COLLATE, unary bitwise-not (~), PIPE_SLASH/
// DPIPE_SLASH (|/, ||/ -> SQRT/CBRT), the << >> bitwise-shift operators, and mysql's XOR.
func Collate(args Args) Expression           { return newNode(KindCollate, args) }
func BitwiseNot(args Args) Expression        { return newNode(KindBitwiseNot, args) }
func Cbrt(args Args) Expression              { return newNode(KindCbrt, args) }
func BitwiseLeftShift(args Args) Expression  { return newNode(KindBitwiseLeftShift, args) }
func BitwiseRightShift(args Args) Expression { return newNode(KindBitwiseRightShift, args) }
func Xor(args Args) Expression               { return newNode(KindXor, args) }
