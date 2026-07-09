package expressions

// TableSample ports exp.TableSample (query.py:1723-1733): the `TABLESAMPLE (...)` modifier
// attached to a table/subquery source's already-wired "sample" arg key.
func TableSample(args Args) Expression { return newNode(KindTableSample, args) }
