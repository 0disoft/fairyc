package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/packagesource"
	"golang.org/x/sys/windows"
)

func TestRemotePackageCLIInterrupt(t *testing.T) {
	const helperCert = "FAIRYC_FETCH_INTERRUPT_TEST_CERT"
	args := []string{"experimental", "fetch-packages"}
	if certPath := os.Getenv(helperCert); certPath != "" {
		cert, err := x509.ParseCertificate(readFetchInterruptFile(t, certPath))
		if err != nil {
			t.Fatal(err)
		}
		pool := x509.NewCertPool()
		pool.AddCert(cert)
		x509.SetFallbackRoots(pool)
		// This child has its own console. The parent permits Ctrl+C only once
		// the loopback server has seen the download and the lock is held.
		go func() {
			if _, err := io.ReadFull(os.Stdin, make([]byte, 1)); err == nil {
				generate := windows.NewLazySystemDLL("kernel32.dll").NewProc("GenerateConsoleCtrlEvent")
				if ok, _, err := generate.Call(windows.CTRL_C_EVENT, 0); ok == 0 {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
		}()
		os.Exit(Run(args, ".", os.Stdout, os.Stderr))
	}

	root := remoteCLIWorkspace(t)
	if code, out, err := runForTest(append(append([]string{}, args...), "--offline"), root); code != 0 || err != "" {
		t.Fatalf("seed: %d %q %q", code, out, err)
	}
	installed := filepath.Join(root, "vendor", "math")
	before := remoteTree(t, installed)
	manifestPath := filepath.Join(root, "fairy.toml")
	manifest := strings.Replace(string(readFetchInterruptFile(t, manifestPath)), `members=["vendor/math"]`, `members=["vendor/math","vendor/stats"]`, 1)
	writeWorkspaceFile(t, root, "fairy.toml", manifest)
	archive := remoteArchiveBytes(t, zip.Store,
		remoteArchiveEntry{name: "fairy.toml", data: []byte(strings.ReplaceAll(remoteLibraryManifest, "math", "stats"))},
		remoteArchiveEntry{name: "src/stats.fy", data: []byte("module stats; pub fn i32 value() { return 7; }\n")})
	started := make(chan struct{})
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.(http.Flusher).Flush()
			close(started)
			<-r.Context().Done()
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	certPath := filepath.Join(t.TempDir(), "loopback.der")
	if err := os.WriteFile(certPath, server.Certificate().Raw, 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := packagesource.Parse(readFetchInterruptFile(t, filepath.Join(root, packagesource.FileName)))
	if err != nil {
		t.Fatal(err)
	}
	plan.Packages = append(plan.Packages, packagesource.Pin{Path: "vendor/stats", PackageUID: "example.stats",
		Version: "1.0.0", URL: server.URL + "/stats.zip", SHA256: fmt.Sprintf("%x", sha256.Sum256(archive))})
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, packagesource.FileName, string(data))

	image, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	start := func() (*exec.Cmd, *bytes.Buffer, *bytes.Buffer, io.WriteCloser) {
		cmd := exec.CommandContext(ctx, image, "-test.run=^TestRemotePackageCLIInterrupt$")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), helperCert+"="+certPath, "GODEBUG=x509usefallbackroots=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
		stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
		cmd.Stdout, cmd.Stderr = stdout, stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		return cmd, stdout, stderr, stdin
	}
	cmd, stdout, stderr, stdin := start()
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	lock := filepath.Join(root, ".fairyc-fetch.lock")
	select {
	case <-started:
		if _, err := os.Stat(lock); err != nil {
			t.Error(err)
		}
		if _, err := stdin.Write([]byte{1}); err != nil {
			t.Error(err)
		}
	case err := <-finished:
		t.Fatalf("before download: %v stdout=%q stderr=%q", err, stdout, stderr)
	case <-ctx.Done():
		t.Fatalf("waiting for download: %v", ctx.Err())
	}
	_ = stdin.Close()
	if err := <-finished; cmd.ProcessState.ExitCode() != ExitPackageFailure || stdout.Len() != 0 || !strings.Contains(stderr.String(), "context canceled") {
		t.Fatalf("interrupt: %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("lock after interrupt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "vendor", "stats")); !os.IsNotExist(err) {
		t.Fatalf("interrupted package published: %v", err)
	}
	if !reflect.DeepEqual(before, remoteTree(t, installed)) {
		t.Fatal("interrupt changed previously installed package")
	}
	cmd, stdout, stderr, stdin = start()
	_ = stdin.Close()
	if err := cmd.Wait(); err != nil || stderr.Len() != 0 || !strings.Contains(stdout.String(), "1 installed, 1 reused") || requests.Load() != 2 {
		t.Fatalf("retry: %v stdout=%q stderr=%q requests=%d", err, stdout, stderr, requests.Load())
	}
}

func readFetchInterruptFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
