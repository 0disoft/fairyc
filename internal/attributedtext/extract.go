package attributedtext

import (
	"errors"
	"fmt"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type OmissionCode string

const (
	OmissionOriginNotAdmitted OmissionCode = "origin-not-admitted"
	OmissionInvalidUTF8       OmissionCode = "invalid-utf8"
	OmissionDisallowedControl OmissionCode = "disallowed-control"
	OmissionFragmentByteLimit OmissionCode = "fragment-byte-limit"
)

type Omission struct {
	Code          OmissionCode
	Kind          Kind
	Origin        Origin
	SymbolVisible string
	Bytes         uint64
}

type Usage struct {
	Fragments int
	TextBytes uint64
}

type Result struct {
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
	symbolID string,
	budget Budget,
) (Result, error) {
	result := Result{Records: make([]Record, 0, 1), Omissions: make([]Omission, 0, 1)}
	if err := ValidateBudget(budget); err != nil {
		return result, err
	}
	subject, err := index.ResolveAttributionSubject(symbolID)
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
	fragmentBytes := exact.Bytes[start:end]
	fragmentLength := uint64(len(fragmentBytes))
	omission := Omission{
		Kind: KindSourceExcerpt, Origin: OriginWorkspaceAuthored,
		SymbolVisible: subject.Subject.Symbol.Visible, Bytes: fragmentLength,
	}
	if exact.Package.SourceKind != workspaceloader.SourceKindLocal {
		omission.Code = OmissionOriginNotAdmitted
		result.Omissions = append(result.Omissions, omission)
		return result, nil
	}
	if fragmentLength > budget.MaxFragmentBytes || fragmentLength > budget.MaxTextBytes {
		omission.Code = OmissionFragmentByteLimit
		result.Omissions = append(result.Omissions, omission)
		return result, nil
	}
	fileKind := semanticsnapshot.FileKind(exact.File.Kind)
	record, err := New(Input{
		Kind: KindSourceExcerpt, Origin: OriginWorkspaceAuthored,
		SymbolVisible: subject.Subject.Symbol.Visible, SymbolFull: subject.Subject.Symbol.Full,
		PackageUID: exact.Package.PackageUID, PackageInstance: exact.Package.Instance.String(),
		PackageCoordinate: exact.Package.Coordinate.String(), PackageSource: exact.Package.Source.String(),
		Path: exact.File.Path, FileKind: fileKind,
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
