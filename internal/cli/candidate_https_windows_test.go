package cli

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

// These explicit opt-in tests add one freshly generated, short-lived local TLS
// certificate to CurrentUser ROOT. Cleanup removes exactly that certificate.
// Never weaken WinHTTP certificate validation or use an insecure client flag.
func candidateHTTPSServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	if os.Getenv("FAIRYC_RUN_HTTPS_TRUST_TESTS") != "1" {
		t.Skip("set FAIRYC_RUN_HTTPS_TRUST_TESTS=1 for temporary CurrentUser test certificate")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Fairy C ephemeral loopback test"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "loopback.cer")
	if err = os.WriteFile(file, der, 0600); err != nil {
		t.Fatal(err)
	}
	thumb := sha1.Sum(der)
	if out, err := exec.Command("certutil", "-user", "-addstore", "Root", file).CombinedOutput(); err != nil {
		t.Fatalf("trust local test certificate: %v %s", err, out)
	}
	t.Cleanup(func() {
		if out, err := exec.Command("certutil", "-user", "-delstore", "Root", hex.EncodeToString(thumb[:])).CombinedOutput(); err != nil {
			t.Errorf("remove test certificate: %v %s", err, out)
		}
	})
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func TestCandidateRemoteHTTPS(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_HTTPS_TRUST_TESTS") != "1" {
		t.Skip("opt-in temporary loopback TLS trust")
	}
	var redirects atomic.Int32
	server := candidateHTTPSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/zip" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Accept-Encoding") != "" {
			t.Errorf("ambient request headers: %v", r.Header)
		}
		switch r.URL.Path {
		case "/ok", "/escaped%2Fleaf":
			w.Header().Set("Content-Encoding", "identity")
			_, _ = w.Write([]byte("zip bytes"))
		case "/chunked":
			w.(http.Flusher).Flush()
			_, _ = w.Write([]byte("zip bytes"))
		case "/redirect":
			http.Redirect(w, r, "/target", http.StatusFound)
		case "/target":
			redirects.Add(1)
		case "/auth":
			w.Header().Set("WWW-Authenticate", "Basic realm=fixture")
			w.WriteHeader(401)
		case "/gzip":
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write([]byte("zip bytes"))
		case "/large":
			w.Header().Set("Content-Length", "1000")
			_, _ = w.Write([]byte("zip bytes"))
		case "/overflow":
			w.(http.Flusher).Flush()
			_, _ = w.Write([]byte(strings.Repeat("x", 257)))
		case "/exact":
			_, _ = w.Write([]byte(strings.Repeat("x", 256)))
		case "/truncated":
			w.Header().Set("Content-Length", "12")
			_, _ = w.Write([]byte("zip bytes"))
		case "/dribble":
			for i := 0; i < 30; i++ {
				_, _ = w.Write([]byte("x"))
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
					return
				case <-time.After(40 * time.Millisecond):
				}
			}
		case "/timeout":
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
		default:
			w.WriteHeader(404)
		}
	}))
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_remote_https")
	writeTestSource(t, root, "src/main.fy", `module main;import project_remote_https as https;import std::bytes as bytes;import std::process as process;import std::io as io;
 @effects(network,clock,console) fn i32 main(){var u8[4096] raw=[0;4096];let n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{return 2;}};let ref u8[] address=ref raw;
 let usize timeout=if(process::arg_count()==3){100}else{30000};switch(https::download(address[..n],256,timeout)){case .ok(data):{discard io::write_bytes(bytes::view(ref data));return 0;}case .err(e):{discard io::write_u32(e);return 7;}};return 7;}`)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 128<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "https-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				path, want string
				code       int
			}{{"/ok", "zip bytes", 0}, {"/chunked", "zip bytes", 0}, {"/exact", strings.Repeat("x", 256), 0}, {"/redirect", "4", 7}, {"/auth", "4", 7}, {"/gzip", "4", 7}, {"/large", "5", 7}, {"/overflow", "5", 7}, {"/truncated", "4", 7}, {"/missing", "4", 7}, {"/timeout", "3", 7}, {"/dribble", "3", 7}} {
				args := []string{server.URL + test.path}
				if test.path == "/timeout" || test.path == "/dribble" {
					args = append(args, "short")
				}
				code, out, diagnostic := fetchCLIProcess(t, path, root, args, "")
				if code != test.code || out != test.want || diagnostic != "" {
					t.Fatalf("%s: %d %q %q", test.path, code, out, diagnostic)
				}
			}
			untrusted := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("untrusted endpoint reached") }))
			defer untrusted.Close()
			code, out, _ := fetchCLIProcess(t, path, root, []string{untrusted.URL + "/ok"}, "")
			if code != 7 || out != "3" {
				t.Fatalf("untrusted TLS: %d %q", code, out)
			}
			code, out, _ = fetchCLIProcess(t, path, root, []string{"http://127.0.0.1/ok"}, "")
			if code != 7 || out != "1" {
				t.Fatalf("plaintext: %d %q", code, out)
			}
		})
	}
	if redirects.Load() != 0 {
		t.Fatal("followed redirect")
	}
}

// Compare complete source bytes, including combinations that require both the
// retained-directory and HTTPS assets, through the real authored packager.
func TestCandidateHTTPSRuntimePackaging(t *testing.T) {
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "native_runtime_assets", "project_remote_fs")
	writeTestSource(t, root, "src/main.fy", `module main;import native_runtime_assets as assets;import project_remote_fs as files;import windows_rooted_file as host;import std::bytes as bytes;import std::io as io;
 @effects(filesystem,console) fn i32 main(){let root=switch(host::open(b".")){case .ok(p):{p}case .err(_):{return 1;}};
 let data=switch(files::read(ref root,b"assets.pack",1048576)){case .ok(p):{p}case .err(_):{return 2;}};let bundle=switch(assets::load(data)){case .ok(p):{p}case .err(_):{return 3;}};
 let input=switch(files::read(ref root,b"header",4096)){case .ok(p):{p}case .err(_):{return 4;}};let output=switch(assets::source(ref bundle,bytes::view(ref input),false)){case .ok(p):{p}case .err(_):{return 5;}};discard io::write_bytes(bytes::view(ref output));return 0;}`)
	pack := filepath.Join(root, "assets.pack")
	if out, err := exec.Command("python", "-B", "../../tools/package-candidate-runtime-assets.py", "--output", pack).CombinedOutput(); err != nil {
		t.Fatalf("runtime pack: %v %s", err, out)
	}
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 128<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(100)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := cstdlib.WindowsMSVCCandidateSourceForSymbols(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "packager-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, header := range []string{"fairyc_private_windows_root_file_link_new_v1(", "fairyc_private_windows_ticks_v1(", "fairyc_private_windows_root_file_read_v1(", "fairyc_private_windows_https_read_v1(", "fairyc_private_windows_root_file_read_v1( fairyc_private_windows_https_open_v1(", "fairyc_private_windows_ticks_v1( fairyc_private_windows_https_close_v1("} {
				if err = os.WriteFile(filepath.Join(root, "header"), []byte(header), 0600); err != nil {
					t.Fatal(err)
				}
				host, err := candidatehost.WindowsSource(header)
				if err != nil {
					t.Fatal(err)
				}
				code, out, diagnostic := fetchCLIProcess(t, path, root, []string{pack}, "")
				want := scalar + "\n" + storage + "\n" + host
				if code != 0 || out != want || diagnostic != "" {
					t.Fatalf("%s: %d %s %q", header, code, fetchCLITextDiff(out, want), diagnostic)
				}
			}
		})
	}
}
