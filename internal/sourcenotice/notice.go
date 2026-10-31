// Package sourcenotice reports source-size candidates without reading file contents.
package sourcenotice

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const DefaultThresholdBytes int64 = 40000
const MaxEntries = 100000

var extensions = []string{".c", ".cc", ".cpp", ".css", ".fy", ".go", ".h", ".hpp", ".js", ".jsx", ".ps1", ".py", ".rs", ".scss", ".sh", ".svelte", ".ts", ".tsx", ".vue"}
var excluded = []string{".cache", ".git", ".venv", "build", "coverage", "dist", "node_modules", "target", "vendor", "venv"}

type Notice struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type Issue struct {
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type Report struct {
	Schema              string   `json:"schema"`
	State               string   `json:"state"`
	ThresholdBytes      int64    `json:"thresholdBytes"`
	MaxEntries          int      `json:"maxEntries"`
	Extensions          []string `json:"extensions"`
	ExcludedDirectories []string `json:"excludedDirectories"`
	VisitedEntries      int      `json:"visitedEntries"`
	SourceFiles         int      `json:"sourceFiles"`
	Files               []Notice `json:"files"`
	SkippedSymlinks     []string `json:"skippedSymlinks"`
	Issues              []Issue  `json:"issues"`
}

// Scan is opt-in metadata-only review. Findings never change compiler limits or inputs.
func Scan(ctx context.Context, root string, threshold int64) Report {
	return scan(ctx, root, threshold, MaxEntries)
}

func scan(ctx context.Context, root string, threshold int64, budget int) Report {
	result := Report{Schema: "fairyc.large-source-notice/v1", State: "unknown", ThresholdBytes: threshold, MaxEntries: budget, Extensions: slices.Clone(extensions), ExcludedDirectories: slices.Clone(excluded), Files: []Notice{}, SkippedSymlinks: []string{}, Issues: []Issue{}}
	if threshold <= 0 || budget <= 0 {
		result.Issues = append(result.Issues, Issue{Reason: "invalid-policy", Detail: "threshold and scan budget must be positive"})
		return result
	}
	if err := ctx.Err(); err != nil {
		result.Issues = append(result.Issues, Issue{Reason: "canceled", Detail: err.Error()})
		return result
	}
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		result.Issues = append(result.Issues, Issue{Reason: "invalid-root", Detail: fmt.Sprint(err)})
		return result
	}
	result.State = "bounded"
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := ctx.Err(); err != nil {
			result.Issues = append(result.Issues, Issue{Path: relative, Reason: "canceled", Detail: err.Error()})
			return filepath.SkipAll
		}
		if result.VisitedEntries >= budget {
			result.Issues = append(result.Issues, Issue{Path: relative, Reason: "scan-limit", Detail: fmt.Sprintf("scan exceeds %d entries", budget)})
			return filepath.SkipAll
		}
		result.VisitedEntries++
		if walkErr != nil {
			result.Issues = append(result.Issues, Issue{Path: relative, Reason: "unreadable-entry", Detail: walkErr.Error()})
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			result.SkippedSymlinks = append(result.SkippedSymlinks, relative)
			return nil
		}
		if entry.IsDir() {
			if path != root && slices.Contains(excluded, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !slices.Contains(extensions, strings.ToLower(filepath.Ext(entry.Name()))) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			result.Issues = append(result.Issues, Issue{Path: relative, Reason: "unreadable-metadata", Detail: err.Error()})
			return nil
		}
		if !info.Mode().IsRegular() {
			result.Issues = append(result.Issues, Issue{Path: relative, Reason: "nonregular-source"})
			return nil
		}
		result.SourceFiles++
		if info.Size() > threshold {
			result.Files = append(result.Files, Notice{Path: relative, Bytes: info.Size()})
		}
		return nil
	})
	if err != nil {
		result.Issues = append(result.Issues, Issue{Reason: "scan-failed", Detail: err.Error()})
	}
	if len(result.Issues) != 0 {
		result.State = "partial"
	}
	slices.SortFunc(result.Files, func(a, b Notice) int { return strings.Compare(a.Path, b.Path) })
	slices.Sort(result.SkippedSymlinks)
	return result
}
