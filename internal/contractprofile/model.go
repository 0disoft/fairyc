// Package contractprofile proves that verified Fairy C MIR belongs to the first
// deterministic smart-contract execution subset.
package contractprofile

import (
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
		Status:              "profile-rejected",
		CostUnit:            CostUnit,
		LoopBudgetSemantics: LoopBudgetSemantics,
		PersistentStorage:   false,
		Functions:           []FunctionReport{},
		SourceDiagnostics:   []SourceDiagnostic{},
		Violations:          []Violation{},
	}
}

// Finalize canonicalizes ordering and derives the public decision fields.
func (r *Report) Finalize() {
	if r == nil {
		return
	}
	slices.SortFunc(r.Functions, func(left, right FunctionReport) int {
		switch {
		case left.Name < right.Name:
			return -1
		case left.Name > right.Name:
			return 1
		default:
			return 0
		}
	})
	for index := range r.Functions {
		slices.Sort(r.Functions[index].InternalCalls)
		r.Functions[index].InternalCalls = slices.Compact(r.Functions[index].InternalCalls)
		if r.Functions[index].InternalCalls == nil {
			r.Functions[index].InternalCalls = []string{}
		}
	}
	slices.SortFunc(r.SourceDiagnostics, compareSourceDiagnostic)
	r.SourceDiagnostics = slices.CompactFunc(r.SourceDiagnostics, func(left, right SourceDiagnostic) bool {
		return compareSourceDiagnostic(left, right) == 0
	})
	slices.SortFunc(r.Violations, compareViolation)
	r.Violations = slices.CompactFunc(r.Violations, func(left, right Violation) bool {
		return compareViolation(left, right) == 0
	})
	if r.Functions == nil {
		r.Functions = []FunctionReport{}
	}
	if r.SourceDiagnostics == nil {
		r.SourceDiagnostics = []SourceDiagnostic{}
	}
	if r.Violations == nil {
		r.Violations = []Violation{}
	}
	r.Accepted = len(r.SourceDiagnostics) == 0 && len(r.Violations) == 0
	switch {
	case r.Accepted:
		r.Status = "accepted"
	case len(r.SourceDiagnostics) != 0:
		r.Status = "source-diagnostics"
	default:
		r.Status = "profile-rejected"
	}
}

// JSON returns deterministic indented JSON followed by one newline.
func (r Report) JSON() ([]byte, error) {
	r.Finalize()
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
