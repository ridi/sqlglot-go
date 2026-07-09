package expressions

// Slice ports exp.Slice (expressions/core.py:2017-2018): the `start:end:step` triple
// inside a bracket subscript, e.g. x[1:2].
func Slice(args Args) Expression { return newNode(KindSlice, args) }

// National ports exp.National (expressions/query.py:585-586, is_primitive=True): a
// national-character string literal, e.g. N'abc'.
func National(args Args) Expression { return newNode(KindNational, args) }
