package embedstore

import (
	"strings"
	"testing"
)

func TestParseDatasetStrictRejectsValuesThatNeedNormalization(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "dataset name whitespace",
			json: `{"datasetName":" dataset ","datasetVersion":"v1","items":[{"id":"item","content":"content","data":{}}]}`,
		},
		{
			name: "explicit id whitespace",
			json: `{"datasetName":"dataset","datasetVersion":"v1","items":[{"id":" item ","content":"content","data":{}}]}`,
		},
		{
			name: "content whitespace",
			json: `{"datasetName":"dataset","datasetVersion":"v1","items":[{"id":"item","content":" two  spaces ","data":{}}]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDataset(strings.NewReader(tc.json), ValidationOptions{Strict: true})
			if err == nil {
				t.Fatal("expected strict validation error")
			}
		})
	}
}

func TestParseDatasetStrictAllowsCanonicalInput(t *testing.T) {
	raw := `{"datasetName":"dataset","datasetVersion":"v1","items":[{"id":"item-1","content":"canonical content","data":{"rank":1}}]}`
	if _, err := ParseDataset(strings.NewReader(raw), ValidationOptions{Strict: true}); err != nil {
		t.Fatalf("ParseDataset() error = %v", err)
	}
}
