package embedstore

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestMemoryStoreSearchVectorOrdersAndFilters(t *testing.T) {
	t.Parallel()
	minimum := float32(0.8)
	store := mustMemoryStore(t, Manifest{Embedding: "openai/model", Dimensions: 2, ItemCount: 4, Normalized: true}, []Item{
		{ID: "second", Content: "second", Data: []byte(`{"n":2}`)},
		{ID: "first", Content: "first", Data: []byte(`{"n":1}`)},
		{ID: "low", Content: "low", Data: []byte(`{"n":3}`)},
		{ID: "negative", Content: "negative", Data: []byte(`{"n":4}`)},
	}, []float32{1, 0, 1, 0, 0.8, 0.6, -1, 0})

	got, err := store.SearchVector(context.Background(), []float32{2, 0}, SearchOptions{Limit: 3, MinScore: &minimum})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("result count = %d, want 3", len(got))
	}
	for i, want := range []string{"second", "first", "low"} {
		if got[i].Rank != i+1 || got[i].ID != want {
			t.Fatalf("result %d = %#v, want rank %d id %q", i, got[i], i+1, want)
		}
	}
}

func TestMemoryStoreSearchVectorDefaultLimitAndDimensionErrors(t *testing.T) {
	t.Parallel()
	items := make([]Item, 6)
	vectors := make([]float32, 12)
	for i := range items {
		items[i] = Item{ID: string(rune('a' + i)), Data: []byte(`{}`)}
		vectors[i*2] = 1
	}
	store := mustMemoryStore(t, Manifest{Embedding: "openai/model", Dimensions: 2, ItemCount: 6, Normalized: true}, items, vectors)

	got, err := store.SearchVector(context.Background(), []float32{1, 0}, SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != DefaultSearchLimit {
		t.Fatalf("default result count = %d, want %d", len(got), DefaultSearchLimit)
	}
	_, err = store.SearchVector(context.Background(), []float32{1, 0, 0}, SearchOptions{})
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Fatalf("dimension error = %v, want ErrDimensionMismatch", err)
	}
	_, err = store.SearchVector(context.Background(), []float32{0, 0}, SearchOptions{})
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("zero query error = %v, want ErrInvalidQuery", err)
	}
}

func TestMemoryStoreConcurrentSearch(t *testing.T) {
	store := mustMemoryStore(t, Manifest{Embedding: "openai/model", Dimensions: 2, ItemCount: 2, Normalized: true}, []Item{{ID: "x", Data: []byte(`{}`)}, {ID: "y", Data: []byte(`{}`)}}, []float32{1, 0, 0, 1})
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := store.SearchVector(context.Background(), []float32{1, 0}, SearchOptions{})
			if err != nil || len(got) == 0 || got[0].ID != "x" {
				t.Errorf("SearchVector() = %#v, %v", got, err)
			}
		}()
	}
	group.Wait()
}

func TestEngineChecksEmbedderCompatibilityAndSearches(t *testing.T) {
	t.Parallel()
	store := mustMemoryStore(t, Manifest{Embedding: "openai/model", Dimensions: 2, ItemCount: 1, Normalized: true}, []Item{{ID: "x", Data: []byte(`{}`)}}, []float32{1, 0})
	_, err := NewEngine(store, fakeEmbedder{embedding: "other/model", dimensions: 2, vector: []float32{1, 0}})
	if !errors.Is(err, ErrModelMismatch) {
		t.Fatalf("model error = %v, want ErrModelMismatch", err)
	}
	_, err = NewEngine(store, fakeEmbedder{embedding: "openai/model", dimensions: 3, vector: []float32{1, 0}})
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Fatalf("dimension error = %v, want ErrDimensionMismatch", err)
	}
	engine, err := NewEngine(store, fakeEmbedder{embedding: "openai/model", dimensions: 2, vector: []float32{2, 0}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := engine.Search(context.Background(), "query", SearchOptions{})
	if err != nil || len(got) != 1 || got[0].ID != "x" {
		t.Fatalf("Search() = %#v, %v", got, err)
	}
	_, err = engine.Search(context.Background(), "  ", SearchOptions{})
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("blank query error = %v, want ErrInvalidQuery", err)
	}
}

func mustMemoryStore(t *testing.T, manifest Manifest, items []Item, vectors []float32) *MemoryStore {
	t.Helper()
	store, err := NewMemoryStore(manifest, items, vectors)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

type fakeEmbedder struct {
	embedding  string
	dimensions int
	vector     []float32
}

func (f fakeEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{f.vector}, nil
}
func (f fakeEmbedder) Embedding() string { return f.embedding }
func (f fakeEmbedder) Dimensions() int   { return f.dimensions }
