package expressions

func FromArgList(kind Kind, args []Expression) Expression {
	specs := argTypesFor(kind)
	m := Args{}
	if len(specs) == 0 {
		return newNode(kind, m)
	}
	if varLenArgs[kind] {
		nNon := len(specs) - 1
		for i := 0; i < nNon && i < len(args); i++ {
			m[specs[i].Key] = args[i]
		}
		rest := []Expression{}
		if nNon <= len(args) {
			rest = args[nNon:]
		}
		m[specs[nNon].Key] = rest
	} else {
		for i, spec := range specs {
			if i < len(args) {
				m[spec.Key] = args[i]
			}
		}
	}
	return newNode(kind, m)
}

var FunctionByName = map[string]func([]Expression) Expression{
	"ABS":                   genericFunction(KindAbs),
	"AVG":                   genericFunction(KindAvg),
	"SUM":                   genericFunction(KindSum),
	"SQRT":                  genericFunction(KindSqrt),
	"LN":                    genericFunction(KindLn),
	"EXP":                   genericFunction(KindExp),
	"MIN":                   genericFunction(KindMin),
	"MAX":                   genericFunction(KindMax),
	"ROUND":                 genericFunction(KindRound),
	"LOG":                   genericFunction(KindLog),
	"POW":                   genericFunction(KindPow),
	"POWER":                 genericFunction(KindPow),
	"STDDEV":                genericFunction(KindStddev),
	"STDEV":                 genericFunction(KindStddev),
	"STDDEV_POP":            genericFunction(KindStddevPop),
	"STDDEV_SAMP":           genericFunction(KindStddevSamp),
	"VARIANCE":              genericFunction(KindVariance),
	"VARIANCE_SAMP":         genericFunction(KindVariance),
	"VAR_SAMP":              genericFunction(KindVariance),
	"VARIANCE_POP":          genericFunction(KindVariancePop),
	"VAR_POP":               genericFunction(KindVariancePop),
	"DAY":                   genericFunction(KindDay),
	"MONTH":                 genericFunction(KindMonth),
	"YEAR":                  genericFunction(KindYear),
	"QUARTER":               genericFunction(KindQuarter),
	"APPROX_DISTINCT":       genericFunction(KindApproxDistinct),
	"APPROX_COUNT_DISTINCT": genericFunction(KindApproxDistinct),
	"HLL":                   genericFunction(KindHll),
	"COUNT_IF":              genericFunction(KindCountIf),
	"COUNTIF":               genericFunction(KindCountIf),
	"QUANTILE":              genericFunction(KindQuantile),
	"COUNT": func(args []Expression) Expression {
		m := Args{"big_int": true}
		if len(args) > 0 {
			m["this"] = args[0]
		}
		if len(args) > 1 {
			m["expressions"] = args[1:]
		}
		return newNode(KindCount, m)
	},
	// Base dialect maps COALESCE/IFNULL/NVL to the same builder with is_nvl unset
	// (parser.py:329); only Oracle-family dialects set is_nvl=True (deferred).
	"COALESCE": coalesceFunction(),
	"IFNULL":   coalesceFunction(),
	"NVL":      coalesceFunction(),
	"GREATEST": func(args []Expression) Expression {
		m := Args{"ignore_nulls": true}
		if len(args) > 0 {
			m["this"] = args[0]
		}
		if len(args) > 1 {
			m["expressions"] = args[1:]
		}
		return newNode(KindGreatest, m)
	},
	"LEAST": func(args []Expression) Expression {
		m := Args{"ignore_nulls": true}
		if len(args) > 0 {
			m["this"] = args[0]
		}
		if len(args) > 1 {
			m["expressions"] = args[1:]
		}
		return newNode(KindLeast, m)
	},
}

func genericFunction(kind Kind) func([]Expression) Expression {
	return func(args []Expression) Expression { return FromArgList(kind, args) }
}

func coalesceFunction() func([]Expression) Expression {
	return func(args []Expression) Expression {
		m := Args{}
		if len(args) > 0 {
			m["this"] = args[0]
		}
		if len(args) > 1 {
			m["expressions"] = args[1:]
		}
		return newNode(KindCoalesce, m)
	}
}

func Abs(args Args) Expression            { return newNode(KindAbs, args) }
func Avg(args Args) Expression            { return newNode(KindAvg, args) }
func Sum(args Args) Expression            { return newNode(KindSum, args) }
func Sqrt(args Args) Expression           { return newNode(KindSqrt, args) }
func Ln(args Args) Expression             { return newNode(KindLn, args) }
func Exp(args Args) Expression            { return newNode(KindExp, args) }
func Min(args Args) Expression            { return newNode(KindMin, args) }
func Max(args Args) Expression            { return newNode(KindMax, args) }
func Round(args Args) Expression          { return newNode(KindRound, args) }
func Log(args Args) Expression            { return newNode(KindLog, args) }
func Pow(args Args) Expression            { return newNode(KindPow, args) }
func Stddev(args Args) Expression         { return newNode(KindStddev, args) }
func StddevPop(args Args) Expression      { return newNode(KindStddevPop, args) }
func StddevSamp(args Args) Expression     { return newNode(KindStddevSamp, args) }
func Variance(args Args) Expression       { return newNode(KindVariance, args) }
func VariancePop(args Args) Expression    { return newNode(KindVariancePop, args) }
func Day(args Args) Expression            { return newNode(KindDay, args) }
func Month(args Args) Expression          { return newNode(KindMonth, args) }
func Year(args Args) Expression           { return newNode(KindYear, args) }
func Quarter(args Args) Expression        { return newNode(KindQuarter, args) }
func ApproxDistinct(args Args) Expression { return newNode(KindApproxDistinct, args) }
func Hll(args Args) Expression            { return newNode(KindHll, args) }
func CountIf(args Args) Expression        { return newNode(KindCountIf, args) }
func Quantile(args Args) Expression       { return newNode(KindQuantile, args) }
func Count(args Args) Expression          { return newNode(KindCount, args) }
func Coalesce(args Args) Expression       { return newNode(KindCoalesce, args) }
func Greatest(args Args) Expression       { return newNode(KindGreatest, args) }
func Least(args Args) Expression          { return newNode(KindLeast, args) }
