package attributedtext

import (
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// ExtractWorkspaceDocumentation emits at most one exact frontend-attached declaration-
// documentation slice from retained validated local-workspace bytes. The attachment comes from the
// immutable semantic-snapshot sidecar; this package neither reparses source nor guesses proximity.
func ExtractWorkspaceDocumentation(
	workspace workspaceloader.Workspace,
	index *atlasmap.QueryIndex,
	snapshot semanticsnapshot.Snapshot,
	request Request,
) (Result, error) {
	result := Result{
		Selection: Selection{Kinds: slices.Clone(request.Kinds), Origins: slices.Clone(request.Origins)},
		Budget:    request.Budget, Records: make([]Record, 0, 1), Omissions: make([]Omission, 0, 1),
	}
	if err := validateDocumentationRequest(request); err != nil {
		return result, err
	}
	subject, err := index.ResolveAttributionSubject(request.SymbolID)
	if err != nil {
		return result, err
	}
	declarationSpan := subject.Subject.Declaration.Span
	if declarationSpan.PackageInstance != subject.Context.PackageInstance || declarationSpan.Source == "" ||
		declarationSpan.StartByte < 0 || declarationSpan.EndByte <= declarationSpan.StartByte {
		return result, fmt.Errorf("Atlas attributed documentation subject has inconsistent compiler span")
	}
	exact, err := workspace.ResolveExactSource(subject.Context.PackageInstance, declarationSpan.Source)
	if err != nil {
		return result, err
	}
	if err := validateExactSourceContext(subject, exact); err != nil {
		return result, err
	}
	origin, err := sourceOrigin(exact.Package.SourceKind)
	if err != nil {
		return result, err
	}
	result.Subject = subjectFromExactSource(subject, exact, declarationSpan)

	documentationSpan, found, err := exactDocumentationSpan(snapshot, subject)
	if err != nil {
		return result, err
	}
	if !found {
		result.Omissions = append(result.Omissions, Omission{
			Code: OmissionKindUnavailable, Kind: KindDeclarationDocumentation, Origin: origin,
			SymbolVisible: subject.Subject.Symbol.Visible, Bytes: 0,
		})
		return result, nil
	}
	if documentationSpan.StartByte < 0 || documentationSpan.EndByte <= documentationSpan.StartByte ||
		documentationSpan.EndByte > len(exact.Bytes) {
		return result, fmt.Errorf("Atlas attributed documentation span exceeds retained source bytes")
	}
	result.Subject = subjectFromExactSource(subject, exact, documentationSpan)
	fragmentBytes := exact.Bytes[documentationSpan.StartByte:documentationSpan.EndByte]
	fragmentLength := uint64(len(fragmentBytes))
	omission := Omission{
		Kind: KindDeclarationDocumentation, Origin: origin,
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
		Kind: KindDeclarationDocumentation, Origin: origin,
		SymbolVisible: subject.Subject.Symbol.Visible, SymbolFull: subject.Subject.Symbol.Full,
		PackageUID: exact.Package.PackageUID, PackageInstance: exact.Package.Instance.String(),
		PackageCoordinate: exact.Package.Coordinate.String(), PackageSource: exact.Package.Source.String(),
		Path: exact.File.Path, FileKind: result.Subject.FileKind,
		FileIdentity: exact.File.Digest.String(), LineMapIdentity: exact.File.LineMap.String(),
		StartByte: uint64(documentationSpan.StartByte), EndByte: uint64(documentationSpan.EndByte),
		Format: FormatFairyDeclarationDocUTF8, Text: string(fragmentBytes),
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

func validateDocumentationRequest(request Request) error {
	if request.SymbolID == "" {
		return fmt.Errorf("Atlas attributed documentation request requires an exact SymbolId")
	}
	if len(request.Kinds) != 1 || request.Kinds[0] != KindDeclarationDocumentation {
		return fmt.Errorf("Atlas attributed documentation request must explicitly select only declaration-documentation")
	}
	if len(request.Origins) != 1 || request.Origins[0] != OriginWorkspaceAuthored {
		return fmt.Errorf("Atlas attributed documentation request must explicitly select only workspace-authored origin")
	}
	return ValidateBudget(request.Budget)
}

func exactDocumentationSpan(
	snapshot semanticsnapshot.Snapshot,
	subject atlasmap.AttributionSubject,
) (semanticsnapshot.Span, bool, error) {
	wanted := subject.Subject.Declaration
	declarationMatches := 0
	for _, candidate := range snapshot.Declarations() {
		if candidate.Key != wanted.Key {
			continue
		}
		declarationMatches++
		if candidate.Span != wanted.Span {
			return semanticsnapshot.Span{}, false, fmt.Errorf("Atlas attributed documentation snapshot declaration does not match the exact query subject")
		}
	}
	if declarationMatches != 1 {
		return semanticsnapshot.Span{}, false, fmt.Errorf("Atlas attributed documentation snapshot does not contain exactly one query declaration")
	}
	var selected semanticsnapshot.DeclarationDocumentation
	found := false
	for _, candidate := range snapshot.DocumentationSpans() {
		if candidate.Declaration != wanted.Key {
			continue
		}
		if found {
			return semanticsnapshot.Span{}, false, fmt.Errorf("Atlas attributed documentation snapshot repeats the query declaration")
		}
		selected, found = candidate, true
	}
	if !found {
		return semanticsnapshot.Span{}, false, nil
	}
	if selected.DeclarationSpan != wanted.Span ||
		selected.Documentation.PackageInstance != wanted.Span.PackageInstance ||
		selected.Documentation.Source != wanted.Span.Source ||
		selected.Documentation.StartByte < 0 ||
		selected.Documentation.EndByte <= selected.Documentation.StartByte ||
		selected.Documentation.EndByte > wanted.Span.StartByte {
		return semanticsnapshot.Span{}, false, fmt.Errorf("Atlas attributed documentation does not match the exact query declaration")
	}
	return selected.Documentation, true, nil
}

func subjectFromExactSource(
	subject atlasmap.AttributionSubject,
	exact workspaceloader.ExactSource,
	span semanticsnapshot.Span,
) Subject {
	return Subject{
		SymbolVisible: subject.Subject.Symbol.Visible, SymbolFull: subject.Subject.Symbol.Full,
		PackageUID: exact.Package.PackageUID, PackageInstance: exact.Package.Instance.String(),
		PackageCoordinate: exact.Package.Coordinate.String(), PackageSource: exact.Package.Source.String(),
		Path: exact.File.Path, FileKind: semanticsnapshot.FileKind(exact.File.Kind),
		FileIdentity: exact.File.Digest.String(), LineMapIdentity: exact.File.LineMap.String(),
		StartByte: uint64(span.StartByte), EndByte: uint64(span.EndByte),
	}
}
