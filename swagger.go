package httputil

import (
	swagger "go.lumeweb.com/gswagger"
	"strings"
)

type SwaggerOption func(*swagger.Definitions)

type FieldSchema interface {
	SortableFields() []string
	FilterOperators() map[string][]string // field -> []operator
}

func WithSchema(schema FieldSchema) SwaggerOption {
	return func(d *swagger.Definitions) {
		// Add sort parameters
		sortFields := schema.SortableFields()
		d.Querystring["_sort"] = swagger.Parameter{
			Description: "Sort by fields: " + strings.Join(sortFields, ", "),
			Schema: &swagger.Schema{
				Value: map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
						"enum": sortFields,
					},
				},
			},
		}

		d.Querystring["_order"] = swagger.Parameter{
			Description: "Sort direction",
			Schema: &swagger.Schema{
				Value: map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
						"enum": []string{"asc", "desc"},
					},
				},
			},
		}

		// Add filter parameters
		operators := schema.FilterOperators()
		for field, ops := range operators {
			d.Querystring[field] = swagger.Parameter{
				Description: "Filter operators: " + strings.Join(ops, ", "),
				Schema: &swagger.Schema{
					Value: map[string]any{
						"type":       "object",
						"properties": createOperatorSchemas(ops),
					},
				},
			}
		}
	}
}

type SchemaProvider interface {
	ForType(any) FieldSchema
}

func createOperatorSchemas(ops []string) map[string]any {
	schemas := make(map[string]any)
	for _, op := range ops {
		schemas[op] = map[string]any{
			"type":        "string", // Actual type resolved by provider
			"description": operatorDocs[op],
		}
	}
	return schemas
}

var operatorDocs = map[string]string{
	"eq": "Equal to",
	"ne": "Not equal to",
	"gt": "Greater than",
	// ... other operator descriptions
}
