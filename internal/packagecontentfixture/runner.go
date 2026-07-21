package packagecontentfixture

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/packagecontent"
)

const schemaVersion = "fairyc.package-source-fixtures/v1"

// Run decodes and compares all package-source digest fixtures.
func Run(path string) (Report, error) {
	definition, err := loadManifest(path)
	if err != nil {
		return Report{}, err
	}
	if definition.Schema == "" || definition.SchemaVersion != schemaVersion || len(definition.Contracts) == 0 || len(definition.Cases) == 0 {
		return Report{}, fmt.Errorf("invalid package source fixture contract header")
	}
	report := Report{CasesRun: len(definition.Cases)}
	hashes := make(map[string]packagecontent.Digest, len(definition.Cases))
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			return Report{}, fmt.Errorf("package source fixture has an empty case id")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return Report{}, fmt.Errorf("duplicate package source fixture %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		files := make([]packagecontent.File, 0, len(fixture.Files))
		decodeFailed := false
		for _, sourceFile := range fixture.Files {
			body, err := hex.DecodeString(sourceFile.ContentHex)
			if err != nil {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("decode %q contentHex: %v", sourceFile.Path, err)})
				decodeFailed = true
				break
			}
			files = append(files, packagecontent.File{Path: sourceFile.Path, Bytes: body})
		}
		if decodeFailed {
			continue
		}
		digest, err := packagecontent.Compute(files)
		if err != nil {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("compute digest: %v", err)})
			continue
		}
		hashes[fixture.ID] = digest
	}
	for _, fixture := range definition.Cases {
		actual, found := hashes[fixture.ID]
		if !found {
			continue
		}
		switch fixture.Relation {
		case "baseline":
			if fixture.CompareTo != "" || fixture.ExpectHash == "" {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "baseline requires expectHash and no compareTo"})
			} else if actual.String() != fixture.ExpectHash {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("hash=%q want=%q", actual, fixture.ExpectHash)})
			}
		case "same", "different":
			reference, ok := hashes[fixture.CompareTo]
			if fixture.CompareTo == "" || !ok {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("comparison fixture %q is unavailable", fixture.CompareTo)})
				continue
			}
			equal := actual == reference
			if fixture.Relation == "same" && !equal || fixture.Relation == "different" && equal {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("hash=%s compare=%s relation=%s", actual, reference, fixture.Relation)})
			}
		default:
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown relation %q", fixture.Relation)})
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open package source manifest: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode package source manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode package source manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode package source manifest: %w", err)
	}
	return definition, nil
}
