package workflows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDSLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-workflow.json")
	content := `{
		"version": "1.0",
		"name": "test.workflow",
		"schedule": {"cron": "30s", "active": true},
		"steps": [{"id": "step1", "type": "call", "name": "Do thing",
			"call": {"action": "http.request", "input": {"url": "http://example.com", "method": "POST"}}}]
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	dslStruct, name, err := parseDSLFile(path)
	if err != nil {
		t.Fatalf("parseDSLFile: %v", err)
	}
	if name != "test.workflow" {
		t.Errorf("name = %q, want %q", name, "test.workflow")
	}
	if dslStruct == nil {
		t.Fatal("dslStruct is nil")
	}
	nameField := dslStruct.Fields["name"].GetStringValue()
	if nameField != "test.workflow" {
		t.Errorf("struct name = %q, want %q", nameField, "test.workflow")
	}
}

func TestParseDSLFile_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"version":"1.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := parseDSLFile(path)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestDSLHash_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hash-test.json")
	content := `{"version":"1.0","name":"hash.test","steps":[]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s1, _, _ := parseDSLFile(path)
	s2, _, _ := parseDSLFile(path)
	h1 := dslHash(s1)
	h2 := dslHash(s2)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %s != %s", h1, h2)
	}
	if h1 == "" {
		t.Error("hash is empty")
	}
}
