package probe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/schema"
)

func expressionsFor(expression exp.Expression, key string) []exp.Expression {
	if expression == nil {
		return nil
	}
	if node, ok := expression.(*exp.Node); ok {
		return node.ExpressionsFor(key)
	}
	value := expression.Arg(key)
	switch v := value.(type) {
	case []exp.Expression:
		return v
	case exp.Expression:
		if v == nil {
			return nil
		}
		return []exp.Expression{v}
	default:
		return nil
	}
}

func asExpression(value any) exp.Expression {
	if expression, ok := value.(exp.Expression); ok {
		return expression
	}
	return nil
}

func truthy(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case []exp.Expression:
		return len(v) > 0
	case []any:
		return len(v) > 0
	}
	return true
}

func boolPtr(value bool) *bool { return &value }

func stringSet(values ...string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}

func addSet(dst map[string]bool, src map[string]bool) {
	for value := range src {
		dst[value] = true
	}
}

func subtractSet(base map[string]bool, remove map[string]bool) map[string]bool {
	out := map[string]bool{}
	for value := range base {
		if !remove[value] {
			out[value] = true
		}
	}
	return out
}

func sortedSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	if len(values) < 2 {
		return
	}
	for i := 1; i < len(values); i++ {
		value := values[i]
		j := i - 1
		for j >= 0 && values[j] > value {
			values[j+1] = values[j]
			j--
		}
		values[j+1] = value
	}
}

func lowerSchema(sch *schema.Mapping) (*schema.Mapping, map[string][]string, map[string]map[string]bool, map[string]bool) {
	lowered := lowerMappingKeys(sch)
	schemaCols := map[string][]string{}
	schemaCI := map[string]map[string]bool{}
	schemaTables := map[string]bool{}
	if lowered == nil {
		lowered = schema.NewMapping()
	}
	for _, table := range lowered.Keys() {
		schemaTables[table] = true
		value, _ := lowered.Get(table)
		cols, ok := value.(*schema.Mapping)
		if !ok || cols == nil {
			continue
		}
		ci := map[string]bool{}
		for _, col := range cols.Keys() {
			schemaCols[table] = append(schemaCols[table], col)
			ci[col] = true
		}
		schemaCI[table] = ci
	}
	return lowered, schemaCols, schemaCI, schemaTables
}

func lowerMappingKeys(m *schema.Mapping) *schema.Mapping {
	if m == nil {
		return schema.NewMapping()
	}
	out := schema.NewMapping()
	for _, key := range m.Keys() {
		value, _ := m.Get(key)
		if child, ok := value.(*schema.Mapping); ok {
			value = lowerMappingKeys(child)
		}
		out.Set(strings.ToLower(key), value)
	}
	return out
}

func mappingToJSON(m *schema.Mapping) json.RawMessage {
	var buf bytes.Buffer
	writeMappingJSON(&buf, m)
	return json.RawMessage(buf.Bytes())
}

func writeMappingJSON(buf *bytes.Buffer, m *schema.Mapping) {
	buf.WriteByte('{')
	if m != nil {
		keys := m.Keys()
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONScalar(buf, key)
			buf.WriteByte(':')
			value, _ := m.Get(key)
			writeJSONValue(buf, value)
		}
	}
	buf.WriteByte('}')
}

func writeJSONValue(buf *bytes.Buffer, value any) {
	switch v := value.(type) {
	case *schema.Mapping:
		writeMappingJSON(buf, v)
	default:
		writeJSONScalar(buf, v)
	}
}

func writeJSONScalar(buf *bytes.Buffer, value any) {
	encoded, err := json.Marshal(value)
	if err != nil {
		encoded = []byte("null")
	}
	buf.Write(encoded)
}

func decodeSchemaJSON(schemaJSON string) (*schema.Mapping, error) {
	dec := json.NewDecoder(strings.NewReader(schemaJSON))
	dec.UseNumber()
	value, err := decodeJSONValue(dec)
	if err != nil {
		return nil, err
	}
	mapping, ok := value.(*schema.Mapping)
	if !ok {
		return nil, fmt.Errorf("schema JSON must be an object")
	}
	return mapping, nil
}

func decodeJSONValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch value := tok.(type) {
	case json.Delim:
		switch value {
		case '{':
			m := schema.NewMapping()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("schema object key must be a string")
				}
				child, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				m.Set(key, child)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return m, nil
		case '[':
			var out []any
			for dec.More() {
				child, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				out = append(out, child)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return out, nil
		}
		return nil, fmt.Errorf("unexpected JSON delimiter %q", value)
	case string, bool, nil, json.Number:
		return value, nil
	default:
		return value, nil
	}
}
