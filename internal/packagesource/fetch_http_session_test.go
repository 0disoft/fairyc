package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func sessionPackageArchive(t *testing.T, uid string) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, file := range []struct{ name, body string }{{"fairy.toml", strings.Replace(packageManifest, "example.math", uid, 1)}, {"src/math.fy", "module math;pub fn i32 value(){return 42;}\n"}} {
		entry, err := writer.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte(file.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestFetchDefaultHTTPSConnectionReuse(t *testing.T) {
	if workspace := os.Getenv("FAIRYC_TEST_HTTP_SESSION_ROOT"); workspace != "" {
		certificate, err := base64.StdEncoding.DecodeString(os.Getenv("FAIRYC_TEST_HTTP_SESSION_CERT"))
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(certificate)
		if err != nil {
			t.Fatal(err)
		}
		roots := x509.NewCertPool()
		roots.AddCert(cert)
		x509.SetFallbackRoots(roots)
		fetcher := Fetcher{}
		offline := os.Getenv("FAIRYC_TEST_HTTP_SESSION_OFFLINE") == "1"
		result, err := fetcher.Fetch(t.Context(), workspace, offline)
		if err != nil || result.Packages != MaxPackages {
			t.Fatalf("default fetch: %+v %v", result, err)
		}
		if fetcher.Client != nil {
			t.Fatal("caller Fetcher mutated")
		}
		return
	}
	var connections, requests atomic.Int32
	archives := map[string][]byte{}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		data, ok := archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.StartTLS()
	defer server.Close()
	root := t.TempDir()
	plan := Plan{Schema: Schema}
	var members []string
	var dependencies strings.Builder
	for i := 0; i < MaxPackages; i++ {
		name := fmt.Sprintf("math%d", i)
		uid := "example." + name
		data := sessionPackageArchive(t, uid)
		urlPath := "/" + name + ".zip"
		archives[urlPath] = data
		pin := Pin{Path: "vendor/" + name, PackageUID: uid, Version: "1.0.0", URL: server.URL + urlPath, SHA256: hash(data)}
		plan.Packages = append(plan.Packages, pin)
		members = append(members, fmt.Sprintf("%q", pin.Path))
		fmt.Fprintf(&dependencies, "[dependencies.%s]\npath=%q\nversion=\"1.0.0\"\nfeatures=[]\n", name, pin.Path)
	}
	slices.Sort(members)
	manifest := strings.Replace(packageManifest, "example.math", "example.app", 1) + "[workspace]\nmembers=[" + strings.Join(members, ",") + "]\n" + dependencies.String()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"fairy.toml": []byte(manifest), FileName: data, "test-ca.pem": pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(offline bool) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestFetchDefaultHTTPSConnectionReuse$")
		cmd.Env = append(os.Environ(), "FAIRYC_TEST_HTTP_SESSION_ROOT="+root, "FAIRYC_TEST_HTTP_SESSION_CERT="+base64.StdEncoding.EncodeToString(server.Certificate().Raw), "GODEBUG="+os.Getenv("GODEBUG")+",x509usefallbackroots=1")
		if offline {
			cmd.Env = append(cmd.Env, "FAIRYC_TEST_HTTP_SESSION_OFFLINE=1")
		}
		if executable := os.Getenv("FAIRYC_HTTP_SESSION_CLI"); executable != "" {
			args := []string{"experimental", "fetch-packages"}
			if offline {
				args = append(args, "--offline")
			}
			cmd = exec.CommandContext(ctx, executable, args...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "SSL_CERT_FILE="+filepath.Join(root, "test-ca.pem"), "GODEBUG="+os.Getenv("GODEBUG")+",x509sslcertoverrideplatform=1")
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fetch process: %v %s", err, out)
		}
	}
	run(false)
	t.Logf("packages=%d requests=%d TCP/TLS connections=%d", MaxPackages, requests.Load(), connections.Load())
	if requests.Load() != MaxPackages || connections.Load() != 1 {
		t.Fatalf("expected %d GETs over one connection; got GET=%d connections=%d", MaxPackages, requests.Load(), connections.Load())
	}
	run(true)
	if requests.Load() != MaxPackages || connections.Load() != 1 {
		t.Fatal("offline fetch performed network I/O")
	}
}

type callerOwnedSourceTransport struct {
	*http.Transport
	closes atomic.Int32
}

func (transport *callerOwnedSourceTransport) CloseIdleConnections() {
	transport.closes.Add(1)
	transport.Transport.CloseIdleConnections()
}

func TestFetchDoesNotCloseCallerHTTPSPool(t *testing.T) {
	data := archiveBytes(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	client := server.Client()
	transport := &callerOwnedSourceTransport{Transport: client.Transport.(*http.Transport).Clone()}
	client.Transport = transport
	defer client.CloseIdleConnections()
	pin := testPin()
	pin.URL = server.URL + "/math.zip"
	pin.SHA256 = hash(data)
	result, err := (Fetcher{Client: client}).Fetch(t.Context(), fixtureWorkspace(t, pin), false)
	if err != nil || result.Downloaded != 1 {
		t.Fatalf("caller client fetch: %+v %v", result, err)
	}
	if transport.closes.Load() != 0 {
		t.Fatal("fetch closed caller-owned pool")
	}
	client.CloseIdleConnections()
	if transport.closes.Load() != 1 {
		t.Fatal("caller could not close its own pool")
	}
}
