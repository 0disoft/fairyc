package candidatehost

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxHTTPSParentDNSDeadlineNative(t *testing.T) {
	linuxHTTPSDeadlineNative(t, "https://packages.fairyc-parent-deadline.invalid/a.zip", "dns")
}

// TCP accepts connections but never starts TLS; no certificate or trust-store
// change can make this a false positive for a successful trusted download.
func TestLinuxHTTPSParentTLSDeadlineNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires real Linux amd64 and system libcurl; execution remains batched")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	connections := make(chan net.Conn, 8)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(connections)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			select {
			case connections <- conn:
			default:
				_ = conn.Close()
			}
		}
	}()
	defer func() {
		_ = listener.Close()
		<-done
		for conn := range connections {
			_ = conn.Close()
		}
	}()
	linuxHTTPSDeadlineNative(t, "https://"+listener.Addr().String()+"/a.zip", "tls")
}

func linuxHTTPSDeadlineNative(t *testing.T, address, kind string) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires real Linux amd64 and system libcurl; execution remains batched")
	}
	source, err := LinuxSource("fairyc_private_linux_https_open_v1(")
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, source+linuxHTTPSDeadlineCases, "typedef int fy_https_deadline_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "https-deadline")
				if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, path, address, kind)
				cmd.Env = append(os.Environ(), "PATH=")
				if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
					t.Fatalf("deadline: %v %s", err, out)
				}
			})
		}
	}
}

const linuxHTTPSDeadlineCases = `
#include <assert.h>
int main(int argc,char **argv){
    assert(argc==3);size_t started=0,finished=0;assert(fairyc_private_linux_monotonic_ms_v1(&started)==0);
    size_t raw=0;uint32_t code=fairyc_private_linux_https_open_v1((const uint8_t *)argv[1],strlen(argv[1])+1,1024,100,&raw);
    if(code==0){uint32_t status=0;uint8_t encoding[128];size_t length=0,declared=0;code=fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&length,&declared);}
    if(raw!=0){assert(fairyc_private_linux_https_close_v1(raw)==0);}
    assert(code==ETIMEDOUT || code==65536u+(uint32_t)CURLE_OPERATION_TIMEDOUT ||
        (strcmp(argv[2],"dns")==0 && code==65536u+(uint32_t)CURLE_COULDNT_RESOLVE_HOST));
    assert(fairyc_private_linux_monotonic_ms_v1(&finished)==0 && finished-started<1500);
    int status=0;errno=0;assert(waitpid(-1,&status,WNOHANG)==-1 && errno==ECHILD);return 0;
}
`

// Real fork/pipe/poll/clock/signal/reaping and descriptor boundary. Only the
// network producer is replaced by a controlled in-child byte producer here.
func TestLinuxHTTPSParentProcessNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires real Linux amd64; execution remains batched")
	}
	source, err := LinuxSource("fairyc_private_linux_https_open_v1(")
	if err != nil {
		t.Fatal(err)
	}
	source = strings.Replace(source, linuxHTTPSWorker, linuxHTTPSProcessProducer, 1)
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, source+linuxHTTPSProcessCases, "typedef int fy_https_process_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				runLinuxHTTPSProbe(t, artifact.Bytes, "https-process")
			})
		}
	}
}

// Keep this replay's inputs checked on Windows without describing that as an
// executed Linux process test or trusting an ignored build artifact.
func TestLinuxHTTPSParentNativeInputs(t *testing.T) {
	source, err := LinuxSource("fairyc_private_linux_https_open_v1(")
	if err != nil || !strings.Contains(source, linuxHTTPSParent) || !strings.Contains(source, linuxHTTPSWorker) {
		t.Fatalf("native source selection: %v", err)
	}
	for _, required := range []string{"fork()", "pipe2(", "poll(", "PR_SET_PDEATHSIG", "close_range(", "SIGKILL", "waitpid(", "WEXITED | WNOWAIT | WNOHANG"} {
		if !strings.Contains(source, required) {
			t.Fatalf("native boundary missing %s", required)
		}
	}
	for _, required := range []string{"slow-header", "slow-body", "end-error", "bad-frame", "no-end", "check_parent_death", "F_DUPFD_CLOEXEC", "ECHILD"} {
		if !strings.Contains(linuxHTTPSProcessCases+linuxHTTPSProcessProducer, required) {
			t.Fatalf("native fixture missing %s", required)
		}
	}
}

const linuxHTTPSProcessProducer = `
#include <assert.h>
#include <fcntl.h>
#include <signal.h>
#include <unistd.h>
static void pause_producer(void){struct timespec delay={2,0};while(nanosleep(&delay,&delay)!=0 && errno==EINTR){}}
static uint32_t fy_linux_https_transfer(const char *address,size_t maximum,size_t timeout,fy_linux_https_writer writer,void *sink){
    assert(maximum==100 && timeout>0 && timeout<=30000);
    for(int fd=0;fd<=257;++fd){if(fd==3){assert(fcntl(fd,F_GETFD)>=0);}else{errno=0;assert(fcntl(fd,F_GETFD)<0 && errno==EBADF);}}
    fy_linux_https_response response;memset(&response,0,sizeof response);response.write=writer;response.sink=sink;response.maximum=maximum;
    if(strstr(address,"slow-header")!=NULL){pause_producer();}
    if(strstr(address,"end-error")!=NULL){return fy_linux_https_send(&response,3,65596,NULL,0);}
    if(strstr(address,"bad-frame")!=NULL){fy_linux_https_frame frame={2,1,0};return writer(sink,&frame,sizeof frame);}
    uint32_t code=fy_linux_https_line(&response,"HTTP/2 200\r\n",12);if(code!=0){return code;}
    code=fy_linux_https_header("\r\n",1,2,&response)==2?0:EIO;if(code!=0){return code;}
    if(strstr(address,"slow-body")!=NULL || strstr(address,"parent-death")!=NULL){pause_producer();}
    if(strstr(address,"no-end")!=NULL){return 0;}
    char data[5]={'a',0,(char)-1,'b','c'};if(fy_linux_https_body(data,1,sizeof data,&response)!=sizeof data){return EIO;}
    return fy_linux_https_send(&response,3,0,NULL,0);
}
`

