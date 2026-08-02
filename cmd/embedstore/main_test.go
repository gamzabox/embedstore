package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gamzabox/embedstore"
)

func TestInspectOutputListAndID(t *testing.T) {
	path := writeInspectFixture(t)
	list, err := inspectOutput(path, true, "", "text", false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(list), "0  alpha\n1  beta\n"; got != want {
		t.Fatalf("list = %q, want %q", got, want)
	}
	detail, err := inspectOutput(path, false, "beta", "json", true)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		ID      string          `json:"id"`
		Index   int             `json:"index"`
		Content string          `json:"content"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(detail, &got); err != nil {
		t.Fatal(err)
	}
	var data map[string]int
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatal(err)
	}
	if got.ID != "beta" || got.Index != 1 || got.Content != "second" || data["rank"] != 2 {
		t.Fatalf("detail = %#v", got)
	}
}

func TestInspectOutputRejectsMutuallyExclusiveAndMissingID(t *testing.T) {
	path := writeInspectFixture(t)
	if _, err := inspectOutput(path, true, "alpha", "text", false); err == nil {
		t.Fatal("expected list/id conflict")
	}
	if _, err := inspectOutput(path, false, "missing", "text", false); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing id error = %v", err)
	}
}

func writeInspectFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.embed")
	manifest := embedstore.Manifest{FormatVersion: 1, DatasetName: "dataset", DatasetVersion: "v1", Embedding: "openai/text-embedding-3-small", Dimensions: 2, VectorType: "float32", Normalized: true, ItemCount: 2, CreatedAt: time.Now().UTC(), ContentIncluded: true}
	items := []embedstore.Item{{ID: "alpha", Content: "first", Data: json.RawMessage(`{"rank":1}`)}, {ID: "beta", Content: "second", Data: json.RawMessage(`{"rank":2}`)}}
	if err := embedstore.WriteFile(path, manifest, items, []float32{1, 0, 0, 1}); err != nil {
		t.Fatal(err)
	}
	return path
}
