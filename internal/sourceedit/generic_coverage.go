package sourceedit

import (
	"context"
	"fmt"
	"strconv"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type genericTarget struct{ module, name string }

type genericCoverage map[genericplan.DeclarationID]map[string]struct{}

func requireGenericCoverage(ctx context.Context, w workspaceloader.Workspace, edits []preparedBody) (genericCoverage, error) {
	targets := make(map[genericTarget]bool)
	for _, edit := range edits {
		if edit.generic {
			targets[genericTarget{edit.module, edit.function}] = true
		}
	}
	if len(targets) == 0 {
		return nil, nil
	}
	root, err := w.RootPackageID()
	if err != nil {
		return nil, err
	}
	result, err := workspacecompiler.CheckGenericContext(ctx, w)
	if err != nil || !result.Verified {
		return nil, fmt.Errorf("generic edit baseline did not verify")
	}
	required := make(genericCoverage, len(targets))
	for _, declaration := range result.GenericProvenance.Declarations() {
		if declaration.Package == root && declaration.Kind == genericsource.OriginFunction &&
			targets[genericTarget{declaration.Module, declaration.Name}] {
			required[declaration.ID] = make(map[string]struct{})
		}
	}
	if len(required) != len(targets) {
		return nil, fmt.Errorf("generic edit target has no verified declaration")
	}
	for _, specialization := range result.GenericProvenance.Specializations() {
		if bindings, ok := required[specialization.Declaration]; ok {
			bindings[bindingKey(specialization.Bindings)] = struct{}{}
		}
	}
	for _, bindings := range required {
		if len(bindings) == 0 {
			return nil, fmt.Errorf("generic edit target requires a current concrete specialization")
		}
	}
	return required, nil
}

func (required genericCoverage) check(provenance compiler.GenericProvenance) error {
	if len(required) == 0 {
		return nil
	}
	verified := make(genericCoverage, len(required))
	for _, specialization := range provenance.Specializations() {
		if _, ok := required[specialization.Declaration]; !ok {
			continue
		}
		if verified[specialization.Declaration] == nil {
			verified[specialization.Declaration] = make(map[string]struct{})
		}
		verified[specialization.Declaration][bindingKey(specialization.Bindings)] = struct{}{}
	}
	for declaration, bindings := range required {
		for binding := range bindings {
			if _, ok := verified[declaration][binding]; !ok {
				return fmt.Errorf("edited workspace removed a required generic specialization")
			}
		}
	}
	return nil
}

// The generated specialization ID includes the edited body digest and can
// change. The original declaration and ordered concrete bindings identify
// the instance whose verification must survive the edit.
func bindingKey(bindings []genericplan.Binding) string {
	var key []byte
	for _, binding := range bindings {
		key = strconv.AppendQuote(key, binding.Parameter)
		key = strconv.AppendQuote(key, string(binding.Type))
	}
	return string(key)
}
