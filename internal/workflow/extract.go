package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

type FieldRule struct {
	Name         string `json:"name"`
	Selector     string `json:"selector"`
	SelectorType string `json:"selector_type,omitempty"`
	Attribute    string `json:"attribute,omitempty"`
	Regex        string `json:"regex,omitempty"`
	Clean        string `json:"clean,omitempty"`
	Type         string `json:"type,omitempty"`
	Required     bool   `json:"required,omitempty"`
	Multiple     bool   `json:"multiple,omitempty"`
	Default      any    `json:"default,omitempty"`
}

type ExtractRequest struct {
	ContentType string
	Content     string
	Fields      []FieldRule
}

// Extract applies CSS, XPath, or JSONPath field rules to one saved document.
// It intentionally has no network or filesystem access and is safe to replay.
func Extract(request ExtractRequest) (map[string]any, error) {
	if strings.TrimSpace(request.Content) == "" {
		return nil, fmt.Errorf("document content is empty")
	}
	result := make(map[string]any, len(request.Fields))
	contentType := strings.ToLower(strings.TrimSpace(request.ContentType))
	var jsonValue any
	var document *goquery.Document
	var htmlRoot *html.Node
	for _, rule := range request.Fields {
		if strings.TrimSpace(rule.Name) == "" || strings.TrimSpace(rule.Selector) == "" {
			return nil, fmt.Errorf("field name and selector are required")
		}
		kind := strings.ToLower(strings.TrimSpace(rule.SelectorType))
		if kind == "" {
			if strings.HasPrefix(strings.TrimSpace(rule.Selector), "$") || contentType == "json" || contentType == "application/json" {
				kind = "jsonpath"
			} else {
				kind = "css"
			}
		}
		var value any
		var err error
		switch kind {
		case "jsonpath", "json_path":
			if jsonValue == nil {
				if err = json.Unmarshal([]byte(request.Content), &jsonValue); err != nil {
					return nil, fmt.Errorf("parse JSON document: %w", err)
				}
			}
			value, err = jsonPathValue(jsonValue, rule.Selector)
			if text, ok := value.(string); ok {
				value = cleanValue(text, rule)
			}
			if items, ok := value.([]any); ok {
				cleaned := make([]any, len(items))
				for i, item := range items {
					cleaned[i] = item
					if text, ok := item.(string); ok {
						cleaned[i] = cleanValue(text, rule)
					}
				}
				value = cleaned
			}
		case "xpath":
			if htmlRoot == nil {
				htmlRoot, err = htmlquery.Parse(strings.NewReader(request.Content))
			}
			if err == nil {
				value, err = xpathValue(htmlRoot, rule)
			}
		default:
			if document == nil {
				document, err = goquery.NewDocumentFromReader(strings.NewReader(request.Content))
			}
			if err == nil {
				value, err = cssValue(document, rule)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", rule.Name, err)
		}
		if isEmptyValue(value) {
			if rule.Required {
				return nil, fmt.Errorf("required field %q is empty", rule.Name)
			}
			value = rule.Default
		}
		value, err = normalizeValue(value, rule)
		if err != nil {
			return nil, fmt.Errorf("normalize %s: %w", rule.Name, err)
		}
		if rule.Required && isEmptyValue(value) {
			return nil, fmt.Errorf("required field %q is empty", rule.Name)
		}
		result[rule.Name] = value
	}
	return result, nil
}

func cssValue(document *goquery.Document, rule FieldRule) (any, error) {
	selection := document.Find(rule.Selector)
	if selection.Length() == 0 {
		return nil, nil
	}
	if rule.Multiple {
		values := make([]any, 0, selection.Length())
		selection.Each(func(_ int, item *goquery.Selection) {
			if rule.Attribute != "" {
				if value, ok := item.Attr(rule.Attribute); ok {
					values = append(values, cleanValue(value, rule))
				}
				return
			}
			values = append(values, cleanValue(item.Text(), rule))
		})
		return values, nil
	}
	selection = selection.First()
	if rule.Attribute != "" {
		value, ok := selection.Attr(rule.Attribute)
		if !ok {
			return nil, nil
		}
		return cleanValue(value, rule), nil
	}
	return cleanValue(selection.Text(), rule), nil
}

func xpathValue(root *html.Node, rule FieldRule) (any, error) {
	if rule.Multiple {
		nodes := htmlquery.Find(root, rule.Selector)
		values := make([]any, 0, len(nodes))
		for _, node := range nodes {
			if rule.Attribute != "" {
				values = append(values, cleanValue(htmlquery.SelectAttr(node, rule.Attribute), rule))
			} else {
				values = append(values, cleanValue(htmlquery.InnerText(node), rule))
			}
		}
		if len(values) == 0 {
			return nil, nil
		}
		return values, nil
	}
	node := htmlquery.FindOne(root, rule.Selector)
	if node == nil {
		return nil, nil
	}
	if rule.Attribute != "" {
		return cleanValue(htmlquery.SelectAttr(node, rule.Attribute), rule), nil
	}
	return cleanValue(htmlquery.InnerText(node), rule), nil
}

func cleanValue(value string, rule FieldRule) string {
	value = strings.TrimSpace(value)
	if rule.Clean == "whitespace" {
		value = strings.Join(strings.Fields(value), " ")
	}
	if rule.Regex != "" {
		if re, err := regexp.Compile(rule.Regex); err == nil {
			matches := re.FindStringSubmatch(value)
			if len(matches) > 1 {
				value = matches[1]
			} else if len(matches) == 0 {
				value = ""
			}
		}
	}
	return strings.TrimSpace(value)
}

func normalizeValue(value any, rule FieldRule) (any, error) {
	if value == nil || rule.Type == "" {
		return value, nil
	}
	switch strings.ToLower(rule.Type) {
	case "string":
		return fmt.Sprint(value), nil
	case "int", "integer":
		return strconv.ParseInt(strings.TrimSpace(fmt.Sprint(value)), 10, 64)
	case "float", "number":
		return strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
	case "bool", "boolean":
		return strconv.ParseBool(strings.TrimSpace(fmt.Sprint(value)))
	case "array":
		if array, ok := value.([]any); ok {
			return array, nil
		}
		return strings.Fields(fmt.Sprint(value)), nil
	default:
		return nil, fmt.Errorf("unsupported field type: %s", rule.Type)
	}
}

func jsonPathValue(root any, path string) (any, error) {
	path = strings.TrimSpace(path)
	if !supportedJSONPath.MatchString(path) {
		return nil, fmt.Errorf("unsupported JSONPath subset")
	}
	if path == "$" {
		return root, nil
	}
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("JSONPath must start with $")
	}
	path = strings.TrimPrefix(path, "$")
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	path = strings.Trim(path, ".")
	current := root
	if path == "" {
		return current, nil
	}
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		switch item := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = item[part]
			if !ok {
				return nil, nil
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(item) {
				return nil, nil
			}
			current = item[index]
		default:
			return nil, nil
		}
	}
	return current, nil
}
