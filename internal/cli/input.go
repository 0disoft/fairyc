package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/source"
)

type loadedSource struct {
	file source.File
}

type sourceTooLargeError struct {
	relativePath string
	limit        int
}

func (e sourceTooLargeError) Error() string {
	return fmt.Sprintf("input exceeds the experimental %d-byte limit", e.limit)
}

func loadSource(rootPath, argument string, limit int) (loadedSource, error) {
	if argument == "" || argument == "-" {
		return loadedSource{}, errors.New("input must be one source-root-relative .fy file")
	}
	if filepath.IsAbs(argument) || filepath.VolumeName(argument) != "" {
		return loadedSource{}, errors.New("input path must be relative to the current source root")
	}
	if filepath.Separator != '\\' && strings.Contains(argument, "\\") {
		return loadedSource{}, errors.New("input path uses a non-native path separator")
	}
	clean := filepath.Clean(argument)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return loadedSource{}, errors.New("input path escapes the current source root")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return loadedSource{}, errors.New("open current source root")
	}
	defer root.Close()
	file, err := root.Open(clean)
	if err != nil {
		return loadedSource{}, errors.New("open input inside the current source root")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return loadedSource{}, errors.New("inspect input file")
	}
	if !info.Mode().IsRegular() {
		return loadedSource{}, errors.New("input is not a regular file")
	}
	bytes, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return loadedSource{}, errors.New("read input file")
	}
	relativePath := filepath.ToSlash(clean)
	if len(bytes) > limit {
		return loadedSource{}, sourceTooLargeError{relativePath: relativePath, limit: limit}
	}
	return loadedSource{file: source.NewFile(relativePath, bytes)}, nil
}
