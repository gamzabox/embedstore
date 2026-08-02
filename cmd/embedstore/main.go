package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/embedstore/embedstore"
	openai "github.com/embedstore/embedstore/embedding/openai"
)

func main() {
	if len(os.Args) < 2 {
		die("usage: embedstore <validate|build|search|inspect|verify>")
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "build":
		build(os.Args[2:])
	case "search":
		search(os.Args[2:])
	case "inspect":
		inspect(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		die("unknown command")
	}
}
func validate(args []string) {
	f := flag.NewFlagSet("validate", flag.ExitOnError)
	input := f.String("input", "", "input JSON")
	max := f.Int("max-content-bytes", 0, "maximum content bytes")
	f.Parse(args)
	require(*input, "--input")
	h, err := os.Open(*input)
	if err != nil {
		die(err.Error())
	}
	defer h.Close()
	d, err := embedstore.ParseDataset(h, embedstore.ValidationOptions{MaxContentBytes: *max})
	if err != nil {
		die(err.Error())
	}
	fmt.Printf("Validation successful\nItems: %d\n", len(d.Items))
}
func build(args []string) {
	f := flag.NewFlagSet("build", flag.ExitOnError)
	input := f.String("input", "", "input JSON")
	output := f.String("output", "", "output embed")
	embedding := f.String("embedding", "", "provider/model")
	dimensions := f.Int("dimensions", 0, "dimensions")
	overwrite := f.Bool("overwrite", false, "replace output")
	timeout := f.Duration("timeout", 30*time.Second, "timeout")
	f.Parse(args)
	require(*input, "--input")
	require(*output, "--output")
	require(*embedding, "--embedding")
	if _, err := os.Stat(*output); err == nil && !*overwrite {
		die("output exists; use --overwrite")
	}
	model, err := openai.ParseEmbedding(*embedding)
	if err != nil {
		die(err.Error())
	}
	h, err := os.Open(*input)
	if err != nil {
		die(err.Error())
	}
	defer h.Close()
	d, err := embedstore.ParseDataset(h, embedstore.ValidationOptions{})
	if err != nil {
		die(err.Error())
	}
	contents := make([]string, len(d.Items))
	for i := range d.Items {
		contents[i] = d.Items[i].Content
	}
	client := openai.New(openai.WithModel(model), openai.WithDimensions(*dimensions), openai.WithHTTPClient(&http.Client{Timeout: *timeout}))
	fmt.Fprintln(os.Stderr, "warning: content is sent to the embedding provider")
	vectors, err := client.Embed(context.Background(), contents)
	if err != nil {
		die(err.Error())
	}
	if len(vectors) == 0 {
		die("provider returned no vectors")
	}
	dim := len(vectors[0])
	flat := make([]float32, 0, len(vectors)*dim)
	for _, v := range vectors {
		if len(v) != dim {
			die("provider returned inconsistent dimensions")
		}
		flat = append(flat, v...)
	}
	m := embedstore.Manifest{FormatVersion: 1, DatasetName: d.DatasetName, DatasetVersion: d.DatasetVersion, Embedding: *embedding, Dimensions: dim, VectorType: "float32", Normalized: true, ItemCount: len(d.Items), CreatedAt: time.Now().UTC(), ContentIncluded: true}
	tmp := *output + ".tmp"
	if err := embedstore.WriteFile(tmp, m, d.Items, flat); err != nil {
		die(err.Error())
	}
	if _, err := embedstore.VerifyFile(tmp); err != nil {
		_ = os.Remove(tmp)
		die(err.Error())
	}
	if err := os.Rename(tmp, *output); err != nil {
		die(err.Error())
	}
	fmt.Printf("Built %s (%d items)\n", *output, len(d.Items))
}
func searchOld(args []string) {
	f := flag.NewFlagSet("search", flag.ExitOnError)
	path := f.String("file", "", "embed file")
	query := f.String("query", "", "query")
	limit := f.Int("limit", 5, "limit")
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
	results, err := e.Search(context.Background(), *query, embedstore.SearchOptions{Limit: *limit})
	if err != nil {
		die(err.Error())
	}
	for _, r := range results {
		fmt.Printf("%d. score=%.4f id=%s\n   data: %s\n", r.Rank, r.Score, r.ID, r.Data)
	}
}
func inspectOld(args []string) {
	f := flag.NewFlagSet("inspect", flag.ExitOnError)
	path := f.String("file", "", "embed file")
	jsonOut := f.Bool("json", false, "JSON output")
	f.Parse(args)
	require(*path, "--file")
	m, err := embedstore.InspectFile(*path)
	if err != nil {
		die(err.Error())
	}
	if *jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(m)
		return
	}
	fmt.Printf("Dataset name: %s\nDataset version: %s\nEmbedding: %s\nDimensions: %d\nItems: %d\n", m.DatasetName, m.DatasetVersion, m.Embedding, m.Dimensions, m.ItemCount)
}
func verify(args []string) {
	f := flag.NewFlagSet("verify", flag.ExitOnError)
	path := f.String("file", "", "embed file")
	f.Parse(args)
	require(*path, "--file")
	m, err := embedstore.VerifyFile(*path)
	if err != nil {
		die(err.Error())
	}
	fmt.Printf("Verification successful\nItems: %d\n", m.ItemCount)
}
func require(v, n string) {
	if v == "" {
		die(n + " is required")
	}
}
func die(s string) { fmt.Fprintln(os.Stderr, "error:", s); os.Exit(1) }
