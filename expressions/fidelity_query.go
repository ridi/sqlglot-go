package expressions

// Builders for partition and ANALYZE nodes from expressions/query.py.
func PartitionRange(args Args) Expression   { return newNode(KindPartitionRange, args) }
func AnalyzeHistogram(args Args) Expression { return newNode(KindAnalyzeHistogram, args) }
func AnalyzeWith(args Args) Expression      { return newNode(KindAnalyzeWith, args) }
func UsingData(args Args) Expression        { return newNode(KindUsingData, args) }
