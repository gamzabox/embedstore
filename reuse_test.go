package embedstore

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestBuildFileWithReuseMergesInputAndExistingItems(t *testing.T) {
	dir := t.TempDir()
	reuse := filepath.Join(dir, "old.embed")
	out := filepath.Join(dir, "new.embed")
	old := Manifest{FormatVersion: 1, DatasetName: "dataset", DatasetVersion: "old", Embedding: "fake/model", Dimensions: 2, VectorType: "float32", Normalized: true, ItemCount: 2, ContentIncluded: true}
	if err := WriteFile(reuse, old, []Item{{ID: "keep", Content: "old", Data: json.RawMessage(`{"from":"old"}`)}, {ID: "replace", Content: "old replace", Data: json.RawMessage(`{}`)}}, []float32{1, 0, 0, 1}); err != nil {
		t.Fatal(err)
	}
	input := Dataset{DatasetName: "dataset", DatasetVersion: "new", Items: []Item{{ID: "replace", Content: "new replace", Data: json.RawMessage(`{"from":"new"}`)}, {ID: "added", Content: "new", Data: json.RawMessage(`{}`)}}}
	fake := &buildFake{vectors: [][]float32{{1, 0}, {0, 1}, {1, 1}}}
	if err := BuildFileWithReuse(context.Background(), input, reuse, out, BuildOptions{Embedder: fake, Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	s, err := LoadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	items := s.(*MemoryStore).Items()
	if len(items) != 3 || items[0].ID != "replace" || items[1].ID != "added" || items[2].ID != "keep" {
		t.Fatalf("items %#v", items)
	}
}
func TestBuildFileWithReuseRejectsDatasetAndMissingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.embed")
	m := Manifest{FormatVersion: 1, DatasetName: "one", DatasetVersion: "v", Embedding: "fake/model", Dimensions: 2, VectorType: "float32", Normalized: true, ItemCount: 1, ContentIncluded: false}
	if err := WriteFile(path, m, []Item{{ID: "x", Data: json.RawMessage(`{}`)}}, []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	d := Dataset{DatasetName: "two", DatasetVersion: "v", Items: []Item{{ID: "x", Content: "x", Data: json.RawMessage(`{}`)}}}
	err := BuildFileWithReuse(context.Background(), d, path, filepath.Join(dir, "out.embed"), BuildOptions{Embedder: &buildFake{vectors: [][]float32{{1, 0}}}})
	if err == nil {
		t.Fatal("expected reuse failure")
	}
}
