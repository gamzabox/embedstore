package embedstore

import (
	"bytes"
	"encoding/json"
	"testing"
)

func FuzzParseDataset(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		[]byte(`{`),
		[]byte(`not JSON`),
		[]byte(`{"datasetName":"knowledge","datasetVersion":"1","items":[{"content":"hello","data":{"topic":"test"}}]}`),
		[]byte(`{"datasetName":"knowledge","datasetVersion":"1","items":[{"id":"same","content":"one","data":{}},{"id":"same","content":"two","data":{}}]}`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		dataset, err := ParseDataset(bytes.NewReader(data), ValidationOptions{})
		if err != nil {
			return
		}
		if dataset.DatasetName == "" || dataset.DatasetVersion == "" || len(dataset.Items) == 0 {
			t.Fatalf("successful parse returned incomplete dataset: %#v", dataset)
		}
		for index, item := range dataset.Items {
			if item.ID == "" || item.Content == "" || !json.Valid(item.Data) {
				t.Fatalf("successful parse returned invalid item %d: %#v", index, item)
			}
		}
	})
}
