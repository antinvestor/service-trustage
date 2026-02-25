package business

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaValidator compiles and caches JSON Schemas for form definition validation.
type SchemaValidator struct {
	mu    sync.RWMutex
	cache map[string]*jsonschema.Schema
	seq   int
}

// NewSchemaValidator creates a new SchemaValidator.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		cache: make(map[string]*jsonschema.Schema),
	}
}

// Validate validates data against the given JSON Schema string.
// Returns nil if the schema is empty (no validation required) or if data is valid.
func (v *SchemaValidator) Validate(schemaJSON string, data string) error {
	if schemaJSON == "" {
		return nil
	}

	schema, err := v.getOrCompile(schemaJSON)
	if err != nil {
		return fmt.Errorf("%w: compile schema: %w", ErrSchemaValidationFailed, err)
	}

	var dataVal any
	if unmarshalErr := json.Unmarshal([]byte(data), &dataVal); unmarshalErr != nil {
		return fmt.Errorf("%w: invalid JSON data: %w", ErrInvalidFormData, unmarshalErr)
	}

	if validateErr := schema.Validate(dataVal); validateErr != nil {
		return fmt.Errorf("%w: %w", ErrSchemaValidationFailed, validateErr)
	}

	return nil
}

// ValidateSchema checks that a schema string is valid JSON Schema.
// Returns nil if the schema is empty or valid.
func (v *SchemaValidator) ValidateSchema(schemaJSON string) error {
	if schemaJSON == "" {
		return nil
	}

	_, err := v.compile(schemaJSON)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSchemaValidationFailed, err)
	}

	return nil
}

func (v *SchemaValidator) getOrCompile(schemaJSON string) (*jsonschema.Schema, error) {
	v.mu.RLock()
	if cached, ok := v.cache[schemaJSON]; ok {
		v.mu.RUnlock()
		return cached, nil
	}
	v.mu.RUnlock()

	schema, err := v.compile(schemaJSON)
	if err != nil {
		return nil, err
	}

	v.mu.Lock()
	v.cache[schemaJSON] = schema
	v.mu.Unlock()

	return schema, nil
}

func (v *SchemaValidator) compile(schemaJSON string) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}

	v.mu.Lock()
	v.seq++
	resourceURL := fmt.Sprintf("urn:formstore:schema:%d", v.seq)
	v.mu.Unlock()

	c := jsonschema.NewCompiler()
	if addErr := c.AddResource(resourceURL, doc); addErr != nil {
		return nil, fmt.Errorf("add schema resource: %w", addErr)
	}

	return c.Compile(resourceURL)
}
