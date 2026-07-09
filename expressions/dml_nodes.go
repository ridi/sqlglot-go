package expressions

// FROM-clause / DML tail cluster builders: Directory (dml.py:187, Hive/Spark `INSERT
// OVERWRITE [LOCAL] DIRECTORY '...'`), RowFormatDelimitedProperty/RowFormatSerdeProperty/
// SerdeProperties (properties.py:405,418,452, the Hive `ROW FORMAT ...` clause used by
// Directory), and WithTableHint/IndexTableHint (query.py:923,927 - T-SQL `WITH (...)` /
// MySQL `USE|FORCE|IGNORE INDEX` table hints).
func Directory(args Args) Expression { return newNode(KindDirectory, args) }
func RowFormatDelimitedProperty(args Args) Expression {
	return newNode(KindRowFormatDelimitedProperty, args)
}
func RowFormatSerdeProperty(args Args) Expression { return newNode(KindRowFormatSerdeProperty, args) }
func SerdeProperties(args Args) Expression        { return newNode(KindSerdeProperties, args) }
func WithTableHint(args Args) Expression          { return newNode(KindWithTableHint, args) }
func IndexTableHint(args Args) Expression         { return newNode(KindIndexTableHint, args) }
