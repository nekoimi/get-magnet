package dataset_repo

import "testing"

func TestValidateSchema(t *testing.T) {
	valid := SchemaInput{UniqueKeyFields: []string{"sku"}, EmptyValuePolicy: "preserve", Fields: []FieldInput{{Key: "sku", Label: "SKU", Type: "string", Required: true}, {Key: "title", Label: "Title", Type: "string"}}}
	if err := ValidateSchema(valid); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(*SchemaInput)
	}{
		{"missing key", func(s *SchemaInput) { s.UniqueKeyFields = []string{"other"} }},
		{"optional key", func(s *SchemaInput) { s.Fields[0].Required = false }},
		{"duplicate field", func(s *SchemaInput) { s.Fields[1].Key = "sku" }},
		{"unsupported type", func(s *SchemaInput) { s.Fields[1].Type = "object" }},
		{"invalid policy", func(s *SchemaInput) { s.EmptyValuePolicy = "skip" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copy := valid
			copy.Fields = append([]FieldInput(nil), valid.Fields...)
			copy.UniqueKeyFields = append([]string(nil), valid.UniqueKeyFields...)
			tc.edit(&copy)
			if err := ValidateSchema(copy); err == nil {
				t.Fatal("expected schema rejection")
			}
		})
	}
}
