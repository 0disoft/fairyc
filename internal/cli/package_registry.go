package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/durablefile"
	"github.com/0disoft/fairyc/internal/packagesource"
)

func packageReviewPath(name string) bool {
	return name != "." && fs.ValidPath(name) && !strings.Contains(name, `\`)
}
func readPackageReview(root *os.Root, name string, limit int) ([]byte, error) {
	if !packageReviewPath(name) {
		return nil, fmt.Errorf("package review file needs a portable workspace path")
	}
	file, err := durablefile.OpenRegular(root, filepath.FromSlash(name))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() > int64(limit) {
		return nil, fmt.Errorf("package review file byte bound")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, fmt.Errorf("package review file read or byte bound")
	}
	return data, nil
}

func runPackageRegistry(command string, args []string, sourceRoot string, stdout, stderr io.Writer) int {
	count := 4
	if command == "registry-update" {
		count = 5
	}
	if len(args) != count {
		return invocationError(stderr, command+" requires trust, catalog, signature, requests and (for update) current sources files")
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return packageFailure(stderr, err)
	}
	defer root.Close()
	limits := []int{packagesource.MaxTrustBytes, packagesource.MaxCatalogBytes, packagesource.MaxSignatureBytes, packagesource.MaxPlanBytes, packagesource.MaxPlanBytes}
	data := make([][]byte, count)
	for i, name := range args {
		data[i], err = readPackageReview(root, name, limits[i])
		if err != nil {
			return packageFailure(stderr, err)
		}
	}
	verified, err := packagesource.VerifyRegistryCatalog(data[0], data[1], data[2], time.Now().UTC())
	if err != nil {
		return packageFailure(stderr, err)
	}
	var result any
	if command == "registry-update" {
		current, parseErr := packagesource.Parse(data[4])
		if parseErr != nil {
			return packageFailure(stderr, parseErr)
		}
		result, err = packagesource.ProposeRegistryUpdate(current, verified, data[3])
	} else {
		result, err = packagesource.ResolveRegistry(verified, data[3])
	}
	if err != nil {
		return packageFailure(stderr, err)
	}
	if err = json.NewEncoder(stdout).Encode(result); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func writeGitSnapshot(root *os.Root, name string, data []byte) error {
	if !packageReviewPath(name) {
		return fmt.Errorf("snapshot output needs a portable workspace path")
	}
	file, temporary, err := durablefile.CreateTemporary(root, filepath.Dir(filepath.FromSlash(name)))
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	n, err := file.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return err
	}
	// Hard-link publication is atomic and never replaces an existing destination.
	if err = root.Link(temporary, filepath.FromSlash(name)); err != nil {
		return err
	}
	return durablefile.SyncDirectory(root, filepath.Dir(filepath.FromSlash(name)))
}

func runPackageGitSnapshot(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(args) != 3 || !packageReviewPath(args[2]) {
		return invocationError(stderr, "git-snapshot requires a local repository, full commit and portable output file")
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return packageFailure(stderr, err)
	}
	defer root.Close()
	if _, err = root.Lstat(filepath.FromSlash(args[2])); err == nil {
		return packageFailure(stderr, fmt.Errorf("snapshot output already exists"))
	} else if !errors.Is(err, os.ErrNotExist) {
		return packageFailure(stderr, err)
	}
	repository := args[0]
	if !filepath.IsAbs(repository) {
		repository = filepath.Join(sourceRoot, repository)
	}
	snapshot, err := packagesource.PrepareGitSnapshot(context.Background(), repository, args[1])
	if err != nil {
		return packageFailure(stderr, err)
	}
	if err = writeGitSnapshot(root, args[2], snapshot.Archive); err != nil {
		return packageFailure(stderr, err)
	}
	if err = json.NewEncoder(stdout).Encode(snapshot); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
