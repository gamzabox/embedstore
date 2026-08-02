package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gamzabox/embedstore"
)

func TestInspectAndVerifyCommands(t *testing.T) {
	dir := t.TempDir()
	path := writeE2EEmbed(t, dir)
	binary := buildCLI(t, dir)
	inspect := exec.Command(binary, "inspect", "--file", path, "--list")
	output, err := inspect.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(output), "0  alpha\n1  beta\n"; got != want {
		t.Fatalf("list=%q want=%q", got, want)
	}
	verify := exec.Command(binary, "verify", "--file", path)
	output, err = verify.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "Verification successful") {
		t.Fatalf("verify=%q", output)
	}
}
func TestVerifyCommandRejectsChecksumCorruption(t *testing.T) {
	dir := t.TempDir()
	path := writeE2EEmbed(t, dir)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[10] ^= 0xff
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(buildCLI(t, dir), "verify", "--file", path)
	stderr := new(strings.Builder)
	command.Stderr = stderr
	if err = command.Run(); err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}
func writeE2EEmbed(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fixture.embed")
	m := embedstore.Manifest{FormatVersion: 1, DatasetName: "d", DatasetVersion: "v", Embedding: "openai/model", Dimensions: 2, VectorType: "float32", Normalized: true, ItemCount: 2, CreatedAt: time.Now().UTC(), ContentIncluded: true}
	items := []embedstore.Item{{ID: "alpha", Content: "first", Data: json.RawMessage(`{}`)}, {ID: "beta", Content: "second", Data: json.RawMessage(`{}`)}}
	if err := embedstore.WriteFile(path, m, items, []float32{1, 0, 0, 1}); err != nil {
		t.Fatal(err)
	}
	return path
}
