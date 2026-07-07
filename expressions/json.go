package expressions

func JSONExtract(args Args) Expression        { return newNode(KindJSONExtract, args) }
func JSONExtractScalar(args Args) Expression  { return newNode(KindJSONExtractScalar, args) }
func JSONBExtract(args Args) Expression       { return newNode(KindJSONBExtract, args) }
func JSONBExtractScalar(args Args) Expression { return newNode(KindJSONBExtractScalar, args) }
func JSONCast(args Args) Expression           { return newNode(KindJSONCast, args) }
