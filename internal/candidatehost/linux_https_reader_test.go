package candidatehost

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestLinuxHTTPSReaderBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC pipe-input substitutes; Linux runtime execution stays batched")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxHTTPSReaderProbe+linuxHTTPSProtocol+linuxHTTPSReader+linuxHTTPSReaderCases)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxHTTPSProbe(t, artifact.Bytes, "https-reader.exe")
		})
	}
}

func runLinuxHTTPSProbe(t *testing.T, image []byte, name string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, image, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	cmd.Env = append(os.Environ(), "PATH=")
	if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
		t.Fatalf("HTTPS probe: %v %s", err, out)
	}
}

const linuxHTTPSReaderProbe = `
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#ifndef EPROTO
#define EPROTO 71
#endif
#ifndef EOVERFLOW
#define EOVERFLOW 75
#endif
static uint8_t wire[132000];static size_t wire_length,position,step,fault_at;static uint32_t fault;
static uint32_t input(void *context,void *data,size_t capacity,size_t *copied){
    assert(context==wire && capacity>0);*copied=0;if(position>=fault_at && fault!=0){return fault;}
    size_t size=wire_length-position;if(size>capacity){size=capacity;}if(size>step){size=step;}
    memcpy(data,wire+position,size);position+=size;*copied=size;return 0;
}
static uint32_t output(void *context,const void *data,size_t size){assert(context==wire && size<=sizeof wire-wire_length);memcpy(wire+wire_length,data,size);wire_length+=size;return 0;}
`

const linuxHTTPSReaderCases = `
static fy_linux_https_response producer;
static fy_linux_https_reader reader;
static void reset(size_t maximum){
    wire_length=position=0;step=SIZE_MAX;fault_at=SIZE_MAX;fault=0;
    memset(&producer,0,sizeof producer);producer.write=output;producer.sink=wire;producer.maximum=maximum;
    memset(&reader,0,sizeof reader);reader.input=input;reader.context=wire;reader.maximum=maximum;
}
/* Independent byte layout; do not use the producer to create valid fixtures. */
static void frame(uint32_t kind,uint32_t count,uint32_t code){uint32_t words[3]={kind,count,code};assert(output(wire,words,12)==0);}
static void head(void){uint8_t bytes[144]={0};uint32_t status=200;size_t absent=SIZE_MAX;memcpy(bytes,&status,4);memcpy(bytes+8,&absent,8);frame(1,144,0);assert(output(wire,bytes,sizeof bytes)==0);}
static void body(size_t count){static uint8_t data[65536];assert(count<=sizeof data);for(size_t i=0;i<count;++i){data[i]=(uint8_t)(i%251);}frame(2,(uint32_t)count,0);assert(output(wire,data,count)==0);}
static void end(uint32_t code){frame(3,0,code);}
static void good(void){head();body(2);body(3);end(0);}
static void failure(uint32_t wanted){uint8_t data[65536];size_t copied=77;assert(fy_linux_https_reader_read(&reader,data,sizeof data,&copied)==wanted && copied==0);size_t at=position;assert(fy_linux_https_reader_read(&reader,data,1,&copied)==wanted && position==at && copied==0);}
static void malformed(void){
    reset(100);frame(2,1,0);failure(EPROTO);
    reset(100);frame(4,0,0);failure(EPROTO);
    reset(100);frame(1,143,0);failure(EPROTO);
    reset(100);frame(1,144,EIO);failure(EPROTO);
    reset(100);end(0);failure(EPROTO);
    reset(100);end(65596);failure(65596);
    reset(100);head();head();failure(EPROTO);
    reset(100);head();frame(2,0,0);failure(EPROTO);
    reset(100);head();frame(2,65537,0);failure(EPROTO);
    reset(100);head();frame(2,1,EIO);failure(EPROTO);
    reset(100);head();frame(3,1,0);failure(EPROTO);
    reset(100);head();end(0);wire[wire_length++]=1;failure(EPROTO);
    reset(100);head();uint32_t bad=600;memcpy(wire+12,&bad,4);failure(EPROTO);
    reset(100);head();bad=99;memcpy(wire+12,&bad,4);failure(EPROTO);
    reset(100);head();bad=129;memcpy(wire+16,&bad,4);failure(EPROTO);
    reset(100);head();bad=1;memcpy(wire+16,&bad,4);wire[28]=0;failure(EPROTO);
    reset(1);head();body(3);failure(EPROTO);
    reset(1);head();body(2);body(1);failure(EPROTO);
    reset(100);good();fault_at=156;fault=ETIMEDOUT;failure(ETIMEDOUT);
    reset(100);head();body(3);end(EIO);failure(EIO);
}
int main(void){
    for(size_t stride=1;stride<=200;stride+=7){
        reset(5);good();step=stride;assert(fy_linux_https_reader_headers(&reader)==0 && reader.head.status==200 && reader.head.declared==SIZE_MAX);
        uint8_t data[8]={0};size_t count=99;assert(fy_linux_https_reader_read(&reader,data,1,&count)==0 && count==1 && data[0]==0 && reader.remaining==1);
        assert(fy_linux_https_reader_headers(&reader)==0);
        assert(fy_linux_https_reader_read(&reader,data,8,&count)==0 && count==4 && data[0]==1 && data[1]==0 && data[2]==1 && data[3]==2);
        assert(fy_linux_https_reader_read(&reader,data,8,&count)==0 && count==0 && reader.ended);
    }
    reset(65536);head();body(65536);body(1);end(0);uint8_t data[65536];size_t count=0;
    assert(fy_linux_https_reader_read(&reader,data,sizeof data,&count)==0 && count==65536);for(size_t i=0;i<count;++i){assert(data[i]==(uint8_t)(i%251));}
    assert(fy_linux_https_reader_read(&reader,data,1,&count)==0 && count==1 && data[0]==0);
    assert(fy_linux_https_reader_read(&reader,data,1,&count)==0 && count==0);
    reset(100);good();size_t complete=wire_length;
    for(size_t cut=0;cut<complete;++cut){reset(100);good();wire_length=cut;failure(EPROTO);}
    malformed();
    /* Exercise the actual producer/reader composition too, separately from
     * byte-authored valid and malformed fixtures. */
    reset(100);assert(fy_linux_https_line(&producer,"HTTP/2 200\r\n",12)==0);assert(fy_linux_https_line(&producer,"\r\n",2)==0);
    char bytes[3]={'a',0,(char)-1};assert(fy_linux_https_header("X: v\r\n",1,6,&producer)==6);
    assert(fy_linux_https_body(bytes,1,3,&producer)==3);assert(fy_linux_https_send(&producer,3,0,NULL,0)==0);
    assert(fy_linux_https_reader_read(&reader,data,4,&count)==0 && count==3 && memcmp(data,bytes,3)==0);
    return 0;
}
`
