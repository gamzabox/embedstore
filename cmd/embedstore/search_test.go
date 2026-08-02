package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/embedstore/embedstore"
)

func TestSearchOutputJSONOptions(t *testing.T) {
	minimum := float32(0.5)
	results := []embedstore.SearchResult{
		{Rank: 1, ID: "alpha", Score: 0.9, Content: "first", Data: json.RawMessage(`{"topic":"a"}`), Vector: []float32{1, 0}},
		{Rank: 2, ID: "beta", Score: 0.7, Content: "second", Data: json.RawMessage(`{"topic":"b"}`), Vector: []float32{0, 1}},
	}
	got, err := searchOutput("query", "openai/model", 5, results, searchOutputOptions{format: "json", pretty: true, includeContent: true, includeVector: true, minScore: &minimum})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Query   string `json:"query"`
		Results []struct {
			Content string    `json:"content"`
			Vector  []float32 `json:"vector"`
		} `json:"results"`
	}
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Query != "query" || len(payload.Results) != 2 || payload.Results[0].Content != "first" || len(payload.Results[0].Vector) != 2 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSearchOutputTextOmitsContentAndVector(t *testing.T) {
	got, err := searchOutput("query", "openai/model", 5, []embedstore.SearchResult{{Rank: 1, ID: "alpha", Score: 0.9, Content: "secret", Data: json.RawMessage(`{}`), Vector: []float32{1}}}, searchOutputOptions{format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "Query: query") || !strings.Contains(text, "score=0.9000") || strings.Contains(text, "secret") || strings.Contains(text, "vector") {
		t.Fatalf("text = %q", text)
	}
	if _, err := searchOutput("q", "e", 5, nil, searchOutputOptions{format: "text", includeVector: true}); err == nil {
		t.Fatal("expected vector/text validation error")
	}
}
