package workflows

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func parseDSLFile(path string) (*structpb.Struct, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	return parseDSL(raw, path)
}

// parseDSL decodes a workflow document; label names it in errors.
func parseDSL(raw []byte, label string) (*structpb.Struct, string, error) {
	s := &structpb.Struct{}
	if unmarshalErr := protojson.Unmarshal(raw, s); unmarshalErr != nil {
		return nil, "", fmt.Errorf("parse %s: %w", label, unmarshalErr)
	}
	nameVal, ok := s.GetFields()["name"]
	if !ok || nameVal.GetStringValue() == "" {
		return nil, "", fmt.Errorf("%s: missing or empty 'name' field", label)
	}
	return s, nameVal.GetStringValue(), nil
}
