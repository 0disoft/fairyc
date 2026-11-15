// Package desktopapp prepares project-owned Windows resources and verifies
// local package receipts. It does not compile Fairy sources or start a GUI.
package desktopapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/windowsbuild"
)

// These helpers are workspace-owned build tools, like the app's native sources.
// No shell, repository discovery, network installer or GUI launch is introduced.
func Declared(sourceRoot string) (bool, error) {
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return false, nil
	}
	defer root.Close()
	if _, err = root.Stat("app.toml"); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	file, err := root.Open(windowsbuild.FileName)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, windowsbuild.MaxConfigBytes+1))
	if err != nil {
		return false, err
	}
	config, err := windowsbuild.Parse(data)
	return config.Subsystem == "windows", err
}

func Helper(root, script string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.EvalSymlinks(filepath.Join(absolute, script))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(absolute, candidate)
	if err != nil || !filepath.IsLocal(relative) {
		return "", errors.New("desktop helper must be inside the project")
	}
	info, err := os.Stat(candidate)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("desktop helper must be a regular file")
	}
	return candidate, nil
}

type Python struct {
	path   string
	prefix []string
}

func FindPython() (Python, error) {
	for _, name := range []string{"python", "python3", "py"} {
		binary, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		prefix := []string{}
		if name == "py" {
			prefix = append(prefix, "-3")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		args := append(append([]string{}, prefix...), "-E", "-s", "-B", "-c", "import sys;sys.exit(0 if sys.version_info >= (3,11) else 1)")
		err = exec.CommandContext(ctx, binary, args...).Run()
		cancel()
		if err == nil {
			return Python{binary, prefix}, nil
		}
	}
	return Python{}, errors.New("desktop tooling requires Python 3.11+ on PATH (python, python3 or py -3); install Python and reopen the terminal")
}

type capture struct {
	data     bytes.Buffer
	exceeded bool
}

func (b *capture) Write(p []byte) (int, error) {
	n := len(p)
	remaining := helperOutputBytes - b.data.Len()
	if n > remaining {
		b.exceeded = true
		p = p[:remaining]
	}
	_, _ = b.data.Write(p)
	return n, nil
}
func RunHelper(python Python, root, script string, arguments ...string) ([]byte, error) {
	return RunHelperWithOptions(python, root, script, arguments)
}

func PrepareBuild(root string) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return nil
	}
	declared, err := Declared(root)
	if err != nil || !declared {
		return err
	}
	python, err := FindPython()
	if err != nil {
		return err
	}
	_, err = RunHelper(python, root, "prepare_resources.py")
	return err
}

type Report struct {
	Schema    string `json:"schema"`
	Archive   string `json:"archive"`
	SHA256    string `json:"sha256"`
	ExeBytes  int64  `json:"exeBytes"`
	ZipBytes  int64  `json:"zipBytes"`
	Signed    bool   `json:"signed"`
	Published bool   `json:"published"`
}

func ParseReport(data []byte, root string) (Report, error) {
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
	root, err := filepath.Abs(root)
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
		!result.OK || result.Signed || result.Published || result.GUILaunched || result.InputInjected || result.ExeBytes <= 0 || result.ZipBytes <= 0 ||
		hashErr != nil || len(digest) != sha256.Size || result.ZipSHA256 != strings.ToLower(result.ZipSHA256) {
		return Report{}, errors.New("invalid desktop packaging receipt")
	}
	directory, err := os.OpenRoot(filepath.Join(root, "dist"))
	if err != nil {
		return Report{}, errors.New("packaged archive directory is missing")
	}
	defer directory.Close()
	file, err := directory.Open(relative)
	if err != nil {
		return Report{}, errors.New("packaged archive is missing or outside dist")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 33<<20 {
		return Report{}, errors.New("packaged archive is missing or exceeds 33 MiB")
	}
	archive, err := io.ReadAll(io.LimitReader(file, (33<<20)+1))
	if err != nil {
		return Report{}, errors.New("packaged archive is missing")
	}
	actual := sha256.Sum256(archive)
	if len(archive) > 33<<20 || int64(len(archive)) != result.ZipBytes || !bytes.Equal(actual[:], digest) {
		return Report{}, errors.New("packaged archive checksum mismatch")
	}
	return Report{Schema: "fairyc.package-report/v1", Archive: "dist/" + relative, SHA256: result.ZipSHA256, ExeBytes: result.ExeBytes, ZipBytes: result.ZipBytes}, nil
}
