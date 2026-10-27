package compiler

import (
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

type genericSourceKey struct {
	owner  genericplan.PackageID
	module string
}

type genericSourceEntry struct {
	file      *semantic.File
	ambiguous bool
}

// Retain ambiguity instead of rejecting here: provenance reports an error only
// when a declaration requests that source, in declaration order.
func indexGenericSources(files []semantic.File, owners map[hir.PackageID]genericplan.PackageID) map[genericSourceKey]genericSourceEntry {
	sources := make(map[genericSourceKey]genericSourceEntry, len(files))
	for i := range files {
		file := &files[i]
		if file.Module == nil {
			continue
		}
		key := genericSourceKey{owners[file.Package], file.Module.Path}
		if entry, found := sources[key]; found {
			entry.ambiguous = true
			sources[key] = entry
		} else {
			sources[key] = genericSourceEntry{file: file}
		}
	}
	return sources
}
