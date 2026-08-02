package embedstore

import (
	"strings"
	"testing"
)

func TestParseDatasetGeneratesStableIDsAndCanonicalData(t *testing.T) {
	raw := `{"datasetName":"knowledge","datasetVersion":"1","items":[{"content":"  hello   world ","data":{"b":2,"a":1}}]}`
	one, err := ParseDataset(strings.NewReader(raw), ValidationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	two, err := ParseDataset(strings.NewReader(raw), ValidationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if one.Items[0].ID == "" || one.Items[0].ID != two.Items[0].ID {
		t.Fatalf("unstable id: %q / %q", one.Items[0].ID, two.Items[0].ID)
	}
	if string(one.Items[0].Data) != `{"a":1,"b":2}` {
		t.Fatalf("data = %s", one.Items[0].Data)
	}
}

func TestParseDatasetRejectsDuplicateAndBlankContent(t *testing.T) {
	_, err := ParseDataset(strings.NewReader(`{"datasetName":"x","datasetVersion":"1","items":[{"id":"a","content":"x","data":{}},{"id":"a","content":"y","data":{}}]}`), ValidationOptions{})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	_, err = ParseDataset(strings.NewReader(`{"datasetName":"x","datasetVersion":"1","items":[{"content":" ","data":{}}]}`), ValidationOptions{})
	if err == nil {
		t.Fatal("expected blank content error")
	}
}
