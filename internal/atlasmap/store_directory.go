package atlasmap

import (
	"errors"
	"io"
	"os"
)

const mapDirectoryBatchSize = 128

// Directory entry memory is independent of the number of retained or foreign
// files. OpenRoot also rejects a special file in place of the package directory.
func visitMapDirectory(root *os.Root, name string, visit func(os.DirEntry) error) (err error) {
	boundary, err := root.OpenRoot(name)
	if err != nil {
		return err
	}
	defer boundary.Close()
	directory, err := boundary.Open(".")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	for {
		entries, readErr := directory.ReadDir(mapDirectoryBatchSize)
		for _, entry := range entries {
			if err := visit(entry); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
