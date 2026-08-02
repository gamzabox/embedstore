package main_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBuildAndSearchCommandsAgainstLocalOpenAI(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		var request struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "test-model" || request.Dimensions != 2 {
			t.Fatalf("request %#v", request)
		}
		data := make([]map[string]any, len(request.Input))
		for i, input := range request.Input {
			vector := []float32{1, 0}
			if input == "second" {
				vector = []float32{0, 1}
			}
			data[i] = map[string]any{"index": i, "embedding": vector}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	output := filepath.Join(dir, "data.embed")
	if err := os.WriteFile(input, []byte(`{"datasetName":"d","datasetVersion":"v","items":[{"id":"one","content":"first","data":{}},{"id":"two","content":"second","data":{}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	binary := buildCLI(t, dir)
	env := append(os.Environ(), "OPENAI_API_KEY=test-key", "OPENAI_BASE_URL="+server.URL)
	build := exec.Command(binary, "build", "--input", input, "--output", output, "--embedding", "openai/test-model", "--dimensions", "2", "--batch-size", "1")
	build.Env = env
	stderr := new(strings.Builder)
	build.Stderr = stderr
	if out, err := build.Output(); err != nil {
		t.Fatalf("build: %v\nstderr=%s", err, stderr.String())
	} else if !strings.Contains(string(out), "Built") {
		t.Fatalf("build output=%q", out)
	}
	search := exec.Command(binary, "search", "--file", output, "--query", "first", "--output", "json")
	search.Env = env
	result, err := search.Output()
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err = json.Unmarshal(result, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Results) != 2 || payload.Results[0].ID != "one" {
		t.Fatalf("results=%s", result)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
}
