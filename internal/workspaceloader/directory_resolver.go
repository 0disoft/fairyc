package workspaceloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type directorySpelling struct {
	name  string
	count int
}

// A resolver belongs to one load. Only directory name listings are shared;
// Lstat still checks every selected segment, including cached parent listings.
type directoryResolver struct {
	listings map[string]map[string]directorySpelling
	readDir  func(string) ([]os.DirEntry, error)
}

func newDirectoryResolver() *directoryResolver {
	return &directoryResolver{
		listings: make(map[string]map[string]directorySpelling),
		readDir:  os.ReadDir,
	}
}

func (r *directoryResolver) listing(directory string) (map[string]directorySpelling, error) {
	if names, found := r.listings[directory]; found {
		return names, nil
	}
	entries, err := r.readDir(directory)
	if err != nil {
		return nil, err
	}
	names := make(map[string]directorySpelling, len(entries))
	for _, entry := range entries {
		key := foldDirectoryName(entry.Name())
		spelling := names[key]
		spelling.name = entry.Name()
		spelling.count++
		names[key] = spelling
	}
	r.listings[directory] = names
	return names, nil
}

// Match strings.EqualFold, including cycles such as K/k/Kelvin sign and
// S/s/long s. ToLower alone does not identify every Unicode simple-fold class.
func foldDirectoryName(name string) string {
	return strings.Map(func(value rune) rune {
		minimum := value
		for folded := unicode.SimpleFold(value); folded != value; folded = unicode.SimpleFold(folded) {
			if folded < minimum {
				minimum = folded
			}
		}
		return minimum
	}, name)
}

func (r *directoryResolver) resolve(base, relative string) (string, error) {
	current := base
	for _, segment := range strings.Split(relative, "/") {
		names, err := r.listing(current)
		if err != nil {
			return "", &LoadError{Reason: ReasonPath, Path: relative, Cause: err, Detail: err.Error()}
		}
		spelling := names[foldDirectoryName(segment)]
		if spelling.count != 1 || spelling.name != segment {
			return "", &LoadError{Reason: ReasonPathCase, Path: relative, Detail: fmt.Sprintf("segment %q does not have one exact on-disk spelling", segment)}
		}
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return "", &LoadError{Reason: ReasonPath, Path: relative, Cause: err, Detail: err.Error()}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", &LoadError{Reason: ReasonSymlink, Path: relative, Detail: "directory path contains a symlink"}
		}
		if !info.IsDir() {
			return "", &LoadError{Reason: ReasonPath, Path: relative, Detail: "path is not a directory"}
		}
	}
	return current, nil
}
