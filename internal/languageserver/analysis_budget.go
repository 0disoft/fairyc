package languageserver

import (
	"fmt"
	"sort"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	defaultAnalysisSourceBudget int64 = 128 << 20
	analysisBudgetDiagnostic          = "analysis-budget-unavailable"
)

type analysisBudgetCandidate struct {
	analysis *analysis
	bytes    int64
}

func (s *server) analysisSourceLimit() int64 {
	if s.analysisBudget > 0 {
		return s.analysisBudget
	}
	return defaultAnalysisSourceBudget
}

func analysisSourceBytes(a *analysis) int64 {
	if a == nil {
		return 0
	}
	if a.sourceCharged {
		return a.sourceCharge
	}
	var total int64
	paths := make(map[string]struct{}, len(a.files))
	for _, file := range a.files {
		total += int64(len(file.Text))
		paths[file.Path] = struct{}{}
	}
	// A failed workspace load can retain the offending closed file only through
	// its diagnostic position index, without ever producing a source catalog.
	for filename, index := range a.positions {
		if _, counted := paths[filename]; !counted && index != nil {
			total += int64(len(index.text))
		}
	}
	return total
}

func workspaceSourceBytes(workspace workspaceloader.Workspace) int64 {
	var total int64
	for _, pkg := range workspace.Packages {
		for _, file := range pkg.Sources {
			total += int64(len(file.Bytes))
		}
	}
	return total
}

func (s *server) retainedAnalyses() map[*analysis]struct{} {
	retained := make(map[*analysis]struct{})
	for _, d := range s.docs {
		if d.View != nil {
			retained[d.View] = struct{}{}
		}
		if d.LastClean != nil {
			retained[d.LastClean] = struct{}{}
		}
	}
	return retained
}

func (s *server) retainedAnalysisSourceBytes() int64 {
	var total int64
	for a := range s.retainedAnalyses() {
		total += analysisSourceBytes(a)
	}
	return total
}

func (s *server) touchAnalysis(a *analysis) {
	if a == nil {
		return
	}
	s.analysisClock++
	a.lastUsed = s.analysisClock
}

func (s *server) reserveAnalysisSource(required int64) (bool, error) {
	limit := s.analysisSourceLimit()
	if required < 0 || required > limit {
		return false, nil
	}
	retained := s.retainedAnalyses()
	var current int64
	for a := range retained {
		current += analysisSourceBytes(a)
	}
	if current <= limit-required {
		return true, nil
	}
	candidates := make([]analysisBudgetCandidate, 0, len(retained))
	for a := range retained {
		if _, protected := s.analysisProtect[a]; protected {
			continue
		}
		candidates = append(candidates, analysisBudgetCandidate{a, analysisSourceBytes(a)})
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].analysis.lastUsed != candidates[right].analysis.lastUsed {
			return candidates[left].analysis.lastUsed < candidates[right].analysis.lastUsed
		}
		return candidates[left].analysis.root < candidates[right].analysis.root
	})
	for _, candidate := range candidates {
		if current <= limit-required {
			break
		}
		if err := s.evictAnalysis(candidate.analysis); err != nil {
			return false, err
		}
		current -= candidate.bytes
	}
	return current <= limit-required, nil
}

func (s *server) evictAnalysis(target *analysis) error {
	if target == nil {
		return nil
	}
	keys := make([]string, 0, len(s.docs))
	for uri := range s.docs {
		keys = append(keys, uri)
	}
	sort.Strings(keys)
	for _, uri := range keys {
		d := s.docs[uri]
		publish := false
		if d.View == target {
			d.View = nil
			d.analysisEvicted = true
			d.completionIndex = nil
			d.positionIndex = nil
			publish = true
		}
		if d.LastClean == target {
			d.LastClean = nil
			d.analysisEvicted = true
		}
		if publish {
			if err := s.publishAnalysisBudget(d, "Analysis was evicted to keep the session source cache within its byte budget."); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *server) publishAnalysisBudget(d *document, message string) error {
	if s.out == nil {
		return nil
	}
	diagnostic := lspDiagnostic{
		Range: textRange(d.Text, 0, 0), Severity: 1, Code: analysisBudgetDiagnostic,
		Source: "fairyc", Message: message,
	}
	return s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri": d.URI, "version": d.Version, "diagnostics": []lspDiagnostic{diagnostic},
	})
}

func (s *server) analysisBudgetUnavailable(d *document, root string, overlays map[string][]byte) *analysis {
	a := &analysis{
		root: root, files: map[sourceKey]sourceView{}, imports: map[sourceKey]hir.PackageID{},
		diagnostics: map[string][]lspDiagnostic{}, budgetExceeded: true,
	}
	message := fmt.Sprintf("Analysis source set exceeds the session retention budget of %d bytes.", s.analysisSourceLimit())
	if len(overlays) == 0 {
		a.add(d.Path, d.Text, 0, 0, analysisBudgetDiagnostic, message)
		return a
	}
	keys := make([]string, 0, len(overlays))
	for filename := range overlays {
		keys = append(keys, filename)
	}
	sort.Strings(keys)
	for _, filename := range keys {
		a.add(filename, overlays[filename], 0, 0, analysisBudgetDiagnostic, message)
	}
	if len(a.diagnostics[d.Path]) == 0 {
		a.add(d.Path, d.Text, 0, 0, analysisBudgetDiagnostic, message)
	}
	return a
}
