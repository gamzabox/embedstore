package embedstore

import (
	"strings"
	"testing"
)

func TestStrictValidationRejectsWhitespaceNormalization(t *testing.T) {
	input := `{"datasetName":" dataset ","datasetVersion":"v","items":[{"id":" id ","content":" hello ","data":{}}]}`
	if _, err := ParseDataset(strings.NewReader(input), ValidationOptions{}); err != nil {
		t.Fatalf("non-strict error: %v", err)
	}
	if _, err := ParseDataset(strings.NewReader(input), ValidationOptions{Strict: true}); err == nil {
		t.Fatal("expected strict validation error")
	}
}
