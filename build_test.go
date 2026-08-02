package embedstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFileBatchesAndAtomicallyReplaces(t *testing.T) {
	dataset := Dataset{DatasetName: "d", DatasetVersion: "v", Items: []Item{{ID: "a", Content: "a", Data: json.RawMessage(`{}`)}, {ID: "b", Content: "b", Data: json.RawMessage(`{}`)}, {ID: "c", Content: "c", Data: json.RawMessage(`{}`)}}}
	fake := &buildFake{vectors: [][]float32{{2, 0}, {0, 2}, {1, 1}}}
	path := filepath.Join(t.TempDir(), "data.embed")
	if err := BuildFile(context.Background(), dataset, path, BuildOptions{Embedder: fake, BatchSize: 2, Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	if len(fake.calls) != 2 || len(fake.calls[0]) != 2 || len(fake.calls[1]) != 1 {
		t.Fatalf("batches %#v", fake.calls)
	}
	store, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Count() != 3 || store.Manifest().Dimensions != 2 {
		t.Fatalf("store %#v", store.Manifest())
	}
	if _, err = os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temporary file remains")
	}
}
func TestBuildFilePreservesExistingOutputOnEmbeddingFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.embed")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	d := Dataset{DatasetName: "d", DatasetVersion: "v", Items: []Item{{ID: "a", Content: "a", Data: json.RawMessage(`{}`)}}}
	err := BuildFile(context.Background(), d, path, BuildOptions{Embedder: &buildFake{err: context.Canceled}, Overwrite: true})
	if err == nil {
		t.Fatal("expected failure")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "old" {
		t.Fatal("output replaced")
	}
}

type buildFake struct {
	vectors [][]float32
	calls   [][]string
	err     error
}

func (f *buildFake) Embed(_ context.Context, in []string) ([][]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.calls = append(f.calls, append([]string(nil), in...))
	n := len(in)
	out := append([][]float32(nil), f.vectors[:n]...)
	f.vectors = f.vectors[n:]
	return out, nil
}
func (f *buildFake) Embedding() string { return "fake/model" }
func (f *buildFake) Dimensions() int   { return 2 }
