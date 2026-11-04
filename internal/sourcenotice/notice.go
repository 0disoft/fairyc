// Package sourcenotice reports source-size candidates without reading file contents.
package sourcenotice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/directoryscan"
	"github.com/0disoft/fairyc/internal/durablefile"
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

func scan(ctx context.Context, root string, threshold int64, budget int) (result Report) {
	result = Report{Schema: "fairyc.large-source-notice/v1", State: "unknown", ThresholdBytes: threshold, MaxEntries: budget, Extensions: slices.Clone(extensions), ExcludedDirectories: slices.Clone(excluded), Files: []Notice{}, SkippedSymlinks: []string{}, Issues: []Issue{}}
	if threshold <= 0 || budget <= 0 {
		result.Issues = append(result.Issues, Issue{Reason: "invalid-policy", Detail: "threshold and scan budget must be positive"})
		return result
	}
	if err := ctx.Err(); err != nil {
		result.Issues = append(result.Issues, Issue{Reason: "canceled", Detail: err.Error()})
		return result
	}
	before, err := os.Lstat(root)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		result.Issues = append(result.Issues, Issue{Reason: "invalid-root", Detail: fmt.Sprint(err)})
		return result
	}
	opened, err := os.OpenRoot(root)
	if err != nil {
		result.Issues = append(result.Issues, Issue{Reason: "invalid-root", Detail: err.Error()})
		return result
	}
	defer func() {
		if closeErr := opened.Close(); closeErr != nil {
			result.Issues = append(result.Issues, Issue{Reason: "scan-failed", Detail: closeErr.Error()})
			result.State = "partial"
		}
	}()
	openedInfo, statErr := opened.Stat(".")
	after, pathErr := os.Lstat(root)
	if statErr != nil || pathErr != nil || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, openedInfo) || !os.SameFile(before, after) {
		result.Issues = append(result.Issues, Issue{Reason: "invalid-root", Detail: "scan root changed while opening"})
		return result
	}
	result.State = "bounded"
	scanBudget := &directoryscan.Budget{Context: ctx, Remaining: budget}
	scanBudget.Remaining--
	result.VisitedEntries++
	noticeScan{root: opened, budget: scanBudget, result: &result}.walk("")
	if len(result.Issues) != 0 {
		result.State = "partial"
	}
	slices.SortFunc(result.Files, func(a, b Notice) int { return strings.Compare(a.Path, b.Path) })
	slices.Sort(result.SkippedSymlinks)
	return result
}

type noticeScan struct {
	root   *os.Root
	budget *directoryscan.Budget
	result *Report
}

func (scan noticeScan) walk(relative string) bool {
	display := relative
	if display == "" {
		display = "."
	}
	if err := scan.budget.Context.Err(); err != nil {
		scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "canceled", Detail: err.Error()})
		return false
	}
	native := filepath.FromSlash(relative)
	if native == "" {
		native = "."
	}
	directory, err := durablefile.OpenDirectory(scan.root, native)
	if err != nil {
		scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "unreadable-entry", Detail: err.Error()})
		return true
	}
	entries, readErr := directoryscan.ReadSorted(directory, scan.budget)
	closeErr := directory.Close()
	scan.result.VisitedEntries += len(entries)
	for _, entry := range entries {
		if err := scan.budget.Context.Err(); err != nil {
			scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "canceled", Detail: err.Error()})
			return false
		}
		child := entry.Name()
		if relative != "" {
			child = path.Join(relative, child)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			scan.result.SkippedSymlinks = append(scan.result.SkippedSymlinks, child)
			continue
		}
		if entry.IsDir() {
			if slices.Contains(excluded, entry.Name()) {
				continue
			}
			if !scan.walk(child) {
				return false
			}
			continue
		}
		if !slices.Contains(extensions, strings.ToLower(filepath.Ext(entry.Name()))) {
			continue
		}
		info, err := scan.root.Lstat(filepath.FromSlash(child))
		if err != nil {
			scan.result.Issues = append(scan.result.Issues, Issue{Path: child, Reason: "unreadable-metadata", Detail: err.Error()})
			continue
		}
		if !info.Mode().IsRegular() {
			scan.result.Issues = append(scan.result.Issues, Issue{Path: child, Reason: "nonregular-source"})
			continue
		}
		scan.result.SourceFiles++
		if info.Size() > scan.result.ThresholdBytes {
			scan.result.Files = append(scan.result.Files, Notice{Path: child, Bytes: info.Size()})
		}
	}
	if closeErr != nil {
		scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "scan-failed", Detail: closeErr.Error()})
		return false
	}
	if readErr == nil {
		return true
	}
	switch {
	case errors.Is(readErr, context.Canceled), errors.Is(readErr, context.DeadlineExceeded):
		scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "canceled", Detail: readErr.Error()})
		return false
	case errors.Is(readErr, directoryscan.ErrLimit):
		scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "scan-limit", Detail: fmt.Sprintf("scan exceeds %d entries", scan.result.MaxEntries)})
		return false
	default:
		scan.result.Issues = append(scan.result.Issues, Issue{Path: display, Reason: "unreadable-entry", Detail: readErr.Error()})
		return true
	}
}
