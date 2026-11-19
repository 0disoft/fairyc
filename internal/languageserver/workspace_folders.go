package languageserver

import (
	"errors"
	"path/filepath"
	"slices"
)

const maxWorkspaceFolders = 64

type workspaceFolder struct {
	URI string `json:"uri"`
}

type initializeParams struct {
	RootURI          string            `json:"rootUri"`
	RootPath         string            `json:"rootPath"`
	WorkspaceFolders []workspaceFolder `json:"workspaceFolders"`
	Capabilities     struct {
		TextDocument struct {
			CodeAction struct {
				LiteralSupport *struct{} `json:"codeActionLiteralSupport"`
			} `json:"codeAction"`
		} `json:"textDocument"`
		Workspace struct {
			WorkspaceEdit struct {
				DocumentChanges bool `json:"documentChanges"`
			} `json:"workspaceEdit"`
		} `json:"workspace"`
	} `json:"capabilities"`
}

func (p initializeParams) roots() ([]string, error) {
	if p.WorkspaceFolders != nil {
		return workspaceFolderPaths(p.WorkspaceFolders)
	}
	if p.RootURI != "" {
		return workspaceFolderPaths([]workspaceFolder{{URI: p.RootURI}})
	}
	if p.RootPath == "" {
		return nil, nil
	}
	if !filepath.IsAbs(p.RootPath) {
		return nil, errors.New("rootPath must be absolute")
	}
	return []string{filepath.Clean(p.RootPath)}, nil
}

type workspaceFolderChange struct {
	Added   []workspaceFolder `json:"added"`
	Removed []workspaceFolder `json:"removed"`
}

func (change workspaceFolderChange) apply(roots []string) ([]string, error) {
	added, err := workspaceFolderPaths(change.Added)
	if err != nil {
		return nil, err
	}
	removed, err := workspaceFolderPaths(change.Removed)
	if err != nil {
		return nil, err
	}
	next := slices.DeleteFunc(slices.Clone(roots), func(root string) bool {
		return hasFolder(removed, root)
	})
	for _, root := range added {
		if !hasFolder(next, root) {
			next = append(next, root)
		}
	}
	if len(next) > maxWorkspaceFolders {
		return nil, errors.New("editor workspace folder limit exceeded")
	}
	return next, nil
}

func workspaceFolderPaths(folders []workspaceFolder) ([]string, error) {
	if len(folders) > maxWorkspaceFolders {
		return nil, errors.New("editor workspace folder limit exceeded")
	}
	paths := make([]string, 0, len(folders))
	for _, folder := range folders {
		path, err := filePath(folder.URI)
		if err != nil {
			return nil, err
		}
		if !hasFolder(paths, path) {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func hasFolder(paths []string, path string) bool {
	return slices.ContainsFunc(paths, func(candidate string) bool {
		return sameDocumentPath(candidate, path)
	})
}

func (s *server) workspaceRoot(filename string) string {
	var nearest string
	for _, root := range s.workspaceRoots {
		if len(root) > len(nearest) && contained(root, filename) {
			nearest = root
		}
	}
	return nearest
}
