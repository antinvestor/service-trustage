package workflows

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	s := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, s); err != nil {
		return nil, "", fmt.Errorf("parse %s: %w", path, err)
	}
	nameVal, ok := s.Fields["name"]
	if !ok || nameVal.GetStringValue() == "" {
		return nil, "", fmt.Errorf("%s: missing or empty 'name' field", path)
	}
	return s, nameVal.GetStringValue(), nil
}

func dslHash(s *structpb.Struct) string {
	b, err := protojson.MarshalOptions{Indent: "", Multiline: false}.Marshal(s)
	if err != nil {
		return ""
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	sorted, _ := json.Marshal(m)
	h := sha256.Sum256(sorted)
	return hex.EncodeToString(h[:])
}
