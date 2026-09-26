package workflow

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ApplyTransform applies the intentionally small, JSON-only transform
// vocabulary used by workflow definitions. It never performs I/O or executes
// code, so historical documents can safely replay the same operations.
func ApplyTransform(values map[string]any, config map[string]any) error {
	if values == nil {
		return fmt.Errorf("transform values are nil")
	}
	raw, ok := config["operations"]
	if !ok || raw == nil {
		return nil
	}
	operations, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("transform.operations must be an array")
	}
	for index, rawOperation := range operations {
		operation, ok := rawOperation.(map[string]any)
		if !ok {
			return fmt.Errorf("transform operation %d must be an object", index)
		}
		op := strings.ToLower(strings.TrimSpace(stringValue(operation["op"])))
		field := strings.TrimSpace(stringValue(operation["field"]))
		switch op {
		case "rename":
			to := strings.TrimSpace(stringValue(operation["to"]))
			if field == "" || to == "" {
				return fmt.Errorf("transform rename requires field and to")
			}
			if value, exists := values[field]; exists {
				values[to] = value
				delete(values, field)
			}
		case "trim", "lower", "upper":
			if field == "" {
				return fmt.Errorf("transform %s requires field", op)
			}
			if value, exists := values[field]; exists {
				if value == nil {
					continue
				}
				text, ok := value.(string)
				if !ok {
					return fmt.Errorf("transform %s field %q must be a string", op, field)
				}
				switch op {
				case "trim":
					values[field] = strings.TrimSpace(text)
				case "lower":
					values[field] = strings.ToLower(text)
				case "upper":
					values[field] = strings.ToUpper(text)
				}
			}
		case "split":
			if field == "" {
				return fmt.Errorf("transform split requires field")
			}
			if value, exists := values[field]; exists {
				text, ok := value.(string)
				if !ok {
					return fmt.Errorf("transform split field %q must be a string", field)
				}
				separator := stringValue(operation["separator"])
				if separator == "" {
					separator = ","
				}
				parts := strings.Split(text, separator)
				result := make([]any, 0, len(parts))
				for _, part := range parts {
					result = append(result, strings.TrimSpace(part))
				}
				values[field] = result
			}
		case "join":
			if field == "" {
				return fmt.Errorf("transform join requires field")
			}
			if value, exists := values[field]; exists {
				items, ok := value.([]any)
				if !ok {
					return fmt.Errorf("transform join field %q must be an array", field)
				}
				separator := stringValue(operation["separator"])
				values[field] = strings.Join(stringValues(items), separator)
			}
		case "default", "set":
			if field == "" {
				return fmt.Errorf("transform %s requires field", op)
			}
			if op == "set" || isEmptyValue(values[field]) {
				values[field] = operation["value"]
			}
		case "delete":
			if field == "" {
				return fmt.Errorf("transform delete requires field")
			}
			delete(values, field)
		default:
			return fmt.Errorf("unsupported transform operation: %s", op)
		}
	}
	return nil
}

// ValidateValues validates extracted fields without mutating them.
func ValidateValues(values map[string]any, config map[string]any) error {
	raw, ok := config["fields"]
	if !ok || raw == nil {
		return nil
	}
	fields, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("validate.fields must be an array")
	}
	for index, rawField := range fields {
		field, ok := rawField.(map[string]any)
		if !ok {
			return fmt.Errorf("validate field %d must be an object", index)
		}
		name := strings.TrimSpace(stringValue(field["name"]))
		if name == "" {
			return fmt.Errorf("validate field %d name is required", index)
		}
		value, exists := values[name]
		if required, _ := field["required"].(bool); required && (!exists || isEmptyValue(value)) {
			return fmt.Errorf("required field %q is empty", name)
		}
		if !exists || isEmptyValue(value) {
			continue
		}
		if expected := strings.ToLower(strings.TrimSpace(stringValue(field["type"]))); expected != "" && !matchesType(value, expected) {
			return fmt.Errorf("field %q has invalid type, expected %s", name, expected)
		}
		if pattern := stringValue(field["regex"]); pattern != "" {
			text := stringValue(value)
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("field %q regex is invalid: %w", name, err)
			}
			if !re.MatchString(text) {
				return fmt.Errorf("field %q does not match regex", name)
			}
		}
		if rawEnum, exists := field["enum"]; exists && rawEnum != nil {
			allowed, ok := rawEnum.([]any)
			if !ok {
				return fmt.Errorf("field %q enum must be an array", name)
			}
			matched := false
			for _, item := range allowed {
				if reflect.DeepEqual(item, value) || stringValue(item) == stringValue(value) {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("field %q is not an allowed value", name)
			}
		}
	}
	return nil
}

func matchesType(value any, expected string) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "int", "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "float", "number":
		_, ok := value.(float32)
		if ok {
			return true
		}
		_, ok = value.(float64)
		return ok
	case "bool", "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		return reflect.ValueOf(value).Kind() == reflect.Slice
	default:
		return false
	}
}

func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	if array, ok := value.([]any); ok {
		return len(array) == 0
	}
	return false
}

func stringValues(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, stringValue(value))
	}
	return result
}
