package main_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/embedstore/embedstore"
)

func TestBuildCommandReuseMergesItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		data := make([]map[string]any, len(request.Input))
		for i := range request.Input {
			data[i] = map[string]any{"index": i, "embedding": []float32{1, 0}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()
	dir := t.TempDir()
	reuse := filepath.Join(dir, "old.embed")
	out := filepath.Join(dir, "new.embed")
	input := filepath.Join(dir, "input.json")
	m := embedstore.Manifest{FormatVersion: 1, DatasetName: "d", DatasetVersion: "old", Embedding: "openai/test-model", Dimensions: 2, VectorType: "float32", Normalized: true, ItemCount: 2, CreatedAt: time.Now().UTC(), ContentIncluded: true}
	items := []embedstore.Item{{ID: "keep", Content: "keep", Data: json.RawMessage(`{}`)}, {ID: "replace", Content: "old", Data: json.RawMessage(`{}`)}}
	if err := embedstore.WriteFile(reuse, m, items, []float32{1, 0, 0, 1}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte(`{"datasetName":"d","datasetVersion":"new","items":[{"id":"replace","content":"new","data":{}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(buildCLI(t, dir), "build", "--input", input, "--output", out, "--embedding", "openai/test-model", "--dimensions", "2", "--reuse", reuse)
	command.Env = append(os.Environ(), "OPENAI_API_KEY=test", "OPENAI_BASE_URL="+server.URL)
	if stderr, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, stderr)
	}
	inspect := exec.Command(buildCLI(t, dir), "inspect", "--file", out, "--list")
	listed, err := inspect.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(listed), "0  replace\n1  keep\n"; got != want {
		t.Fatalf("list=%q want=%q", got, want)
	}
}
