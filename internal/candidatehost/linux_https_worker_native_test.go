package candidatehost

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// Authored actual-libcurl negative TLS replay. No test CA is installed, no
// insecure flag is used, and this does not prove positive trusted downloads.
func TestLinuxHTTPSWorkerRejectsUntrustedNative(t *testing.T) {
	linuxHTTPSUntrustedNative(t, linuxHTTPSProtocol+"\n"+linuxHTTPSWorker, linuxHTTPSUntrustedNativeProbe)
}

func TestLinuxHTTPSParentRejectsUntrustedNative(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_https_open_v1(")
	if err != nil {
		t.Fatal(err)
	}
	linuxHTTPSUntrustedNative(t, source, linuxHTTPSParentUntrustedNativeProbe)
}

func linuxHTTPSUntrustedNative(t *testing.T, source, probe string) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires real Linux amd64, libcurl 8 development headers and system runtime; execution remains batched")
	}
	var requests atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, source+probe, "typedef int fy_https_worker_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "https-worker")
				if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, path, server.URL+"/source.zip")
				cmd.Env = append(os.Environ(), "PATH=")
				if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
					t.Fatalf("untrusted TLS: %v %s", err, out)
				}
			})
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("untrusted endpoint received %d HTTP requests", requests.Load())
	}
}

const linuxHTTPSParentUntrustedNativeProbe = `
#include <assert.h>
int main(int argc,char **argv){
    assert(argc==2);size_t raw=0;uint32_t code=fairyc_private_linux_https_open_v1((const uint8_t *)argv[1],strlen(argv[1])+1,1024,3000,&raw);
    assert(code==0 && raw!=0);uint32_t status=0;uint8_t encoding[128];size_t length=0,declared=0;
    assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&length,&declared)==65536u+(uint32_t)CURLE_PEER_FAILED_VERIFICATION);
    assert(fairyc_private_linux_https_close_v1(raw)==0);int exit_status=0;errno=0;assert(waitpid(-1,&exit_status,WNOHANG)==-1 && errno==ECHILD);return 0;
}
`

const linuxHTTPSUntrustedNativeProbe = `
#include <assert.h>
static uint32_t observed_error;static int frames;
static uint32_t observe(void *context,const void *data,size_t size){
    assert(context==NULL && size==sizeof(fy_linux_https_frame));fy_linux_https_frame frame;memcpy(&frame,data,size);
    assert(frame.kind==3 && frame.length==0 && frame.code==65536u+(uint32_t)CURLE_PEER_FAILED_VERIFICATION);observed_error=frame.code;++frames;return 0;
}
int main(int argc,char **argv){assert(argc==2);uint32_t code=fy_linux_https_transfer(argv[1],1024,1000,observe,NULL);assert(code==observed_error && frames==1);return 0;}
`
