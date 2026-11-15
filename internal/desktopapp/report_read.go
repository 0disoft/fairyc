package desktopapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/durablefile"
)

const maxReportArchiveBytes = 33 << 20

// ReportReadOptions separates directory opening, body I/O and resource closure.
// Nil fields use Background, Root.OpenRoot, io.ReadAll and Closer.Close.
// Cancellation is cooperative; it cannot interrupt a blocked host I/O call.
type ReportReadOptions struct {
	Context       context.Context
	OpenDirectory func(*os.Root, string) (*os.Root, error)
	ReadAll       func(io.Reader) ([]byte, error)
	Close         func(io.Closer) error
}

// ParseReportWithOptions preserves ParseReport's receipt and required inputs.
// dist must be an ordinary directory of the retained project root, with the
// same identity before and after open. No failure returns a success report.
func ParseReportWithOptions(data []byte, root string, options ...ReportReadOptions) (report Report, err error) {
	if len(options) > 1 {
		return Report{}, errors.New("at most one packaging report options value is supported")
	}
	var selected ReportReadOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if selected.OpenDirectory == nil {
		selected.OpenDirectory = (*os.Root).OpenRoot
	}
	if selected.ReadAll == nil {
		selected.ReadAll = io.ReadAll
	}
	if selected.Close == nil {
		selected.Close = func(resource io.Closer) error { return resource.Close() }
	}
	ctx := selected.Context
	var owned []io.Closer
	defer func() {
		for i := len(owned) - 1; i >= 0; i-- {
			err = errors.Join(err, selected.Close(owned[i]))
		}
		err = errors.Join(err, ctx.Err(), context.Cause(ctx))
		if err != nil {
			report = Report{}
		}
	}()
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	var result struct {
		OK            bool   `json:"ok"`
		Archive       string `json:"archive"`
		ZipSHA256     string `json:"zipSha256"`
		ExeBytes      int64  `json:"exeBytes"`
		ZipBytes      int64  `json:"zipBytes"`
		Signed        bool   `json:"signed"`
		Published     bool   `json:"published"`
		GUILaunched   bool   `json:"guiLaunched"`
		InputInjected bool   `json:"inputInjected"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return Report{}, errors.New("invalid desktop packaging receipt")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Report{}, err
	}
	relative, err := filepath.Rel(filepath.Join(root, "dist"), result.Archive)
	digest, hashErr := hex.DecodeString(result.ZipSHA256)
	if err != nil || !filepath.IsLocal(relative) || filepath.Base(relative) != relative || !strings.HasSuffix(relative, ".zip") ||
		!result.OK || result.Signed || result.Published || result.GUILaunched || result.InputInjected || result.ExeBytes <= 0 || result.ZipBytes <= 0 || result.ZipBytes > maxReportArchiveBytes ||
		hashErr != nil || len(digest) != sha256.Size || result.ZipSHA256 != strings.ToLower(result.ZipSHA256) {
		return Report{}, errors.New("invalid desktop packaging receipt")
	}
	project, err := os.OpenRoot(root)
	if err != nil {
		return Report{}, err
	}
	owned = append(owned, project)
	before, err := project.Lstat("dist")
	if err != nil {
		return Report{}, fmt.Errorf("inspect packaged archive directory: %w", err)
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return Report{}, errors.New("packaged archive directory must not be a link or special file")
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	directory, err := selected.OpenDirectory(project, "dist")
	if err != nil {
		return Report{}, fmt.Errorf("open packaged archive directory: %w", err)
	}
	if directory == nil {
		return Report{}, errors.New("packaged archive directory handle is missing")
	}
	owned = append(owned, directory)
	opened, err := directory.Stat(".")
	if err != nil {
		return Report{}, err
	}
	if !opened.IsDir() || !os.SameFile(before, opened) {
		return Report{}, durablefile.ErrIdentityChanged
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	file, err := durablefile.OpenRegular(directory, relative)
	if err != nil {
		return Report{}, fmt.Errorf("packaged archive is missing or outside dist: %w", err)
	}
	owned = append(owned, file)
	info, err := file.Stat()
	if err != nil {
		return Report{}, err
	}
	if info.Size() > maxReportArchiveBytes {
		return Report{}, errors.New("packaged archive is missing or exceeds 33 MiB")
	}
	archive, readErr := selected.ReadAll(io.LimitReader(&reportContextReader{ctx: ctx, reader: file}, maxReportArchiveBytes+1))
	if err := errors.Join(readErr, ctx.Err(), context.Cause(ctx)); err != nil {
		return Report{}, fmt.Errorf("read packaged archive: %w", err)
	}
	actual := sha256.Sum256(archive)
	if len(archive) > maxReportArchiveBytes || int64(len(archive)) != result.ZipBytes || !bytes.Equal(actual[:], digest) {
		return Report{}, errors.New("packaged archive checksum mismatch")
	}
	return Report{Schema: "fairyc.package-report/v1", Archive: "dist/" + relative, SHA256: result.ZipSHA256, ExeBytes: result.ExeBytes, ZipBytes: result.ZipBytes}, nil
}

type reportContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *reportContextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, errors.Join(err, context.Cause(reader.ctx))
	}
	n, err := reader.reader.Read(buffer[:min(len(buffer), 32<<10)])
	if canceled := reader.ctx.Err(); canceled != nil {
		return n, errors.Join(err, canceled, context.Cause(reader.ctx))
	}
	return n, err
}
