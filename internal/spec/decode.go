package spec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Format is a supported project and pack document format.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// DecodeProject strictly decodes one project document.
func DecodeProject(input []byte, format Format) (Project, error) {
	var project Project
	if err := decodeOne(input, format, &project); err != nil {
		return Project{}, fmt.Errorf("decode project: %w", err)
	}
	return project, nil
}

// DecodeCapabilityPack strictly decodes one capability pack document.
func DecodeCapabilityPack(input []byte, format Format) (CapabilityPack, error) {
	var pack CapabilityPack
	if err := decodeOne(input, format, &pack); err != nil {
		return CapabilityPack{}, fmt.Errorf("decode capability pack: %w", err)
	}
	return pack, nil
}

// DetectFormat selects JSON for object input and YAML otherwise.
func DetectFormat(input []byte) Format {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return FormatJSON
	}
	return FormatYAML
}

// ParseFormat converts a CLI or file-extension value into a Format.
func ParseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimPrefix(value, ".")) {
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unsupported format %q", value)
	}
}

func decodeOne(input []byte, format Format, target any) error {
	switch format {
	case FormatJSON:
		if err := rejectDuplicateJSONKeys(input); err != nil {
			return err
		}
		decoder := json.NewDecoder(bytes.NewReader(input))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(target); err != nil {
			return err
		}
		return requireEOF(decoder.Decode(new(any)))
	case FormatYAML:
		decoder := yaml.NewDecoder(bytes.NewReader(input))
		decoder.KnownFields(true)
		if err := decoder.Decode(target); err != nil {
			return err
		}
		var extra any
		return requireEOF(decoder.Decode(&extra))
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func rejectDuplicateJSONKeys(input []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	if err := inspectJSONValue(decoder); err != nil {
		return err
	}
	return requireEOF(decoder.Decode(new(any)))
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key must be a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("unexpected JSON delimiter")
	}
}

func requireEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("multiple documents are not supported")
}
