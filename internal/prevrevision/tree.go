package prevrevision

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// TreeReport is the stable machine-readable result of one repository migration plan.
type TreeReport struct {
	Root    string   `json:"root"`
	Mode    string   `json:"mode"`
	Scanned int      `json:"scanned"`
	Changed int      `json:"changed"`
	Written int      `json:"written"`
	Skipped int      `json:"skipped"`
	Files   []string `json:"files,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

type plannedFile struct {
	path   string
	rel    string
	mode   fs.FileMode
	source []byte
}

// MigrateTree plans every eligible .fy source before optionally replacing any file.
func MigrateTree(root string, write bool) (TreeReport, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return TreeReport{}, err
	}
	report := TreeReport{Root: filepath.ToSlash(absolute), Mode: "check"}
	if write {
		report.Mode = "write"
	}
	var plan []plannedFile
	err = filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(absolute, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel != "." && (entry.Name() == ".git" || entry.Name() == ".cache") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".fy") {
			return nil
		}
		if strings.Contains(rel, "spec/fixtures/parser/cases/fail/") {
			report.Skipped++
			return nil
		}
		report.Scanned++
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result, err := Migrate(rel, input)
		if err != nil {
			report.Errors = append(report.Errors, rel+": "+err.Error())
			return nil
		}
		if !result.Changed {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		plan = append(plan, plannedFile{path: path, rel: rel, mode: info.Mode(), source: result.Source})
		return nil
	})
	if err != nil {
		return report, err
	}
	slices.Sort(report.Errors)
	if len(report.Errors) != 0 {
		return report, fmt.Errorf("migration plan rejected %d files", len(report.Errors))
	}
	slices.SortFunc(plan, func(left, right plannedFile) int { return strings.Compare(left.rel, right.rel) })
	report.Changed = len(plan)
	for _, file := range plan {
		report.Files = append(report.Files, file.rel)
	}
	if !write {
		return report, nil
	}
	for _, file := range plan {
		if err := atomicReplace(file.path, file.source, file.mode); err != nil {
			return report, fmt.Errorf("write %s: %w", file.rel, err)
		}
		report.Written++
	}
	return report, nil
}

func atomicReplace(path string, content []byte, mode fs.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".fairyc-migrate-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryPath, mode.Perm()); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

// JSON returns one indented report terminated by a newline.
func (r TreeReport) JSON() ([]byte, error) {
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
