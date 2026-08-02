package embedstore

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewMemoryStoreRejectsDuplicateIDs(t *testing.T) {
	_, err := NewMemoryStore(Manifest{Dimensions: 2, ItemCount: 2}, []Item{{ID: "duplicate", Data: json.RawMessage(`{}`)}, {ID: "duplicate", Data: json.RawMessage(`{}`)}}, []float32{1, 0, 0, 1})
	if !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("error=%v want ErrInvalidFile", err)
	}
}
