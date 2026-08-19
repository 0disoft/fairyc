package contractprofile

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	// MaxSourceFiles bounds one explicit profile-analysis closure.
	MaxSourceFiles               = 64
	maxAggregateSourceMultiplier = 4
)

const maxIterAttribute = "max_iter"

// SourceInput is one immutable source-root-relative Fairy C input.
type SourceInput struct {
	Path  string
	Bytes []byte
}

// AnalyzeSources parses one explicit source closure, consumes profile-only attributes, runs the
// ordinary semantic-to-verified-MIR pipeline, and applies the deterministic contract profile.
func AnalyzeSources(inputs []SourceInput) (Report, error) {
	report := NewReport()
	if len(inputs) == 0 {
		return report, errors.New("contract profile requires at least one source file")
	}
	if len(inputs) > MaxSourceFiles {
		return report, fmt.Errorf("contract profile accepts at most %d source files", MaxSourceFiles)
	}

	ordered := make([]SourceInput, len(inputs))
	for index, input := range inputs {
		ordered[index] = SourceInput{
			Path:  strings.ReplaceAll(input.Path, "\\", "/"),
			Bytes: append([]byte(nil), input.Bytes...),
		}
	}
	slices.SortFunc(ordered, func(left, right SourceInput) int {
		return strings.Compare(left.Path, right.Path)
	})

	limits := source.DefaultLimits()
	maxAggregateBytes := limits.MaxBytes * maxAggregateSourceMultiplier
	aggregateBytes := 0
	seenPaths := make(map[string]struct{}, len(ordered))
	semanticFiles := make([]semantic.File, 0, len(ordered))
	loopBudgets := make(map[string]uint64)
	for _, input := range ordered {
		if input.Path == "" {
			return report, errors.New("contract profile source path is empty")
		}
		file := source.NewFile(input.Path, input.Bytes)
		if _, err := file.ExpectedModule(); err != nil {
			return report, fmt.Errorf("contract profile source path %q: %w", input.Path, err)
		}
		if _, duplicate := seenPaths[file.RelativePath]; duplicate {
			return report, fmt.Errorf("contract profile source path %q is listed more than once", file.RelativePath)
		}
		seenPaths[file.RelativePath] = struct{}{}
		aggregateBytes += len(file.Bytes)
		if aggregateBytes > maxAggregateBytes {
			return report, errors.New("contract profile source closure exceeds the aggregate byte limit")
		}
		if len(file.Bytes) > limits.MaxBytes {
			report.SourceDiagnostics = append(report.SourceDiagnostics, SourceDiagnostic{
				Code:     "E0010",
				Location: Location{Source: file.RelativePath},
				Message:  diagnosticMeaning("E0010", "shared parser budget exhausted"),
			})
			continue
		}

		front, err := frontend.AnalyzeSyntax(file, limits)
		if err != nil {
			return report, fmt.Errorf("analyze syntax %q: %w", file.RelativePath, err)
		}
		if len(front.Tree.Diagnostics) != 0 {
			for _, value := range front.Tree.Diagnostics {
				report.SourceDiagnostics = append(report.SourceDiagnostics, SourceDiagnostic{
					Code: value.Code,
					Location: Location{
						Source:    file.RelativePath,
						StartByte: value.Span.StartByte,
						EndByte:   value.Span.EndByte,
					},
					Message: diagnosticMeaning(value.Code, "syntax diagnostic"),
				})
			}
			continue
		}
		if front.AST == nil {
			return report, fmt.Errorf("syntax-clean source %q has no AST", file.RelativePath)
		}

		bounds, violations := consumeProfileAttributes(file.RelativePath, front.AST)
		report.Violations = append(report.Violations, violations...)
		for name, value := range bounds {
			loopBudgets[name] = value
		}
		semanticFiles = append(semanticFiles, semantic.File{Path: file.RelativePath, Module: front.AST})
	}

	if len(report.SourceDiagnostics) != 0 || len(report.Violations) != 0 {
		report.Finalize()
		return report, nil
	}

	checked, pipelineErr := compiler.Check(semanticFiles)
	if pipelineErr != nil {
		return report, fmt.Errorf("verified MIR pipeline: %w", pipelineErr)
	}
	for _, value := range checked.Semantic.Diagnostics {
		report.SourceDiagnostics = append(report.SourceDiagnostics, SourceDiagnostic{
			Code: value.Code,
			Location: Location{
				Source:    value.Primary.Source,
				StartByte: value.Primary.StartByte,
				EndByte:   value.Primary.EndByte,
			},
			Message: diagnosticMeaning(value.Code, "semantic diagnostic"),
		})
	}
	for _, value := range checked.Semantic.Unsupported {
		report.SourceDiagnostics = append(report.SourceDiagnostics, SourceDiagnostic{
			Code: "FC0001",
			Location: Location{
				Source:    value.Span.Source,
				StartByte: value.Span.StartByte,
				EndByte:   value.Span.EndByte,
			},
			Message: "unsupported Fairy C source form: " + value.Feature,
		})
	}
	if len(report.SourceDiagnostics) != 0 {
		report.Finalize()
		return report, nil
	}
	if !checked.Verified {
		return report, errors.New("semantic analysis completed without verified MIR")
	}

	report = Analyze(checked.MIR, loopBudgets)
	report.Finalize()
	return report, nil
}

