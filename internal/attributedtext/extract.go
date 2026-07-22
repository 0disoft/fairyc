package attributedtext

import (
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type OmissionCode string

const (
	OmissionOriginNotAdmitted OmissionCode = "origin-not-admitted"
	OmissionKindUnavailable   OmissionCode = "kind-unavailable"
	OmissionInvalidUTF8       OmissionCode = "invalid-utf8"
	OmissionDisallowedControl OmissionCode = "disallowed-control"
	OmissionFragmentByteLimit OmissionCode = "fragment-byte-limit"
)

type Omission struct {
	Code          OmissionCode `json:"code"`
	Kind          Kind         `json:"kind"`
	Origin        Origin       `json:"origin"`
	SymbolVisible string       `json:"symbol_visible"`
	Bytes         uint64       `json:"bytes"`
}

type Usage struct {
	Fragments int    `json:"fragments"`
	TextBytes uint64 `json:"text_bytes"`
}

// Request makes kind, origin, and byte policy explicit. No default selection is inferred.
type Request struct {
	SymbolID string
	Kinds    []Kind
	Origins  []Origin
	Budget   Budget
}

// Subject binds successful text or an omission to the exact current compiler-selected source span.
type Subject struct {
	SymbolVisible     string                    `json:"symbol_visible"`
	SymbolFull        string                    `json:"symbol_full"`
	PackageUID        string                    `json:"package_uid"`
	PackageInstance   string                    `json:"package_instance"`
	PackageCoordinate string                    `json:"package_coordinate"`
	PackageSource     string                    `json:"package_source"`
	Path              string                    `json:"path"`
	FileKind          semanticsnapshot.FileKind `json:"file_kind"`
	FileIdentity      string                    `json:"file_identity"`
	LineMapIdentity   string                    `json:"line_map_identity"`
	StartByte         uint64                    `json:"start_byte"`
	EndByte           uint64                    `json:"end_byte"`
}

type Selection struct {
	Kinds   []Kind   `json:"kinds"`
	Origins []Origin `json:"origins"`
}

type Result struct {
	Subject   Subject
	Selection Selection
	Budget    Budget
	Records   []Record
	Usage     Usage
	Omissions []Omission
}

// ExtractWorkspaceSource emits at most one exact declaration source slice from retained validated
// local workspace bytes. It never reopens source paths and returns text-safety or size exclusions as
// explicit omissions rather than pretending that no source exists.
func ExtractWorkspaceSource(
	workspace workspaceloader.Workspace,
	index *atlasmap.QueryIndex,
	request Request,
) (Result, error) {
	result := Result{
		Selection: Selection{Kinds: slices.Clone(request.Kinds), Origins: slices.Clone(request.Origins)},
		Budget:    request.Budget, Records: make([]Record, 0, 1), Omissions: make([]Omission, 0, 1),
	}
	if err := validateSourceRequest(request); err != nil {
		return result, err
	}
	subject, err := index.ResolveAttributionSubject(request.SymbolID)
	if err != nil {
		return result, err
	}
	span := subject.Subject.Declaration.Span
	if span.PackageInstance != subject.Context.PackageInstance || span.Source == "" ||
		span.StartByte < 0 || span.EndByte <= span.StartByte {
		return result, fmt.Errorf("Atlas attributed source subject has inconsistent compiler span")
	}
	exact, err := workspace.ResolveExactSource(subject.Context.PackageInstance, span.Source)
	if err != nil {
		return result, err
	}
	if err := validateExactSourceContext(subject, exact); err != nil {
		return result, err
	}
	start, end := uint64(span.StartByte), uint64(span.EndByte)
	if end > uint64(len(exact.Bytes)) {
		return result, fmt.Errorf("Atlas attributed source span exceeds retained source bytes")
	}
	result.Subject = Subject{
		SymbolVisible: subject.Subject.Symbol.Visible, SymbolFull: subject.Subject.Symbol.Full,
		PackageUID: exact.Package.PackageUID, PackageInstance: exact.Package.Instance.String(),
		PackageCoordinate: exact.Package.Coordinate.String(), PackageSource: exact.Package.Source.String(),
		Path: exact.File.Path, FileKind: semanticsnapshot.FileKind(exact.File.Kind),
		FileIdentity: exact.File.Digest.String(), LineMapIdentity: exact.File.LineMap.String(),
		StartByte: start, EndByte: end,
	}
	fragmentBytes := exact.Bytes[start:end]
	fragmentLength := uint64(len(fragmentBytes))
	origin, err := sourceOrigin(exact.Package.SourceKind)
	if err != nil {
		return result, err
	}
	omission := Omission{
		Kind: KindSourceExcerpt, Origin: origin,
		SymbolVisible: subject.Subject.Symbol.Visible, Bytes: fragmentLength,
	}
	if origin != OriginWorkspaceAuthored {
		omission.Code = OmissionOriginNotAdmitted
		result.Omissions = append(result.Omissions, omission)
		return result, nil
	}
	if fragmentLength > request.Budget.MaxFragmentBytes || fragmentLength > request.Budget.MaxTextBytes {
		omission.Code = OmissionFragmentByteLimit
		result.Omissions = append(result.Omissions, omission)
		return result, nil
	}
	record, err := New(Input{
		Kind: KindSourceExcerpt, Origin: origin,
		SymbolVisible: subject.Subject.Symbol.Visible, SymbolFull: subject.Subject.Symbol.Full,
		PackageUID: exact.Package.PackageUID, PackageInstance: exact.Package.Instance.String(),
		PackageCoordinate: exact.Package.Coordinate.String(), PackageSource: exact.Package.Source.String(),
		Path: exact.File.Path, FileKind: result.Subject.FileKind,
		FileIdentity: exact.File.Digest.String(), LineMapIdentity: exact.File.LineMap.String(),
		StartByte: start, EndByte: end, Format: FormatFairySourceUTF8, Text: string(fragmentBytes),
	})
	if err != nil {
		var validation *ValidationError
		if !errors.As(err, &validation) {
			return result, err
		}
		switch validation.Reason {
		case ReasonInvalidUTF8:
			omission.Code = OmissionInvalidUTF8
		case ReasonDisallowedControl:
			omission.Code = OmissionDisallowedControl
		default:
			return result, err
		}
		result.Omissions = append(result.Omissions, omission)
		return result, nil
	}
	result.Records = append(result.Records, record)
	result.Usage = Usage{Fragments: 1, TextBytes: fragmentLength}
	return result, nil
}

func validateSourceRequest(request Request) error {
	if request.SymbolID == "" {
		return fmt.Errorf("Atlas attributed source request requires an exact SymbolId")
	}
	if len(request.Kinds) != 1 || request.Kinds[0] != KindSourceExcerpt {
		return fmt.Errorf("Atlas attributed source request must explicitly select only source-excerpt")
	}
	if len(request.Origins) != 1 || request.Origins[0] != OriginWorkspaceAuthored {
		return fmt.Errorf("Atlas attributed source request must explicitly select only workspace-authored origin")
	}
	return ValidateBudget(request.Budget)
}

func sourceOrigin(sourceKind string) (Origin, error) {
	switch sourceKind {
	case workspaceloader.SourceKindLocal:
		return OriginWorkspaceAuthored, nil
	case workspaceloader.SourceKindSDK:
		return OriginCompilerBundled, nil
	default:
		return "", fmt.Errorf("Atlas attributed source package has unsupported source kind %q", sourceKind)
	}
}

func validateExactSourceContext(subject atlasmap.AttributionSubject, exact workspaceloader.ExactSource) error {
	context := subject.Context
	if exact.Package.PackageUID != context.PackageUID || exact.Package.Instance.String() != context.PackageInstance ||
		exact.Package.Coordinate.String() != context.PackageCoordinate || exact.Package.Source.String() != context.PackageSource ||
		exact.File.Path != subject.Subject.Declaration.Span.Source ||
		exact.File.Path != subject.File.Path || exact.File.Digest.String() != subject.File.Digest ||
		exact.File.LineMap.String() != subject.File.LineMap || semanticsnapshot.FileKind(exact.File.Kind) != subject.File.Kind ||
		subject.File.PackageInstance != context.PackageInstance ||
		exact.File.Package != exact.Package.ID {
		return fmt.Errorf("Atlas attributed source does not match the exact query context")
	}
	return nil
}
