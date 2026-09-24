package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Definition is the versioned, provider-neutral workflow document. Nodes are
// deliberately represented as JSON objects so new node configuration can be
// added without changing the database model or the browser protocol.
type Definition struct {
	Trigger      Trigger           `json:"trigger"`
	Acquire      []Node            `json:"acquire,omitempty"`
	Nodes        []Node            `json:"nodes"`
	InputSchema  json.RawMessage   `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage   `json:"output_schema,omitempty"`
	Credentials  map[string]string `json:"credentials,omitempty"`
}

type Trigger struct {
	Type  string `json:"type"`
	Cron  string `json:"cron,omitempty"`
	URL   string `json:"url,omitempty"`
	Input string `json:"input,omitempty"`
}

type Node struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

var nodeTypes = map[string]struct{}{
	"trigger": {}, "acquire": {}, "navigate": {}, "discover": {},
	"extract": {}, "transform": {}, "validate": {}, "deduplicate": {},
	"persist": {}, "event": {}, "list": {}, "detail": {}, "pagination": {},
}

var triggerTypes = map[string]struct{}{"manual": {}, "cron": {}, "api": {}, "webhook": {}}

// ParseDefinition parses and validates a persisted workflow definition.
func ParseDefinition(raw string) (Definition, error) {
	var definition Definition
	if strings.TrimSpace(raw) == "" {
		return definition, fmt.Errorf("workflow definition is empty")
	}
	if err := json.Unmarshal([]byte(raw), &definition); err != nil {
		return definition, fmt.Errorf("invalid workflow definition: %w", err)
	}
	if err := definition.Validate(); err != nil {
		return definition, err
	}
	return definition, nil
}

func (d Definition) Validate() error {
	if strings.TrimSpace(d.Trigger.Type) == "" {
		return fmt.Errorf("trigger.type is required")
	}
	if _, ok := triggerTypes[strings.ToLower(d.Trigger.Type)]; !ok {
		return fmt.Errorf("unsupported trigger type: %s", d.Trigger.Type)
	}
	if len(d.Nodes) == 0 && len(d.Acquire) == 0 {
		return fmt.Errorf("workflow nodes are required")
	}
	seen := make(map[string]struct{}, len(d.Nodes)+len(d.Acquire))
	for _, node := range append(append([]Node{}, d.Acquire...), d.Nodes...) {
		if strings.TrimSpace(node.Name) == "" {
			return fmt.Errorf("node.name is required")
		}
		if _, ok := seen[node.Name]; ok {
			return fmt.Errorf("duplicate node name: %s", node.Name)
		}
		seen[node.Name] = struct{}{}
		if _, ok := nodeTypes[strings.ToLower(node.Type)]; !ok {
			return fmt.Errorf("unsupported node type %q", node.Type)
		}
		if err := validateNode(node); err != nil {
			return fmt.Errorf("node %s: %w", node.Name, err)
		}
	}
	if err := validateSchema(d.InputSchema, "input_schema"); err != nil {
		return err
	}
	if err := validateSchema(d.OutputSchema, "output_schema"); err != nil {
		return err
	}
	for name, ref := range d.Credentials {
		if strings.TrimSpace(name) == "" || !strings.HasPrefix(strings.TrimSpace(ref), "${secret:") || !strings.HasSuffix(strings.TrimSpace(ref), "}") {
			return fmt.Errorf("credential %q must use ${secret:name} reference", name)
		}
	}
	return nil
}

func validateNode(node Node) error {
	if strings.EqualFold(node.Type, "extract") {
		fields, ok := node.Config["fields"]
		if !ok || fields == nil {
			return fmt.Errorf("extract.fields is required")
		}
		if _, ok := fields.([]any); !ok {
			return fmt.Errorf("extract.fields must be an array")
		}
	}
	return nil
}

func validateSchema(raw json.RawMessage, name string) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("%s must be a JSON object: %w", name, err)
	}
	if value == nil {
		return fmt.Errorf("%s must be a JSON object", name)
	}
	return nil
}
