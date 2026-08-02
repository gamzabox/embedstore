package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidateCommandJSONSummary(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	if err := os.WriteFile(input, []byte(`{"datasetName":"dataset","datasetVersion":"v1","items":[{"content":"hello","data":{}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	binary := buildCLI(t, dir)
	command := exec.Command(binary, "validate", "--input", input, "--format", "json", "--output", "json")
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		ItemCount       int `json:"ItemCount"`
		EstimatedTokens int `json:"EstimatedTokens"`
	}
	if err := json.Unmarshal(output, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.ItemCount != 1 || summary.EstimatedTokens < 1 {
		t.Fatalf("summary %#v", summary)
	}
}
func TestValidateCommandRejectsInvalidInput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	if err := os.WriteFile(input, []byte(`{"datasetName":"","items":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(buildCLI(t, dir), "validate", "--input", input)
	if err := command.Run(); err == nil {
		t.Fatal("expected non-zero exit")
	}
}
func buildCLI(t *testing.T, dir string) string {
	t.Helper()
	binary := filepath.Join(dir, "embedstore")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Dir = "."
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	return binary
}
