package genericsource

import (
	"errors"
	"sort"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
)

const (
	FailureDeclaration = "declaration"
	FailureRequest     = "request"
	FailureUnknown     = "unknown-function"
	FailurePrivate     = "private-function"
	FailureExpansion   = "expansion"
)

// SourceError identifies an original source location without parsing error prose
// or generated symbol names. It is internal provenance, not a public diagnostic
// code assignment. The owning compiler maps Owner and Module to a source file.
type SourceError struct {
	Owner  genericplan.PackageID
	Module string
	Span   diagnostic.Span
	Kind   string
	Cause  error
}

func (e *SourceError) Error() string { return e.Cause.Error() }
func (e *SourceError) Unwrap() error { return e.Cause }

func (x *expander) at(span diagnostic.Span, kind string, err error) error {
	if err == nil {
		return nil
	}
	var located *SourceError
	if errors.As(err, &located) {
		return err
	}
	if !span.Valid(x.module.SourceSize) {
		return err
	}
	return &SourceError{Owner: x.owner, Module: x.module.Path, Span: span, Kind: kind, Cause: err}
}

func locatePlanningFailure(modules []*expander, err error) error {
	var request *genericplan.RequestError
	if !errors.As(err, &request) {
		return err
	}
	key := requestKey(request.Request.Declaration, request.Request.Arguments)
	// Equivalent requests may appear in multiple files. Choose the canonical
	// owner/module then the earliest source span, independent of input ordering.
	ordered := append([]*expander(nil), modules...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].owner != ordered[j].owner {
			return ordered[i].owner < ordered[j].owner
		}
		return ordered[i].module.Path < ordered[j].module.Path
	})
	for _, module := range ordered {
		if span, found := module.requests[key]; found {
			return module.at(span, FailureExpansion, err)
		}
	}
	return err
}
