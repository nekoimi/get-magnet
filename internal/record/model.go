package record

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Field struct {
	Key      string
	Type     string
	Required bool
	Multiple bool
}
type Schema struct {
	Version          int
	UniqueKeyFields  []string
	EmptyValuePolicy string
	Fields           []Field
}
type Prepared struct {
	Key    string
	Values map[string]any
	Hash   string
}

func Prepare(schema Schema, raw map[string]any) (Prepared, error) {
	if raw == nil {
		return Prepared{}, errors.New("candidate must be an object")
	}
	if len(schema.UniqueKeyFields) == 0 {
		return Prepared{}, errors.New("dataset has no unique key")
	}
	fields := map[string]Field{}
	for _, f := range schema.Fields {
		fields[f.Key] = f
	}
	out := make(map[string]any, len(raw))
	for name, value := range raw {
		f, ok := fields[name]
		if !ok {
			return Prepared{}, fmt.Errorf("unknown field %q", name)
		}
		if value == nil {
			out[name] = nil
			continue
		}
		if f.Multiple {
			items, ok := value.([]any)
			if !ok {
				return Prepared{}, fmt.Errorf("field %q must be an array", name)
			}
			normalized := make([]any, 0, len(items))
			for _, item := range items {
				v, err := normalize(f.Type, item)
				if err != nil {
					return Prepared{}, fmt.Errorf("field %q: %w", name, err)
				}
				normalized = append(normalized, v)
			}
			out[name] = normalized
		} else {
			v, err := normalize(f.Type, value)
			if err != nil {
				return Prepared{}, fmt.Errorf("field %q: %w", name, err)
			}
			out[name] = v
		}
	}
	for _, f := range schema.Fields {
		if f.Required && empty(out[f.Key]) {
			return Prepared{}, fmt.Errorf("required field %q is empty", f.Key)
		}
	}
	parts := make([]string, 0, len(schema.UniqueKeyFields))
	for _, name := range schema.UniqueKeyFields {
		f, ok := fields[name]
		if !ok || f.Multiple {
			return Prepared{}, fmt.Errorf("invalid unique key field %q", name)
		}
		v := out[name]
		if empty(v) {
			return Prepared{}, fmt.Errorf("unique key field %q is empty", name)
		}
		parts = append(parts, fmt.Sprint(v))
	}
	key := parts[0]
	if len(parts) > 1 {
		encoded, _ := json.Marshal(parts)
		key = string(encoded)
	}
	if len(key) > 512 {
		h := sha256.Sum256([]byte(key))
		key = key[:400] + "-" + hex.EncodeToString(h[:])
	}
	return Prepared{Key: key, Values: out, Hash: Hash(out)}, nil
}

func normalize(kind string, value any) (any, error) {
	switch kind {
	case "string":
		v, ok := value.(string)
		if !ok {
			return nil, errors.New("expected string")
		}
		return strings.TrimSpace(v), nil
	case "url":
		v, ok := value.(string)
		if !ok {
			return nil, errors.New("expected URL string")
		}
		v = strings.TrimSpace(v)
		u, err := url.Parse(v)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "magnet") {
			if !strings.HasPrefix(v, "magnet:?") {
				return nil, errors.New("expected absolute URL")
			}
		}
		return v, nil
	case "integer":
		switch v := value.(type) {
		case json.Number:
			i, e := v.Int64()
			return i, e
		case float64:
			if v != float64(int64(v)) {
				return nil, errors.New("expected integer")
			}
			return int64(v), nil
		case int64, int, int32:
			return v, nil
		default:
			return nil, errors.New("expected integer")
		}
	case "number":
		switch value.(type) {
		case json.Number, float64, int, int64:
			return value, nil
		default:
			return nil, errors.New("expected number")
		}
	case "boolean":
		v, ok := value.(bool)
		if !ok {
			return nil, errors.New("expected boolean")
		}
		return v, nil
	case "datetime":
		v, ok := value.(string)
		if !ok {
			return nil, errors.New("expected date-time string")
		}
		if _, err := time.Parse(time.RFC3339, v); err != nil {
			return nil, err
		}
		return v, nil
	case "json":
		if !json.Valid(mustJSON(value)) {
			return nil, errors.New("invalid JSON")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", kind)
	}
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func empty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	}
	return false
}
func Hash(v map[string]any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func Merge(old, new map[string]any, policy string) (map[string]any, []string) {
	out := make(map[string]any, len(old)+len(new))
	for k, v := range old {
		out[k] = v
	}
	changed := []string{}
	for k, v := range new {
		if policy == "preserve" && empty(v) {
			continue
		}
		if policy == "preserve" {
			if incoming, ok := v.([]any); ok {
				if existing, ok := out[k].([]any); ok {
					seen := map[string]bool{}
					union := make([]any, 0, len(existing)+len(incoming))
					for _, item := range append(existing, incoming...) {
						b, _ := json.Marshal(item)
						if !seen[string(b)] {
							seen[string(b)] = true
							union = append(union, item)
						}
					}
					v = union
				}
			}
		}
		a, _ := json.Marshal(out[k])
		b, _ := json.Marshal(v)
		if string(a) != string(b) {
			out[k] = v
			changed = append(changed, k)
		}
	}
	sort.Strings(changed)
	return out, changed
}
func LegacyMagnetKey(key string) string {
	if i := strings.LastIndex(key, "#duplicate:"); i > 0 {
		if _, e := strconv.ParseInt(key[i+11:], 10, 64); e == nil {
			return key[:i]
		}
	}
	return key
}
