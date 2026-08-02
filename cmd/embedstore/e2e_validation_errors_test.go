package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommandStrictRejectsNormalizedInput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	if err := os.WriteFile(input, []byte(`{"datasetName":"dataset","datasetVersion":"v1","items":[{"id":"item","content":" has surrounding whitespace ","data":{}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(buildCLI(t, dir), "validate", "--input", input, "--strict")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(string(output), "strict validation") {
		t.Fatalf("error output = %q", output)
	}
}

func TestValidateCommandRejectsUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	if err := os.WriteFile(input, []byte(`{"datasetName":"dataset","datasetVersion":"v1","items":[{"content":"valid","data":{}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(buildCLI(t, dir), "validate", "--input", input, "--format", "yaml")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(string(output), "--format must be json") {
		t.Fatalf("error output = %q", output)
	}
}
