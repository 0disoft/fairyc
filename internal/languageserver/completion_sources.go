package languageserver

import "github.com/0disoft/fairyc/internal/ast"

type completionLocalSource struct {
	module *ast.Module
	key    sourceKey
}

type completionSourceIndex struct {
	locals  map[string]completionLocalSource
	modules map[sourceKey][]*ast.Module
}

// Each analysis owns an immutable file set. Cache only requested paths and
// exact package/module pairs; an edit creates a new analysis with empty caches.
// Keep the first physical-path match, including a nil AST, just as completion's
// original scan did. A nil AST still permits fallback to LastClean.
func (a *analysis) completionLocal(path string) (*ast.Module, sourceKey) {
	if cached, found := a.completion.locals[path]; found {
		return cached.module, cached.key
	}
	var selected completionLocalSource
	for key, file := range a.files {
		if file.Path == path {
			selected = completionLocalSource{file.Module, key}
			break
		}
	}
	if a.completion.locals == nil {
		a.completion.locals = make(map[string]completionLocalSource)
	}
	a.completion.locals[path] = selected
	return selected.module, selected.key
}

func (a *analysis) importedCompletionModules(key sourceKey) []*ast.Module {
	if modules, found := a.completion.modules[key]; found {
		return modules
	}
	var modules []*ast.Module
	for candidate, file := range a.files {
		if candidate.Package == key.Package && file.Module != nil && file.Module.Path == key.Path {
			modules = append(modules, file.Module)
		}
	}
	if a.completion.modules == nil {
		a.completion.modules = make(map[sourceKey][]*ast.Module)
	}
	a.completion.modules[key] = modules
	return modules
}
