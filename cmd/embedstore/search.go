package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/gamzabox/embedstore"
	openai "github.com/gamzabox/embedstore/embedding/openai"
)

type searchOutputOptions struct {
	format                                string
	pretty, includeContent, includeVector bool
	minScore                              *float32
}

func search(args []string) {
	f := flag.NewFlagSet("search", flag.ExitOnError)
	path := f.String("file", "", "embed file")
	query := f.String("query", "", "query")
	limit := f.Int("limit", 5, "limit")
	min := f.Float64("min-score", -2, "minimum score")
	format := f.String("output", "text", "text or json")
	pretty := f.Bool("pretty", false, "pretty JSON")
	content := f.Bool("include-content", false, "include content")
	vector := f.Bool("include-vector", false, "include vector")
	f.Parse(args)
	require(*path, "--file")
	require(*query, "--query")
	s, err := embedstore.LoadFile(*path)
	if err != nil {
		die(err.Error())
	}
	m := s.Manifest()
	model, err := openai.ParseEmbedding(m.Embedding)
	if err != nil {
		die(err.Error())
	}
	e, err := embedstore.NewEngine(s, openai.New(openai.WithModel(model), openai.WithDimensions(m.Dimensions)))
	if err != nil {
		die(err.Error())
	}
	var threshold *float32
	if *min >= -1 {
		v := float32(*min)
		threshold = &v
	}
	results, err := e.Search(context.Background(), *query, embedstore.SearchOptions{Limit: *limit, MinScore: threshold})
	if err != nil {
		die(err.Error())
	}
	out, err := searchOutput(*query, m.Embedding, *limit, results, searchOutputOptions{format: *format, pretty: *pretty, includeContent: *content, includeVector: *vector, minScore: threshold})
	if err != nil {
		die(err.Error())
	}
	fmt.Print(string(out))
}
func searchOutput(query, embedding string, limit int, results []embedstore.SearchResult, o searchOutputOptions) ([]byte, error) {
	if o.format == "" {
		o.format = "text"
	}
	if o.includeVector && o.format != "json" {
		return nil, fmt.Errorf("--include-vector requires --output json")
	}
	if o.format != "text" && o.format != "json" {
		return nil, fmt.Errorf("unsupported output %q", o.format)
	}
	if o.format == "text" {
		out := fmt.Sprintf("Query: %s\nEmbedding: %s\nResults: %d\n", query, embedding, len(results))
		for _, r := range results {
			out += fmt.Sprintf("\n%d. score=%.4f id=%s\n", r.Rank, r.Score, r.ID)
			if o.includeContent {
				out += fmt.Sprintf("   content: %s\n", r.Content)
			}
			out += fmt.Sprintf("   data: %s\n", r.Data)
		}
		return []byte(out), nil
	}
	type row struct {
		Rank    int             `json:"rank"`
		ID      string          `json:"id"`
		Score   float32         `json:"score"`
		Content string          `json:"content,omitempty"`
		Data    json.RawMessage `json:"data"`
		Vector  []float32       `json:"vector,omitempty"`
	}
	payload := struct {
		Query     string `json:"query"`
		Embedding string `json:"embedding"`
		Limit     int    `json:"limit"`
		Results   []row  `json:"results"`
	}{query, embedding, limit, make([]row, len(results))}
	for i, r := range results {
		payload.Results[i] = row{Rank: r.Rank, ID: r.ID, Score: r.Score, Data: r.Data}
		if o.includeContent {
			payload.Results[i].Content = r.Content
		}
		if o.includeVector {
			payload.Results[i].Vector = r.Vector
		}
	}
	if o.pretty {
		return json.MarshalIndent(payload, "", "  ")
	}
	return json.Marshal(payload)
}
