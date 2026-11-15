// Package contractprofile proves that verified Fairy C MIR belongs to the first
// deterministic smart-contract execution subset.
package contractprofile

import (
	"cmp"
	"encoding/json"
	"slices"
)

const (
	// ReportSchema identifies the machine-readable proof report emitted by this package.
	ReportSchema = "fairyc.contract-profile-report/v0"
	// ProfileID identifies the closed policy implemented by Analyze and AnalyzeSources.
	ProfileID = "fairyc.deterministic-contract/v0"
	// CostUnit names the backend-neutral abstract unit used before a concrete chain gas schedule exists.
	CostUnit = "fairyc-abstract-step/v0"
	// LoopBudgetSemantics defines the exact conservative meaning of @max_iter in this profile.
	LoopBudgetSemantics = "maximum aggregate metered cycle-edge traversals per function invocation"
	// MaxLoopBudget keeps static multiplication and eventual runtime metering deliberately bounded.
	MaxLoopBudget uint64 = 1_000_000
)

const (
	reportStatusAccepted    = "accepted"
	reportStatusDiagnostics = "source-diagnostics"
	reportStatusRejected    = "profile-rejected"
)

// Location identifies one source-root-relative byte range without retaining source contents.
type Location struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

// SourceDiagnostic records one parser, semantic, or deliberately unsupported source result.
type SourceDiagnostic struct {
	Code     string   `json:"code"`
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// Violation records one fail-closed contract-profile rejection.
type Violation struct {
	Code     string    `json:"code"`
	Function string    `json:"function,omitempty"`
	Location *Location `json:"location,omitempty"`
	Message  string    `json:"message"`
}

// FunctionReport is the backend-neutral proof summary for one verified MIR function.
type FunctionReport struct {
	Name                         string   `json:"name"`
	HasControlFlowCycle          bool     `json:"hasControlFlowCycle"`
	MeteredCycleEdges            int      `json:"meteredCycleEdges"`
	DeclaredLoopBudget           uint64   `json:"declaredLoopBudget,omitempty"`
	LocalAbstractCost            uint64   `json:"localAbstractCost"`
	TransitiveCostUpperBound     uint64   `json:"transitiveCostUpperBound,omitempty"`
	StorageGrowthUpperBoundBytes uint64   `json:"storageGrowthUpperBoundBytes"`
	Bounded                      bool     `json:"bounded"`
	InternalCalls                []string `json:"internalCalls"`
}

// Report is the complete deterministic profile decision for one source closure or verified program.
type Report struct {
	SchemaVersion       string             `json:"schemaVersion"`
	Profile             string             `json:"profile"`
	Status              string             `json:"status"`
	Accepted            bool               `json:"accepted"`
	CostUnit            string             `json:"costUnit"`
	LoopBudgetSemantics string             `json:"loopBudgetSemantics"`
	PersistentStorage   bool               `json:"persistentStorage"`
	Functions           []FunctionReport   `json:"functions"`
	SourceDiagnostics   []SourceDiagnostic `json:"sourceDiagnostics"`
	Violations          []Violation        `json:"violations"`
}

// NewReport returns one initialized report with stable empty collections.
func NewReport() Report {
	return Report{
		SchemaVersion:       ReportSchema,
		Profile:             ProfileID,
		Status:              reportStatusRejected,
		CostUnit:            CostUnit,
		LoopBudgetSemantics: LoopBudgetSemantics,
		PersistentStorage:   false,
		Functions:           []FunctionReport{},
		SourceDiagnostics:   []SourceDiagnostic{},
		Violations:          []Violation{},
	}
}

// Finalize replaces this report with canonical collections and decision fields.
// Collections shared with other reports or caller-owned slices are not modified.
func (r *Report) Finalize() {
	if r == nil {
		return
	}
	*r = canonicalReport(*r)
}

// canonicalReport never sorts or compacts caller-owned backing arrays.
func canonicalReport(r Report) Report {
	r.Functions = canonicalReportFunctions(r.Functions)
	r.SourceDiagnostics = canonicalReportItems(r.SourceDiagnostics, compareSourceDiagnostic)
	r.Violations = canonicalReportItems(r.Violations, compareViolation)
	r.Accepted, r.Status = reportDecision(len(r.SourceDiagnostics), len(r.Violations))
	return r
}

func canonicalReportFunctions(functions []FunctionReport) []FunctionReport {
	functions = append([]FunctionReport{}, functions...)
	slices.SortFunc(functions, func(left, right FunctionReport) int {
		return cmp.Compare(left.Name, right.Name)
	})
	for index := range functions {
		calls := append([]string{}, functions[index].InternalCalls...)
		slices.Sort(calls)
		functions[index].InternalCalls = slices.Compact(calls)
	}
	return functions
}

func canonicalReportItems[T any](values []T, compare func(T, T) int) []T {
	items := append([]T{}, values...)
	slices.SortFunc(items, compare)
	return slices.CompactFunc(items, func(left, right T) bool {
		return compare(left, right) == 0
	})
}

func reportDecision(diagnostics, violations int) (bool, string) {
	if diagnostics != 0 {
		return false, reportStatusDiagnostics
	}
	if violations != 0 {
		return false, reportStatusRejected
	}
	return true, reportStatusAccepted
}

// JSON returns deterministic indented JSON followed by one newline without
// modifying the report or any collection shared with its caller.
func (r Report) JSON() ([]byte, error) {
	r = canonicalReport(r)
	value, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(value, '\n'), nil
}

func compareSourceDiagnostic(left, right SourceDiagnostic) int {
	if value := compareLocation(left.Location, right.Location); value != 0 {
		return value
	}
	switch {
	case left.Code < right.Code:
		return -1
	case left.Code > right.Code:
		return 1
	case left.Message < right.Message:
		return -1
	case left.Message > right.Message:
		return 1
	default:
		return 0
	}
}

func compareViolation(left, right Violation) int {
	switch {
	case left.Function < right.Function:
		return -1
	case left.Function > right.Function:
		return 1
	}
	if left.Location != nil && right.Location == nil {
		return -1
	}
	if left.Location == nil && right.Location != nil {
		return 1
	}
	if left.Location != nil && right.Location != nil {
		if value := compareLocation(*left.Location, *right.Location); value != 0 {
			return value
		}
	}
	switch {
	case left.Code < right.Code:
		return -1
	case left.Code > right.Code:
		return 1
	case left.Message < right.Message:
		return -1
	case left.Message > right.Message:
		return 1
	default:
		return 0
	}
}

func compareLocation(left, right Location) int {
	switch {
	case left.Source < right.Source:
		return -1
	case left.Source > right.Source:
		return 1
	case left.StartByte < right.StartByte:
		return -1
	case left.StartByte > right.StartByte:
		return 1
	case left.EndByte < right.EndByte:
		return -1
	case left.EndByte > right.EndByte:
		return 1
	default:
		return 0
	}
}
