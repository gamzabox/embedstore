package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/embedstore/embedstore"
)

func validate(args []string) {
	f := flag.NewFlagSet("validate", flag.ExitOnError)
	input := f.String("input", "", "input JSON")
	strict := f.Bool("strict", false, "enable strict validation")
	max := f.Int("max-content-bytes", 0, "maximum content bytes")
	format := f.String("format", "json", "input format")
	output := f.String("output", "text", "text or json")
	f.Parse(args)
	require(*input, "--input")
	if *format != "json" {
		die("--format must be json")
	}
	if *output != "text" && *output != "json" {
		die("--output must be text or json")
	}
	h, err := os.Open(*input)
	if err != nil {
		die(err.Error())
	}
	defer h.Close()
	_, summary, err := embedstore.ValidateDataset(h, embedstore.ValidationOptions{Strict: *strict, MaxContentBytes: *max})
	if err != nil {
		die(err.Error())
	}
	if *output == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(summary); err != nil {
			die(err.Error())
		}
		return
	}
	fmt.Printf("Validation successful\nItems: %d\nDuplicate IDs: %d\nEmpty content: %d\nInvalid records: %d\nEstimated tokens: %d\n", summary.ItemCount, summary.DuplicateIDs, summary.EmptyContents, summary.InvalidRecords, summary.EstimatedTokens)
}
