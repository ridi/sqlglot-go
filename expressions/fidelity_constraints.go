package expressions

// Builders for column-constraint nodes from expressions/constraints.py.
func CompressColumnConstraint(args Args) Expression {
	return newNode(KindCompressColumnConstraint, args)
}
func DateFormatColumnConstraint(args Args) Expression {
	return newNode(KindDateFormatColumnConstraint, args)
}
func ExcludeColumnConstraint(args Args) Expression {
	return newNode(KindExcludeColumnConstraint, args)
}
func InlineLengthColumnConstraint(args Args) Expression {
	return newNode(KindInlineLengthColumnConstraint, args)
}
func TitleColumnConstraint(args Args) Expression {
	return newNode(KindTitleColumnConstraint, args)
}
func UppercaseColumnConstraint(args Args) Expression {
	return newNode(KindUppercaseColumnConstraint, args)
}
func WithOperator(args Args) Expression          { return newNode(KindWithOperator, args) }
func InOutColumnConstraint(args Args) Expression { return newNode(KindInOutColumnConstraint, args) }
