package contractprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReportTopLevelFieldsMatchPublishedSchema(t *testing.T) {
	t.Parallel()
	bytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-contract-profile-v0.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(bytes, &schema); err != nil {
		t.Fatal(err)
	}
	reportBytes, err := NewReport().JSON()
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]json.RawMessage
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatal(err)
	}
	actual := make([]string, 0, len(report))
	for name := range report {
		actual = append(actual, name)
	}
	slices.Sort(actual)
	slices.Sort(schema.Required)
	if !slices.Equal(actual, schema.Required) {
		t.Fatalf("report fields = %v, schema required = %v", actual, schema.Required)
	}
}
