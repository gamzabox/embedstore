package embedstore

import (
	"encoding/json"
	"testing"
)

func TestMemoryStoreCopiesItemMetadata(t *testing.T) {
	data := json.RawMessage(`{"value":1}`)
	store, err := NewMemoryStore(Manifest{Dimensions: 2, ItemCount: 1}, []Item{{ID: "item", Data: data}}, []float32{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	data[9] = '9'
	if got := string(store.Items()[0].Data); got != `{"value":1}` {
		t.Fatalf("stored data changed: %s", got)
	}
}
