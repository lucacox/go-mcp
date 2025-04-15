package protocol

import (
	"encoding/json"
)

// JSONSchema represents the structure of a JSON Schema for validation
type JSONSchema struct {
	Type                 string                 `json:"type,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Default              interface{}            `json:"default,omitempty"`

	// Numeric validation
	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMinimum *bool    `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *bool    `json:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"`

	// String validation
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	// Array validation
	MinItems    *int  `json:"minItems,omitempty"`
	MaxItems    *int  `json:"maxItems,omitempty"`
	UniqueItems *bool `json:"uniqueItems,omitempty"`

	// Object validation
	MinProperties *int `json:"minProperties,omitempty"`
	MaxProperties *int `json:"maxProperties,omitempty"`

	// Logical combinators
	AllOf []*JSONSchema `json:"allOf,omitempty"`
	AnyOf []*JSONSchema `json:"anyOf,omitempty"`
	OneOf []*JSONSchema `json:"oneOf,omitempty"`
	Not   *JSONSchema   `json:"not,omitempty"`
}

// NewJSONSchemaFromRaw creates a new JSONSchema from raw JSON data
func NewJSONSchemaFromRaw(data json.RawMessage) (*JSONSchema, error) {
	var schema JSONSchema
	err := json.Unmarshal(data, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

// ObjectSchema creates a new JSONSchema for an object type with the given properties
func ObjectSchema(properties map[string]*JSONSchema, required []string) *JSONSchema {
	return &JSONSchema{
		Type:       "object",
		Required:   required,
		Properties: properties,
	}
}

// StringSchema creates a new JSONSchema for a string type
func StringSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "string",
		Description: description,
	}
}

// NumberSchema creates a new JSONSchema for a number type
func NumberSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "number",
		Description: description,
	}
}

// IntegerSchema creates a new JSONSchema for an integer type
func IntegerSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "integer",
		Description: description,
	}
}

// BooleanSchema creates a new JSONSchema for a boolean type
func BooleanSchema(description string) *JSONSchema {
	return &JSONSchema{
		Type:        "boolean",
		Description: description,
	}
}

// ArraySchema creates a new JSONSchema for an array type with the given item schema
func ArraySchema(items *JSONSchema) *JSONSchema {
	return &JSONSchema{
		Type:  "array",
		Items: items,
	}
}
