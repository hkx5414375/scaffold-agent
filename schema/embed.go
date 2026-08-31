// Package schema embeds public Scaffold Agent JSON schemas.
package schema

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// FS contains all versioned public schemas.
//
//go:embed v1alpha1/*.json
var FS embed.FS

// Read returns one embedded schema by version and file name.
func Read(version, name string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s", version, name)
	content, err := FS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema %q: %w", path, err)
	}
	return content, nil
}

// Validate checks a value against one embedded JSON Schema document.
func Validate(version, name string, value any) error {
	content, err := Read(version, name)
	if err != nil {
		return err
	}
	var document any
	if err := json.Unmarshal(content, &document); err != nil {
		return fmt.Errorf("decode schema %q: %w", name, err)
	}
	identifier := fmt.Sprintf("https://scaffold-agent.dev/schema/%s/%s", version, name)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource(identifier, document); err != nil {
		return fmt.Errorf("add schema resource %q: %w", name, err)
	}
	compiled, err := compiler.Compile(identifier)
	if err != nil {
		return fmt.Errorf("compile schema %q: %w", name, err)
	}
	encodedValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode value for schema %q: %w", name, err)
	}
	var normalizedValue any
	if err := json.Unmarshal(encodedValue, &normalizedValue); err != nil {
		return fmt.Errorf("decode value for schema %q: %w", name, err)
	}
	if err := compiled.Validate(normalizedValue); err != nil {
		return fmt.Errorf("validate with schema %q: %w", name, err)
	}
	return nil
}
