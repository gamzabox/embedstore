package embedstore

import (
	"strings"
	"testing"
)

func TestValidateDatasetReturnsSummary(t *testing.T) {
	d, summary, err := ValidateDataset(strings.NewReader(`{"datasetName":"d","datasetVersion":"v","items":[{"content":"hello world","data":{}},{"content":"second","data":{}}]}`), ValidationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 2 || summary.ItemCount != 2 || summary.DuplicateIDs != 0 || summary.EmptyContents != 0 || summary.InvalidRecords != 0 || summary.EstimatedTokens < 2 {
		t.Fatalf("summary %#v", summary)
	}
}
