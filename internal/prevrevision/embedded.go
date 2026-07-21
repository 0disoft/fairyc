package prevrevision

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// EmbeddedReport describes one fail-closed migration of complete Fairy C programs embedded in Go
// string literals. Previous-revision tests are deliberately outside this scan.
type EmbeddedReport struct {
	Root            string   `json:"root"`
	Mode            string   `json:"mode"`
	FilesScanned    int      `json:"filesScanned"`
	LiteralsScanned int      `json:"literalsScanned"`
	LiteralsChanged int      `json:"literalsChanged"`
	LiteralsSkipped int      `json:"literalsSkipped"`
	FilesChanged    int      `json:"filesChanged"`
	FilesWritten    int      `json:"filesWritten"`
	Files           []string `json:"files,omitempty"`
	Skipped         []string `json:"skipped,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}

// MigrateEmbeddedGoTree plans every complete previous-revision Fairy C program embedded in Go
// source before optionally replacing any file. It never rewrites fragments, format templates, or
// the frozen previous-revision implementation and tests.
func MigrateEmbeddedGoTree(root string, write bool) (EmbeddedReport, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return EmbeddedReport{}, err
	}
	report := EmbeddedReport{Root: filepath.ToSlash(absolute), Mode: "check"}
	if write {
		report.Mode = "write"
	}
	var plan []plannedFile
	err = filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(absolute, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative != "." && (entry.Name() == ".git" || entry.Name() == ".cache" || relative == "internal/prevrevision") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".go") {
			return nil
		}
		report.FilesScanned++
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		output, scanned, changed, skipped, migrationErr := migrateEmbeddedGoFile(relative, input)
		report.LiteralsScanned += scanned
		report.LiteralsChanged += changed
		report.LiteralsSkipped += len(skipped)
		for _, reason := range skipped {
			report.Skipped = append(report.Skipped, relative+": "+reason)
		}
		if migrationErr != nil {
			report.Errors = append(report.Errors, relative+": "+migrationErr.Error())
			return nil
		}
		if changed == 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		plan = append(plan, plannedFile{path: path, rel: relative, mode: info.Mode(), source: output})
		return nil
	})
	if err != nil {
		return report, err
	}
	slices.Sort(report.Errors)
	slices.Sort(report.Skipped)
	if len(report.Errors) != 0 {
		return report, fmt.Errorf("embedded migration plan rejected %d literals", len(report.Errors))
	}
	slices.SortFunc(plan, func(left, right plannedFile) int { return strings.Compare(left.rel, right.rel) })
	report.FilesChanged = len(plan)
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
		report.FilesWritten++
	}
	return report, nil
}

func migrateEmbeddedGoFile(path string, input []byte) ([]byte, int, int, []string, error) {
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, path, input, parser.SkipObjectResolution)
	if err != nil {
		return nil, 0, 0, nil, fmt.Errorf("parse Go source: %w", err)
	}
	var edits []edit
	var skipped []string
	var fatal error
	scanned := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil || !completePreviousFairySource(value) {
			return true
		}
		scanned++
		migrated, err := Migrate(path, []byte(value))
		if err != nil {
			position := files.Position(literal.Pos())
			skipped = append(skipped, fmt.Sprintf("%d:%d: %v", position.Line, position.Column, err))
			return true
		}
		if !migrated.Changed {
			return true
		}
		replacement, err := quoteGoString(literal.Value, string(migrated.Source))
		if err != nil {
			position := files.Position(literal.Pos())
			fatal = fmt.Errorf("%d:%d: %w", position.Line, position.Column, err)
			return true
		}
		start := files.Position(literal.Pos()).Offset
		end := files.Position(literal.End()).Offset
		edits = append(edits, edit{start: start, end: end, replacement: replacement})
		return true
	})
	if fatal != nil {
		return nil, scanned, len(edits), skipped, fatal
	}
	output, err := applyEdits(input, edits)
	if err != nil {
		return nil, scanned, len(edits), skipped, err
	}
	return output, scanned, len(edits), skipped, nil
}

func completePreviousFairySource(value string) bool {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "module ") || !strings.Contains(trimmed, ";") {
		return false
	}
	markers := []string{" -> ", ": ", "drop fn", "if ", "while ", "match ", "=>", "Result::", "Option::", "&mut ", "*mut ", "fn main("}
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func quoteGoString(original, value string) (string, error) {
	if strings.HasPrefix(original, "`") {
		if strings.Contains(value, "`") {
			return "", fmt.Errorf("migrated raw string contains a backtick")
		}
		return "`" + value + "`", nil
	}
	return strconv.Quote(value), nil
}

// JSON returns one indented report terminated by a newline.
func (r EmbeddedReport) JSON() ([]byte, error) {
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