func consumeProfileAttributes(sourcePath string, module *ast.Module) (map[string]uint64, []Violation) {
	bounds := make(map[string]uint64)
	var violations []Violation

	for index := range module.Functions {
		function := &module.Functions[index]
		filtered := make([]ast.Attribute, 0, len(function.Attributes))
		var declarations []ast.Attribute
		for _, attribute := range function.Attributes {
			if attribute.Name == maxIterAttribute {
				declarations = append(declarations, attribute)
				continue
			}
			filtered = append(filtered, attribute)
		}
		function.Attributes = filtered
		if len(declarations) == 0 {
			continue
		}
		functionName := module.Path + "::" + function.Name
		if function.Owner != "" {
			functionName = module.Path + "::" + function.Owner + "::" + function.Name
		}
		if len(declarations) != 1 {
			location := locationFromDiagnosticSpan(sourcePath, declarations[1].Span)
			violations = append(violations, Violation{
				Code:     "CP0101",
				Function: functionName,
				Location: &location,
				Message:  "@max_iter may appear at most once on a function",
			})
			continue
		}
		declaration := declarations[0]
		if len(declaration.Arguments) != 1 {
			location := locationFromDiagnosticSpan(sourcePath, declaration.Span)
			violations = append(violations, Violation{
				Code:     "CP0102",
				Function: functionName,
				Location: &location,
				Message:  fmt.Sprintf("@max_iter requires one integer in 1..%d", MaxLoopBudget),
			})
			continue
		}
		value, ok := parseLoopBudget(declaration.Arguments[0].Raw)
		if !ok {
			location := locationFromDiagnosticSpan(sourcePath, declaration.Arguments[0].Span)
			violations = append(violations, Violation{
				Code:     "CP0102",
				Function: functionName,
				Location: &location,
				Message:  fmt.Sprintf("@max_iter requires one integer in 1..%d", MaxLoopBudget),
			})
			continue
		}
		bounds[functionName] = value
	}

	for _, value := range module.Constants {
		violations = append(violations, misplacedMaxIter(sourcePath, "constant "+value.Name, value.Attributes)...)
	}
	for _, value := range module.StaticAssertions {
		violations = append(violations, misplacedMaxIter(sourcePath, "static_assert", value.Attributes)...)
	}
	for _, value := range module.Structs {
		violations = append(violations, misplacedMaxIter(sourcePath, "struct "+value.Name, value.Attributes)...)
	}
	for _, value := range module.Enums {
		violations = append(violations, misplacedMaxIter(sourcePath, "enum "+value.Name, value.Attributes)...)
	}
	for _, value := range module.Resources {
		violations = append(violations, misplacedMaxIter(sourcePath, "resource "+value.Name, value.Attributes)...)
		if value.Cleanup != nil {
			violations = append(violations, misplacedMaxIter(sourcePath, "cleanup for "+value.Name, value.Cleanup.Attributes)...)
		}
	}
	for _, value := range module.ForeignFunctions {
		violations = append(violations, misplacedMaxIter(sourcePath, "foreign function "+value.Name, value.Attributes)...)
	}
	return bounds, violations
}

func misplacedMaxIter(sourcePath, owner string, attributes []ast.Attribute) []Violation {
	var result []Violation
	for _, attribute := range attributes {
		if attribute.Name != maxIterAttribute {
			continue
		}
		location := locationFromDiagnosticSpan(sourcePath, attribute.Span)
		result = append(result, Violation{
			Code:     "CP0103",
			Location: &location,
			Message:  "@max_iter is a contract-profile attribute for ordinary Fairy C functions, not " + owner,
		})
	}
	return result
}

func parseLoopBudget(raw string) (uint64, bool) {
	value := strings.ReplaceAll(strings.TrimSpace(raw), "_", "")
	if value == "" {
		return 0, false
	}
	base := 10
	switch {
	case strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X"):
		base = 16
		value = value[2:]
	case strings.HasPrefix(value, "0b") || strings.HasPrefix(value, "0B"):
		base = 2
		value = value[2:]
	}
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, base, 64)
	return parsed, err == nil && parsed > 0 && parsed <= MaxLoopBudget
}

func diagnosticMeaning(code, fallback string) string {
	if entry, found := diagnostic.Lookup(code); found {
		return entry.Meaning
	}
	return fallback
}

func locationFromDiagnosticSpan(sourcePath string, value diagnostic.Span) Location {
	return Location{
		Source:    sourcePath,
		StartByte: value.StartByte,
		EndByte:   value.EndByte,
	}
}

func locationFromHIRSpan(value hir.Span) *Location {
	if value.Source == "" && value.StartByte == 0 && value.EndByte == 0 {
		return nil
	}
	return &Location{
		Source:    value.Source,
		StartByte: value.StartByte,
		EndByte:   value.EndByte,
	}
}
