package embedstore

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ValidationOptions controls optional input limits.
type ValidationOptions struct {
	Strict          bool
	MaxContentBytes int
	MaxItems        int
}
type Dataset struct {
	DatasetName    string `json:"datasetName"`
	DatasetVersion string `json:"datasetVersion"`
	Items          []Item `json:"items"`
}
type ValidationSummary struct{ ItemCount, DuplicateIDs, EmptyContents, InvalidRecords, EstimatedTokens int }

// ValidateDataset parses input and returns the summary emitted by the CLI.
func ValidateDataset(r io.Reader, options ValidationOptions) (Dataset, ValidationSummary, error) {
	dataset, err := ParseDataset(r, options)
	if err != nil {
		return Dataset{}, ValidationSummary{}, err
	}
	summary := ValidationSummary{ItemCount: len(dataset.Items)}
	for _, item := range dataset.Items {
		summary.EstimatedTokens += estimateTokens(item.Content)
	}
	return dataset, summary, nil
}

// ParseDataset validates the JSON input and makes data and generated IDs deterministic.
func ParseDataset(r io.Reader, options ValidationOptions) (Dataset, error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	var raw struct {
		DatasetName    string `json:"datasetName"`
		DatasetVersion string `json:"datasetVersion"`
		Items          []struct {
			ID      string          `json:"id"`
			Content string          `json:"content"`
			Data    json.RawMessage `json:"data"`
		} `json:"items"`
	}
	if err := decoder.Decode(&raw); err != nil {
		return Dataset{}, fmt.Errorf("invalid input JSON: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return Dataset{}, fmt.Errorf("invalid input JSON: trailing value")
	}
	if strings.TrimSpace(raw.DatasetName) == "" || strings.TrimSpace(raw.DatasetVersion) == "" {
		return Dataset{}, fmt.Errorf("datasetName and datasetVersion are required")
	}
	if options.Strict && (raw.DatasetName != strings.TrimSpace(raw.DatasetName) || raw.DatasetVersion != strings.TrimSpace(raw.DatasetVersion)) {
		return Dataset{}, fmt.Errorf("strict validation: datasetName and datasetVersion must not have surrounding whitespace")
	}
	if raw.Items == nil || len(raw.Items) == 0 {
		return Dataset{}, fmt.Errorf("items must be a non-empty array")
	}
	if options.MaxItems > 0 && len(raw.Items) > options.MaxItems {
		return Dataset{}, fmt.Errorf("item count exceeds maximum")
	}
	dataset := Dataset{DatasetName: raw.DatasetName, DatasetVersion: raw.DatasetVersion, Items: make([]Item, 0, len(raw.Items))}
	seen := map[string]struct{}{}
	for i, source := range raw.Items {
		content := strings.TrimSpace(source.Content)
		if options.Strict && normalizeContent(source.Content) != source.Content {
			return Dataset{}, fmt.Errorf("strict validation: item %d content must not require whitespace normalization", i)
		}
		if content == "" {
			return Dataset{}, fmt.Errorf("item %d: content is required", i)
		}
		if !utf8.ValidString(content) {
			return Dataset{}, fmt.Errorf("item %d: content is not UTF-8", i)
		}
		if options.MaxContentBytes > 0 && len(content) > options.MaxContentBytes {
			return Dataset{}, fmt.Errorf("item %d: content exceeds maximum", i)
		}
		if len(source.Data) == 0 || !json.Valid(source.Data) {
			return Dataset{}, fmt.Errorf("item %d: data is required JSON", i)
		}
		var value any
		d := json.NewDecoder(strings.NewReader(string(source.Data)))
		d.UseNumber()
		if err := d.Decode(&value); err != nil {
			return Dataset{}, fmt.Errorf("item %d: invalid data", i)
		}
		data, err := json.Marshal(value)
		if err != nil {
			return Dataset{}, fmt.Errorf("item %d: unsupported data", i)
		}
		if options.Strict && source.ID != "" && strings.TrimSpace(source.ID) != source.ID {
			return Dataset{}, fmt.Errorf("strict validation: item %d id must not have surrounding whitespace", i)
		}
		id := source.ID
		if id == "" {
			id = uuidV5("6ba7b811-9dad-11d1-80b4-00c04fd430c8", normalizeContent(content)+"\n"+string(data))
		}
		if len(id) > 128 {
			return Dataset{}, fmt.Errorf("item %d: id exceeds 128 characters", i)
		}
		if _, ok := seen[id]; ok {
			return Dataset{}, fmt.Errorf("duplicate id %q", id)
		}
		seen[id] = struct{}{}
		dataset.Items = append(dataset.Items, Item{ID: id, Content: content, Data: data})
	}
	return dataset, nil
}
func normalizeContent(s string) string { return strings.Join(strings.Fields(s), " ") }
func uuidV5(namespace, name string) string {
	raw, _ := hex.DecodeString(strings.ReplaceAll(namespace, "-", ""))
	h := sha1.New()
	h.Write(raw)
	h.Write([]byte(name))
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x50
	sum[8] = (sum[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}
