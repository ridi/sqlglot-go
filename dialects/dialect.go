package dialects

import (
	"fmt"
	"strings"

	"github.com/sjincho/sqlglot-go/tokens"
)

type Dialect struct {
	TokenizerConfig         tokens.TokenizerConfig
	DPipeIsStringConcat     bool
	StrictStringConcat      bool
	TypedDivision           bool
	SafeDivision            bool
	SupportsColumnJoinMarks bool
	ColonIsVariantExtract   bool
}

func Base() *Dialect {
	return &Dialect{
		TokenizerConfig:     tokens.BaseConfig(),
		DPipeIsStringConcat: true,
	}
}

func GetOrRaise(name string) (*Dialect, error) {
	switch strings.ToLower(name) {
	case "", "base":
		return Base(), nil
	case "mysql", "postgres":
		// TODO(slice 5): wire real MySQL and Postgres dialect behavior.
		return Base(), nil
	default:
		return nil, fmt.Errorf("unknown dialect %q", name)
	}
}

func (d *Dialect) NewTokenizer() *tokens.Tokenizer {
	return tokens.NewTokenizerWithConfig(d.TokenizerConfig)
}
