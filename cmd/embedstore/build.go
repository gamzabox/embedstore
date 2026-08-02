package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gamzabox/embedstore"
	openai "github.com/gamzabox/embedstore/embedding/openai"
)

func build(args []string) {
	f := flag.NewFlagSet("build", flag.ExitOnError)
	input := f.String("input", "", "input JSON")
	output := f.String("output", "", "output embed")
	embedding := f.String("embedding", "", "provider/model")
	dimensions := f.Int("dimensions", 0, "dimensions")
	batchSize := f.Int("batch-size", 100, "maximum inputs per request")
	maxBatchTokens := f.Int("max-batch-tokens", 100000, "maximum estimated input tokens per request")
	maxRetries := f.Int("max-retries", 5, "maximum transient request retries")
	reuse := f.String("reuse", "", "existing embed file to merge")
	overwrite := f.Bool("overwrite", false, "replace output")
	includeContent := f.Bool("include-content", true, "store original content")
	timeout := f.Duration("timeout", 30*time.Second, "request timeout")
	f.Parse(args)
	require(*input, "--input")
	require(*output, "--output")
	require(*embedding, "--embedding")
	model, err := openai.ParseEmbedding(*embedding)
	if err != nil {
		die(err.Error())
	}
	h, err := os.Open(*input)
	if err != nil {
		die(err.Error())
	}
	defer h.Close()
	dataset, err := embedstore.ParseDataset(h, embedstore.ValidationOptions{})
	if err != nil {
		die(err.Error())
	}
	client := openai.New(openai.WithModel(model), openai.WithDimensions(*dimensions), openai.WithMaxRetries(*maxRetries), openai.WithHTTPClient(&http.Client{Timeout: *timeout}))
	fmt.Fprintln(os.Stderr, "warning: content is sent to the embedding provider")
	options := embedstore.BuildOptions{Embedder: client, BatchSize: *batchSize, MaxBatchTokens: *maxBatchTokens, Overwrite: *overwrite, IncludeContent: *includeContent}
	if *reuse != "" {
		err = embedstore.BuildFileWithReuse(context.Background(), dataset, *reuse, *output, options)
	} else {
		err = embedstore.BuildFile(context.Background(), dataset, *output, options)
	}
	if err != nil {
		die(err.Error())
	}
	fmt.Printf("Built %s (%d input items)\n", *output, len(dataset.Items))
}
