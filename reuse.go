package embedstore

import (
	"context"
	"fmt"
)

// BuildFileWithReuse merges input items with a content-bearing existing file.
// Input wins on ID; all merged items are embedded again by BuildFile.
func BuildFileWithReuse(ctx context.Context, input Dataset, reusePath, output string, options BuildOptions) error {
	store, err := LoadFile(reusePath)
	if err != nil {
		return err
	}
	defer store.Close()
	manifest := store.Manifest()
	if manifest.DatasetName != input.DatasetName {
		return fmt.Errorf("reuse dataset %q does not match input dataset %q", manifest.DatasetName, input.DatasetName)
	}
	if !manifest.ContentIncluded {
		return fmt.Errorf("reuse file does not include content")
	}
	memory, ok := store.(*MemoryStore)
	if !ok {
		return fmt.Errorf("reuse store does not expose items")
	}
	items := make([]Item, 0, len(input.Items)+memory.Count())
	seen := make(map[string]struct{}, len(input.Items))
	for _, item := range input.Items {
		items = append(items, item)
		seen[item.ID] = struct{}{}
	}
	for _, item := range memory.Items() {
		if _, exists := seen[item.ID]; !exists {
			items = append(items, item)
		}
	}
	return BuildFile(ctx, Dataset{DatasetName: input.DatasetName, DatasetVersion: input.DatasetVersion, Items: items}, output, options)
}
