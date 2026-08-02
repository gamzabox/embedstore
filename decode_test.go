package embedstore

import "testing"

func TestDecodeDataAcceptsSearchResult(t *testing.T) {
	t.Parallel()
	data, err := DecodeData[struct {
		Topic string `json:"topic"`
	}](SearchResult{Data: []byte(`{"topic":"career"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if data.Topic != "career" {
		t.Fatalf("topic = %q", data.Topic)
	}
}