const linuxHTTPSProcessCases = `
#include <assert.h>
static size_t open_case(const char *name,size_t timeout){
    size_t raw=0;uint32_t code=fairyc_private_linux_https_open_v1((const uint8_t *)name,strlen(name)+1,100,timeout,&raw);
    if(code!=0){if(raw!=0){(void)fairyc_private_linux_https_close_v1(raw);}assert(code==0);}assert(raw!=0);return raw;
}
static void no_children(void){int status=0;errno=0;assert(waitpid(-1,&status,WNOHANG)==-1 && errno==ECHILD);}
static void check_parent_death(void){
    assert(prctl(PR_SET_CHILD_SUBREAPER,1)==0);int channel[2];assert(pipe2(channel,O_CLOEXEC)==0);
    pid_t launcher=fork();assert(launcher>=0);
    if(launcher==0){
        assert(close(channel[0])==0);size_t raw=open_case("https://packages.invalid/parent-death",30000);
        uint32_t status=0;uint8_t encoding[128];size_t length=0,declared=0;
        assert(fairyc_private_linux_https_headers_v1(raw,&status,encoding,128,&length,&declared)==0 && status==200);
        pid_t child=((fy_linux_https_request *)(uintptr_t)raw)->child;
        assert(write(channel[1],&child,sizeof child)==(ssize_t)sizeof child);_exit(0);
    }
    assert(close(channel[1])==0);pid_t child=0;assert(read(channel[0],&child,sizeof child)==(ssize_t)sizeof child && child>1);assert(close(channel[0])==0);
    int status=0;assert(waitpid(launcher,&status,0)==launcher && WIFEXITED(status) && WEXITSTATUS(status)==0);
    int ended=0;
    for(int i=0;i<1000;++i){pid_t result=waitpid(child,&status,WNOHANG);assert(result==0 || result==child);if(result==child){ended=1;break;}struct timespec delay={0,1000000};(void)nanosleep(&delay,NULL);}
    if(!ended){assert(kill(child,SIGKILL)==0);assert(waitpid(child,&status,0)==child);}
    assert(ended && WIFSIGNALED(status) && WTERMSIG(status)==SIGKILL);no_children();
}
int main(void){
    const char *path=getenv("PATH");assert(path!=NULL && path[0]==0);
    int base=open("/dev/null",O_RDONLY|O_CLOEXEC);assert(base>=0);int retained=fcntl(base,F_DUPFD_CLOEXEC,257);assert(retained>=257);assert(close(base)==0);
    size_t raw=open_case("https://packages.invalid/success",3000);uint8_t data[8]={0};size_t copied=0;
    assert(fairyc_private_linux_https_read_v1(raw,data,3,&copied)==0 && copied==3 && data[0]=='a' && data[1]==0 && data[2]==255);
    assert(fairyc_private_linux_https_read_v1(raw,data,8,&copied)==0 && copied==2 && data[0]=='b' && data[1]=='c');
    assert(fairyc_private_linux_https_read_v1(raw,data,1,&copied)==0 && copied==0);
    assert(fairyc_private_linux_https_close_v1(raw)==0);no_children();assert(fcntl(retained,F_GETFD)>=0);
    for(int i=0;i<2;++i){
        size_t started=0,finished=0;assert(fairyc_private_linux_monotonic_ms_v1(&started)==0);
        raw=open_case(i==0?"https://packages.invalid/slow-header":"https://packages.invalid/slow-body",100);
        assert(fairyc_private_linux_https_read_v1(raw,data,8,&copied)==ETIMEDOUT && copied==0);
        assert(fairyc_private_linux_https_close_v1(raw)==0);assert(fairyc_private_linux_monotonic_ms_v1(&finished)==0 && finished-started<1500);no_children();
    }
    raw=open_case("https://packages.invalid/slow-header",30000);assert(fairyc_private_linux_https_close_v1(raw)==0);no_children();
    const char *failures[]={"https://packages.invalid/end-error","https://packages.invalid/bad-frame","https://packages.invalid/no-end"};
    for(size_t i=0;i<3;++i){raw=open_case(failures[i],3000);assert(fairyc_private_linux_https_read_v1(raw,data,8,&copied)==(i==0?65596u:(uint32_t)EPROTO));assert(copied==0);assert(fairyc_private_linux_https_close_v1(raw)==0);no_children();}
    check_parent_death();assert(fcntl(retained,F_GETFD)>=0 && close(retained)==0);return 0;
}
`
