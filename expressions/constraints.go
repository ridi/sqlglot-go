package expressions

// Builders for the CREATE-table constraint node family (see the block comment above the
// Kind consts in kinds.go for the upstream constraints.py line references); each is a
// one-line constructor per the node-model convention (AGENTS.md:34-43).
func ColumnConstraint(args Args) Expression { return newNode(KindColumnConstraint, args) }
func Constraint(args Args) Expression       { return newNode(KindConstraint, args) }
func PrimaryKey(args Args) Expression       { return newNode(KindPrimaryKey, args) }
func PrimaryKeyColumnConstraint(args Args) Expression {
	return newNode(KindPrimaryKeyColumnConstraint, args)
}
func ForeignKey(args Args) Expression              { return newNode(KindForeignKey, args) }
func Reference(args Args) Expression               { return newNode(KindReference, args) }
func NotNullColumnConstraint(args Args) Expression { return newNode(KindNotNullColumnConstraint, args) }
func UniqueColumnConstraint(args Args) Expression  { return newNode(KindUniqueColumnConstraint, args) }
func CheckColumnConstraint(args Args) Expression   { return newNode(KindCheckColumnConstraint, args) }
func DefaultColumnConstraint(args Args) Expression { return newNode(KindDefaultColumnConstraint, args) }
func CollateColumnConstraint(args Args) Expression { return newNode(KindCollateColumnConstraint, args) }
func CommentColumnConstraint(args Args) Expression { return newNode(KindCommentColumnConstraint, args) }
func CharacterSetColumnConstraint(args Args) Expression {
	return newNode(KindCharacterSetColumnConstraint, args)
}
func AutoIncrementColumnConstraint(args Args) Expression {
	return newNode(KindAutoIncrementColumnConstraint, args)
}
func OnUpdateColumnConstraint(args Args) Expression {
	return newNode(KindOnUpdateColumnConstraint, args)
}
func GeneratedAsIdentityColumnConstraint(args Args) Expression {
	return newNode(KindGeneratedAsIdentityColumnConstraint, args)
}
func GeneratedAsRowColumnConstraint(args Args) Expression {
	return newNode(KindGeneratedAsRowColumnConstraint, args)
}
func ComputedColumnConstraint(args Args) Expression {
	return newNode(KindComputedColumnConstraint, args)
}
func CaseSpecificColumnConstraint(args Args) Expression {
	return newNode(KindCaseSpecificColumnConstraint, args)
}
func NotForReplicationColumnConstraint(args Args) Expression {
	return newNode(KindNotForReplicationColumnConstraint, args)
}
func ZeroFillColumnConstraint(args Args) Expression {
	return newNode(KindZeroFillColumnConstraint, args)
}
func InvisibleColumnConstraint(args Args) Expression {
	return newNode(KindInvisibleColumnConstraint, args)
}
func IndexColumnConstraint(args Args) Expression { return newNode(KindIndexColumnConstraint, args) }
func IndexConstraintOption(args Args) Expression { return newNode(KindIndexConstraintOption, args) }
func IndexParameters(args Args) Expression       { return newNode(KindIndexParameters, args) }
func ColumnPrefix(args Args) Expression          { return newNode(KindColumnPrefix, args) }
func AddConstraint(args Args) Expression         { return newNode(KindAddConstraint, args) }
