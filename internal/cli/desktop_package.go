package cli

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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/windowsbuild"
)

// These helpers are workspace-owned build tools, like the app's native sources.
// No shell, repository discovery, network installer or GUI launch is introduced.
func desktopDeclared(sourceRoot string) (bool, error) {
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

func desktopHelper(root, script string) (string, error) {
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

type desktopPython struct {
	path   string
	prefix []string
}

func findDesktopPython() (desktopPython, error) {
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
			return desktopPython{binary, prefix}, nil
		}
	}
	return desktopPython{}, errors.New("desktop tooling requires Python 3.11+ on PATH (python, python3 or py -3); install Python and reopen the terminal")
}

type desktopCapture struct {
	data     bytes.Buffer
	exceeded bool
}

func (b *desktopCapture) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 65536 - b.data.Len()
	if n > remaining {
		b.exceeded = true
		p = p[:remaining]
	}
	_, _ = b.data.Write(p)
	return n, nil
}
func runDesktopHelper(python desktopPython, root, script string, arguments ...string) ([]byte, error) {
	helper, err := desktopHelper(root, script)
	if err != nil {
		return nil, fmt.Errorf("desktop helper %s is missing or outside the project", script)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	args := append(append([]string{}, python.prefix...), "-E", "-s", "-B", helper)
	args = append(args, arguments...)
	command := exec.CommandContext(ctx, python.path, args...)
	command.Dir = filepath.Dir(helper)
	var out, diag desktopCapture
	command.Stdout = &out
	command.Stderr = &diag
	if err = command.Run(); err != nil {
		return nil, fmt.Errorf("desktop helper %s failed; run python -B %s manually for details", script, script)
	}
	if out.exceeded || diag.exceeded {
		return nil, errors.New("desktop helper output exceeded 64 KiB")
	}
	return out.data.Bytes(), nil
}

func prepareDesktopBuild(root string) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return nil
	}
	declared, err := desktopDeclared(root)
	if err != nil || !declared {
		return err
	}
	python, err := findDesktopPython()
	if err != nil {
		return err
	}
	_, err = runDesktopHelper(python, root, "prepare_resources.py")
	return err
}

type desktopPackageReport struct {
	Schema    string `json:"schema"`
	Archive   string `json:"archive"`
	SHA256    string `json:"sha256"`
	ExeBytes  int64  `json:"exeBytes"`
	ZipBytes  int64  `json:"zipBytes"`
	Signed    bool   `json:"signed"`
	Published bool   `json:"published"`
}

func parseDesktopPackageReport(data []byte, root string) (desktopPackageReport, error) {
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
		return desktopPackageReport{}, errors.New("invalid desktop packaging receipt")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return desktopPackageReport{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return desktopPackageReport{}, err
	}
	relative, err := filepath.Rel(filepath.Join(root, "dist"), result.Archive)
	digest, hashErr := hex.DecodeString(result.ZipSHA256)
	if err != nil || !filepath.IsLocal(relative) || filepath.Base(relative) != relative || !strings.HasSuffix(relative, ".zip") ||
		!result.OK || result.Signed || result.Published || result.GUILaunched || result.InputInjected || result.ExeBytes <= 0 || result.ZipBytes <= 0 ||
		hashErr != nil || len(digest) != sha256.Size || result.ZipSHA256 != strings.ToLower(result.ZipSHA256) {
		return desktopPackageReport{}, errors.New("invalid desktop packaging receipt")
	}
	directory, err := os.OpenRoot(filepath.Join(root, "dist"))
	if err != nil {
		return desktopPackageReport{}, errors.New("packaged archive directory is missing")
	}
	defer directory.Close()
	file, err := directory.Open(relative)
	if err != nil {
		return desktopPackageReport{}, errors.New("packaged archive is missing or outside dist")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 33<<20 {
		return desktopPackageReport{}, errors.New("packaged archive is missing or exceeds 33 MiB")
	}
	archive, err := io.ReadAll(io.LimitReader(file, (33<<20)+1))
	if err != nil {
		return desktopPackageReport{}, errors.New("packaged archive is missing")
	}
	actual := sha256.Sum256(archive)
	if len(archive) > 33<<20 || int64(len(archive)) != result.ZipBytes || !bytes.Equal(actual[:], digest) {
		return desktopPackageReport{}, errors.New("packaged archive checksum mismatch")
	}
	return desktopPackageReport{Schema: "fairyc.package-report/v1", Archive: "dist/" + relative, SHA256: result.ZipSHA256, ExeBytes: result.ExeBytes, ZipBytes: result.ZipBytes}, nil
}

func runStablePackage(arguments []string, root string, stdout, stderr io.Writer) int {
	seen, err := parseCommandFlags(arguments, flagSchema{Command: "package", Flags: []flagDefinition{{Name: "--locked", Kind: flagSwitch}, {Name: "--json", Kind: flagSwitch}}}, func(string, string) error { return nil })
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return invocationError(stderr, "package currently supports Windows amd64 desktop projects only")
	}
	declared, err := desktopDeclared(root)
	if err != nil {
		return desktopPackageFailure(stderr, "invalid app.toml/fairy.windows.json", ExitPackageFailure)
	}
	if !declared {
		return invocationError(stderr, "package requires a Win32 desktop project with app.toml and fairy.windows.json")
	}
	if _, err = desktopHelper(root, "package_release.py"); err != nil {
		return desktopPackageFailure(stderr, "package_release.py is missing or outside the project", ExitPackageFailure)
	}
	python, err := findDesktopPython()
	if err != nil {
		return desktopPackageFailure(stderr, err.Error(), ExitBuildFailure)
	}
	buildArgs := []string{"--release"}
	if seen["--locked"] {
		buildArgs = append(buildArgs, "--locked")
	}
	if code := runStableBuild(buildArgs, root, io.Discard, stderr); code != ExitSuccess {
		return code
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, "fairy.toml"))
	if err != nil {
		return desktopPackageFailure(stderr, "cannot read package manifest", ExitPackageFailure)
	}
	manifest, err := projectmanifest.Parse(manifestBytes)
	if err != nil {
		return desktopPackageFailure(stderr, "invalid package manifest", ExitPackageFailure)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return internalError(stderr)
	}
	data, err := runDesktopHelper(python, root, "package_release.py", "--exe", filepath.Join(absolute, "build", manifest.Package.Name+".exe"), "--output", filepath.Join(absolute, "dist"))
	if err != nil {
		return desktopPackageFailure(stderr, err.Error(), ExitBuildFailure)
	}
	report, err := parseDesktopPackageReport(data, root)
	if err != nil {
		return desktopPackageFailure(stderr, err.Error(), ExitBuildFailure)
	}
	if seen["--json"] {
		if err = json.NewEncoder(stdout).Encode(report); err != nil {
			return outputError(stderr)
		}
	} else {
		if _, err = fmt.Fprintf(stdout, "packaged: %s (unsigned, release, local only)\n", report.Archive); err != nil {
			return outputError(stderr)
		}
	}
	return ExitSuccess
}
func desktopPackageFailure(stderr io.Writer, message string, code int) int {
	if _, err := fmt.Fprintf(stderr, "fairyc: %s\n", message); err != nil {
		return ExitInternalFailure
	}
	return code
}
