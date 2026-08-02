package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/gamzabox/embedstore"
)

func inspect(args []string) {
	f := flag.NewFlagSet("inspect", flag.ExitOnError)
	path := f.String("file", "", "embed file")
	list := f.Bool("list", false, "list stored IDs")
	id := f.String("id", "", "stored item ID")
	output := f.String("output", "text", "text or json")
	pretty := f.Bool("pretty", false, "pretty JSON")
	f.Parse(args)
	require(*path, "--file")
	result, err := inspectOutput(*path, *list, *id, *output, *pretty)
	if err != nil {
		die(err.Error())
	}
	fmt.Print(string(result))
}

func inspectOutput(path string, list bool, id, output string, pretty bool) ([]byte, error) {
	if list && id != "" {
		return nil, fmt.Errorf("--list and --id cannot be used together")
	}
	if output != "text" && output != "json" {
		return nil, fmt.Errorf("unsupported output %q", output)
	}
	if !list && id == "" {
		m, err := embedstore.InspectFile(path)
		if err != nil {
			return nil, err
		}
		if output == "json" {
			return marshalOutput(m, pretty)
		}
		return []byte(fmt.Sprintf("Dataset name: %s\nDataset version: %s\nEmbedding: %s\nDimensions: %d\nItems: %d\n", m.DatasetName, m.DatasetVersion, m.Embedding, m.Dimensions, m.ItemCount)), nil
	}
	s, err := embedstore.LoadFile(path)
	if err != nil {
		return nil, err
	}
	store, ok := s.(*embedstore.MemoryStore)
	if !ok {
		return nil, fmt.Errorf("item inspection is unavailable")
	}
	items := store.Items()
	if list {
		if output == "json" {
			rows := make([]map[string]any, len(items))
			for i, item := range items {
				rows[i] = map[string]any{"index": i, "id": item.ID}
			}
			return marshalOutput(rows, pretty)
		}
		var b bytes.Buffer
		for i, item := range items {
			fmt.Fprintf(&b, "%d  %s\n", i, item.ID)
		}
		return b.Bytes(), nil
	}
	for i, item := range items {
		if item.ID == id {
			result := struct {
				ID      string          `json:"id"`
				Index   int             `json:"index"`
				Content string          `json:"content,omitempty"`
				Data    json.RawMessage `json:"data"`
			}{item.ID, i, item.Content, item.Data}
			if output == "json" {
				return json.Marshal(result)
			}
			return []byte(fmt.Sprintf("ID:      %s\nIndex:   %d\nContent: %s\nData:    %s\n", item.ID, i, item.Content, item.Data)), nil
		}
	}
	return nil, fmt.Errorf("id %q not found", id)
}
func marshalOutput(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
