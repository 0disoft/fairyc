package candidatehost

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestLinuxHTTPSWorkerBoundaries(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MSVC libcurl/loader substitutes; Linux TLS execution stays batched")
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	worker := strings.NewReplacer("#include <dlfcn.h>\n", "", "#include <curl/curl.h>\n", "").Replace(linuxHTTPSWorker)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, linuxHTTPSProbeTypes+linuxHTTPSProtocol+linuxHTTPSProbeCapture+linuxHTTPSCurlStub+worker+linuxHTTPSWorkerProbe)
			if err != nil {
				t.Fatal(err)
			}
			runLinuxPathProbe(t, artifact.Bytes, "https-worker.exe")
		})
	}
}

const linuxHTTPSProbeTypes = `
#include <assert.h>
#include <errno.h>
#include <stdarg.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#ifndef EOVERFLOW
#define EOVERFLOW 75
#endif
typedef int CURLcode;
typedef int CURLversion;
typedef struct {int active;} CURL;
struct curl_slist {const char *data;struct curl_slist *next;};
typedef struct {unsigned int version_num;int features;} curl_version_info_data;
typedef enum {
 CURLOPT_URL=1,CURLOPT_PROTOCOLS_STR,CURLOPT_REDIR_PROTOCOLS_STR,CURLOPT_PROXY,CURLOPT_NETRC,
 CURLOPT_DISALLOW_USERNAME_IN_URL,CURLOPT_HTTPAUTH,CURLOPT_PROXYAUTH,CURLOPT_UNRESTRICTED_AUTH,
 CURLOPT_FOLLOWLOCATION,CURLOPT_MAXREDIRS,CURLOPT_SSL_VERIFYPEER,CURLOPT_SSL_VERIFYHOST,
 CURLOPT_SSLVERSION,CURLOPT_CAINFO,CURLOPT_CAPATH,CURLOPT_TIMEOUT_MS,CURLOPT_CONNECTTIMEOUT_MS,
 CURLOPT_NOSIGNAL,CURLOPT_PATH_AS_IS,CURLOPT_HTTP_CONTENT_DECODING,CURLOPT_HTTP_TRANSFER_DECODING,
 CURLOPT_FAILONERROR,CURLOPT_HTTPHEADER,CURLOPT_HEADERFUNCTION,CURLOPT_HEADERDATA,
 CURLOPT_WRITEFUNCTION,CURLOPT_WRITEDATA,PROBE_OPTION_END
} CURLoption;
#define CURLE_OK 0
#define CURL_GLOBAL_DEFAULT 3L
#define CURLVERSION_FIRST 0
#define CURL_VERSION_SSL 4
#define CURL_NETRC_IGNORED 0
#define CURL_SSLVERSION_TLSv1_2 6
#define RTLD_NOW 2
#define RTLD_LOCAL 0
`

const linuxHTTPSProbeCapture = `
static uint8_t wire[262144];static size_t wire_size,wire_calls;
static int test_case,modules,global_live,easy_live,headers_live,perform_calls,option_calls,global_cleanups;
static uint32_t capture(void *context,const void *data,size_t size){
    assert(context==wire && size<=65536 && wire_size+size<=sizeof wire);++wire_calls;
    if(test_case==18 && wire_calls==1){return EIO;}memcpy(wire+wire_size,data,size);wire_size+=size;return 0;
}
static void reset(int value){assert(modules==0 && global_live==0 && easy_live==0 && headers_live==0);test_case=value;wire_size=wire_calls=0;perform_calls=option_calls=global_cleanups=0;}
static size_t wire_check(uint32_t wanted_end,int headers,uint32_t status,size_t declared){
    size_t at=0,body=0;int heads=0,ends=0;
    while(at<wire_size){assert(wire_size-at>=sizeof(fy_linux_https_frame));fy_linux_https_frame frame;memcpy(&frame,wire+at,sizeof frame);at+=sizeof frame;assert(frame.length<=wire_size-at);
        if(frame.kind==1){assert(frame.length==sizeof(fy_linux_https_head) && frame.code==0 && heads++==0 && body==0);fy_linux_https_head h;memcpy(&h,wire+at,sizeof h);assert(h.status==status && h.declared==declared);}
        else if(frame.kind==2){assert(heads==1 && frame.code==0 && frame.length>0 && frame.length<=65536);for(size_t i=0;i<frame.length;++i){uint8_t value=test_case==24 || test_case==25?(uint8_t)((body+i)%251):body+i==0?'a':body+i==1?0:255;assert(wire[at+i]==value);}body+=frame.length;}
        else{assert(frame.kind==3 && frame.length==0 && frame.code==wanted_end && ++ends==1 && at==wire_size);}at+=frame.length;
    }assert(heads==headers && ends==1);return body;
}
`

const linuxHTTPSCurlStub = `
static CURL easy;static struct curl_slist header_list;static curl_version_info_data version_info;
static long numbers[PROBE_OPTION_END];static const char *texts[PROBE_OPTION_END];static uint64_t seen;
typedef size_t (*probe_callback)(char *,size_t,size_t,void *);
static probe_callback header_callback,body_callback;
static void *header_context,*body_context;static struct curl_slist *request_headers;
static CURLcode stub_global_init(long flags){assert(flags==CURL_GLOBAL_DEFAULT);if(test_case==3){return 7;}++global_live;return 0;}
static void stub_global_cleanup(void){assert(global_live==1);--global_live;++global_cleanups;}
static curl_version_info_data *stub_version(CURLversion age){assert(global_live==1 && age==CURLVERSION_FIRST);version_info.version_num=test_case==5?0x075500u:0x080500u;version_info.features=test_case==6?0:CURL_VERSION_SSL;return test_case==4?NULL:&version_info;}
static CURL *stub_init(void){assert(global_live==1);if(test_case==7){return NULL;}easy.active=1;++easy_live;seen=0;memset(numbers,0,sizeof numbers);for(size_t i=0;i<PROBE_OPTION_END;++i){texts[i]=NULL;}header_callback=body_callback=NULL;header_context=body_context=NULL;return &easy;}
static void stub_cleanup(CURL *value){assert(value==&easy && easy.active && easy_live==1);easy.active=0;--easy_live;}
static struct curl_slist *stub_append(struct curl_slist *list,const char *text){assert(list==NULL && strcmp(text,"Accept: application/zip")==0);if(test_case==8){return NULL;}header_list.data=text;header_list.next=NULL;++headers_live;return &header_list;}
static void stub_free_headers(struct curl_slist *list){assert(list==&header_list && headers_live==1);--headers_live;}
static CURLcode stub_option(CURL *value,CURLoption option,...){
    assert(value==&easy && value->active && option>0 && option<PROBE_OPTION_END);++option_calls;
    assert((seen&(UINT64_C(1)<<option))==0);seen|=UINT64_C(1)<<option;va_list args;va_start(args,option);
    switch(option){
    case CURLOPT_URL:case CURLOPT_PROTOCOLS_STR:case CURLOPT_REDIR_PROTOCOLS_STR:case CURLOPT_PROXY:case CURLOPT_CAINFO:case CURLOPT_CAPATH:texts[option]=va_arg(args,const char *);break;
    case CURLOPT_HTTPHEADER:request_headers=va_arg(args,struct curl_slist *);break;
    case CURLOPT_HEADERFUNCTION:header_callback=va_arg(args,probe_callback);break;
    case CURLOPT_WRITEFUNCTION:body_callback=va_arg(args,probe_callback);break;
    case CURLOPT_HEADERDATA:header_context=va_arg(args,void *);break;
    case CURLOPT_WRITEDATA:body_context=va_arg(args,void *);break;
    default:numbers[option]=va_arg(args,long);break;
    }va_end(args);return test_case==9 && option==CURLOPT_SSL_VERIFYPEER?48:0;
}
static int feed(const char *line){size_t length=strlen(line);return header_callback((char *)line,1,length,header_context)==length;}
static CURLcode stub_perform(CURL *value){
    assert(value==&easy && easy.active && global_live==1 && headers_live==1);++perform_calls;
    assert(seen==((UINT64_C(1)<<PROBE_OPTION_END)-2));
    assert(strcmp(texts[CURLOPT_URL],"https://packages.invalid/a%2Fb.zip")==0 && strcmp(texts[CURLOPT_PROTOCOLS_STR],"https")==0 && strcmp(texts[CURLOPT_REDIR_PROTOCOLS_STR],"https")==0);
    assert(strcmp(texts[CURLOPT_PROXY],"")==0 && numbers[CURLOPT_NETRC]==CURL_NETRC_IGNORED && numbers[CURLOPT_DISALLOW_USERNAME_IN_URL]==1);
    assert(numbers[CURLOPT_HTTPAUTH]==0 && numbers[CURLOPT_PROXYAUTH]==0 && numbers[CURLOPT_UNRESTRICTED_AUTH]==0 && numbers[CURLOPT_FOLLOWLOCATION]==0 && numbers[CURLOPT_MAXREDIRS]==0);
    assert(numbers[CURLOPT_SSL_VERIFYPEER]==1 && numbers[CURLOPT_SSL_VERIFYHOST]==2 && numbers[CURLOPT_SSLVERSION]==CURL_SSLVERSION_TLSv1_2);
    assert(strcmp(texts[CURLOPT_CAINFO],"/etc/ssl/certs/ca-certificates.crt")==0 && strcmp(texts[CURLOPT_CAPATH],"/etc/ssl/certs")==0);
    assert(numbers[CURLOPT_TIMEOUT_MS]==30000 && numbers[CURLOPT_CONNECTTIMEOUT_MS]==30000 && numbers[CURLOPT_NOSIGNAL]==1 && numbers[CURLOPT_PATH_AS_IS]==1);
    assert(numbers[CURLOPT_HTTP_CONTENT_DECODING]==0 && numbers[CURLOPT_HTTP_TRANSFER_DECODING]==1 && numbers[CURLOPT_FAILONERROR]==0 && request_headers==&header_list);
    assert(header_callback!=NULL && body_callback!=NULL && header_context!=NULL && header_context==body_context);
    if(test_case==10){return 28;}if(test_case==11){return 60;}
    if(test_case==22){assert(feed("HTTP/1.1 100 Continue\r\n") && feed("X-Ignored: info\r\n") && feed("\r\n") && feed("HTTP/2 103 Early Hints\r\n") && feed("\r\n"));}
    assert(feed(test_case==12?"HTTP/1.1 302 Found\r\n":"HTTP/2 200\r\n"));
    if(test_case==14){assert(feed("Content-Length: 3\r\n") && feed("Transfer-Encoding: chunked\r\n"));if(!feed("\r\n")){return 23;}}
    else if(test_case==15){assert(feed("Content-Length: 3\r\n"));if(!feed("Content-Length: 4\r\n")){return 23;}}
    else if(test_case==24 || test_case==25 || test_case==17){assert(feed("Transfer-Encoding: chunked\r\n"));}
    else{assert(feed("Content-Length: 3\r\n") && feed("content-length: 3\r\n"));}
    assert(feed(test_case==13?"Content-Encoding: gzip\r\n":"Content-Encoding: identity\r\n"));
    if(!feed("\r\n")){return 23;}if(test_case==12){return 0;}
    static char data[65538];size_t length=test_case==16?2:test_case==17?5:test_case==24?65536:test_case==25?65537:3;
    for(size_t i=0;i<length;++i){data[i]=(char)(test_case==24 || test_case==25?(int)(i%251):i==0?'a':i==1?0:255);}
    if(body_callback(data,1,length,body_context)!=length){return 23;}
    if(test_case==16){return 18;}
    assert(feed("X-Checksum: optional-trailer\r\n") && feed("\r\n"));return 0;
}
typedef CURLcode (*init_global_type)(long);typedef void (*cleanup_global_type)(void);typedef curl_version_info_data *(*version_type)(CURLversion);
typedef CURL *(*init_type)(void);typedef CURLcode (*option_type)(CURL *,CURLoption,...);typedef CURLcode (*perform_type)(CURL *);typedef void (*cleanup_type)(CURL *);
typedef struct curl_slist *(*append_type)(struct curl_slist *,const char *);typedef void (*free_type)(struct curl_slist *);
static void *stub_dlopen(const char *path,int flags){assert(flags==(RTLD_NOW|RTLD_LOCAL));assert(strcmp(path,"/usr/lib/x86_64-linux-gnu/libcurl.so.4")==0 || strcmp(path,"/lib/x86_64-linux-gnu/libcurl.so.4")==0);if(test_case==1 || test_case==20 && path[1]=='u'){return NULL;}++modules;return &easy;}
static void *stub_dlsym(void *module,const char *name){
    assert(module==&easy && modules==1);if(test_case==2 && strcmp(name,"curl_easy_perform")==0){return NULL;}
#define PROBE_SYMBOL(text,type,fn) if(strcmp(name,text)==0){type pointer=fn;void *symbol=NULL;memcpy(&symbol,&pointer,sizeof symbol);return symbol;}
    PROBE_SYMBOL("curl_global_init",init_global_type,stub_global_init) PROBE_SYMBOL("curl_global_cleanup",cleanup_global_type,stub_global_cleanup)
    PROBE_SYMBOL("curl_version_info",version_type,stub_version) PROBE_SYMBOL("curl_easy_init",init_type,stub_init)
    PROBE_SYMBOL("curl_easy_setopt",option_type,stub_option) PROBE_SYMBOL("curl_easy_perform",perform_type,stub_perform)
    PROBE_SYMBOL("curl_easy_cleanup",cleanup_type,stub_cleanup) PROBE_SYMBOL("curl_slist_append",append_type,stub_append)
    PROBE_SYMBOL("curl_slist_free_all",free_type,stub_free_headers)
#undef PROBE_SYMBOL
    assert(0);return NULL;
}
static int stub_dlclose(void *module){assert(module==&easy && modules==1 && global_live==0 && easy_live==0 && headers_live==0);--modules;return test_case==19?-1:0;}
#define dlopen stub_dlopen
#define dlsym stub_dlsym
#define dlclose stub_dlclose
`

const linuxHTTPSWorkerProbe = `
static void parser_controls(void){
    fy_linux_https_response p;memset(&p,0,sizeof p);p.write=capture;p.sink=wire;p.maximum=3;
    reset(0);assert(fy_linux_https_line(&p,"HTTP/1.1 200 OK\r\n",17)==0);
    const char *invalid[]={"Content-Length: -1\r\n","Content-Length: 3,3\r\n","Content-Length: \r\n","Content-Length: 18446744073709551615\r\n","Content-Length: 18446744073709551616\r\n"," folded: x\r\n",": x\r\n","Bad Header: x\r\n","X: x\n","Content-Encoding: \r\n","Transfer-Encoding: gzip\r\n"};
    for(size_t i=0;i<sizeof invalid/sizeof invalid[0];++i){fy_linux_https_response copy=p;assert(fy_linux_https_line(&copy,invalid[i],strlen(invalid[i]))!=0);}
    assert(fy_linux_https_line(&p,"Content-Length: 18446744073709551614\r\n",38)==0 && p.head.declared==SIZE_MAX-1);
    fy_linux_https_response copy=p;assert(fy_linux_https_line(&copy,"Content-Length: 3\r\n",19)!=0);
    copy=p;copy.header_bytes=65535;assert(fy_linux_https_line(&copy,"\r\n",2)!=0);copy=p;copy.header_lines=1024;assert(fy_linux_https_line(&copy,"\r\n",2)!=0);
    copy=p;copy.status_seen=0;assert(fy_linux_https_line(&copy,"HTTP/. 200\r\n",12)!=0);
    copy=p;assert(fy_linux_https_header("x",2,1,&copy)==0 && copy.error!=0);
    assert(fy_linux_https_line(&p,"Content-Encoding: identity\r\n",28)==0);
    copy=p;assert(fy_linux_https_line(&copy,"Content-Encoding: identity\r\n",28)!=0);
    assert(fy_linux_https_line(&p,"\r\n",2)==0 && p.sent);
    copy=p;assert(fy_linux_https_line(&copy,"Content-Length: 3\r\n",19)!=0);
    copy=p;assert(fy_linux_https_body("a",SIZE_MAX,2,&copy)==0 && copy.error!=0);
    copy=p;copy.maximum=3;char bytes[6]={'a',0,-1,-1,-1,0};assert(fy_linux_https_body(bytes,1,5,&copy)==0 && copy.written==4 && copy.error==EOVERFLOW);
}
int main(void){
    parser_controls();
    for(int value=0;value<=25;++value){if(value==21 || value==23){continue;}reset(value);
        size_t maximum=value==24?65536:value==25?65537:3;
        uint32_t code=fy_linux_https_transfer("https://packages.invalid/a%2Fb.zip",maximum,30000,capture,wire);
        uint32_t wanted=value==1?ENOENT:value==2 || value==4 || value==5 || value==6?ENOSYS:value==3?65543u:value==7 || value==8?ENOMEM:value==9?65584u:value==10?65564u:value==11?65596u:value==13 || value==14 || value==15?EINVAL:value==16?65554u:value==17?EOVERFLOW:value==18 || value==19?EIO:0;
        assert(code==wanted && modules==0 && global_live==0 && easy_live==0 && headers_live==0);
        int headers=value==0 || value==12 || value==13 || value==16 || value==17 || value==19 || value==20 || value==22 || value==24 || value==25;
        size_t count=wire_check(wanted,headers,value==12?302u:200u,value==17 || value==24 || value==25?SIZE_MAX:3);
        size_t expected=value==0 || value==19 || value==20 || value==22?3:value==16?2:value==17?4:value==24?65536:value==25?65537:0;assert(count==expected);
        if(value>=1 && value<=9){assert(perform_calls==0);}else{assert(perform_calls==1);}
    }
    reset(0);assert(fy_linux_https_transfer(NULL,3,30000,capture,wire)==EINVAL && wire_size==0);
    assert(fy_linux_https_transfer("x",0,30000,capture,wire)==EINVAL && wire_size==0);
    assert(fy_linux_https_transfer("x",16777217,30000,capture,wire)==EINVAL && wire_size==0);
    assert(fy_linux_https_transfer("x",3,0,capture,wire)==EINVAL && wire_size==0);
    assert(fy_linux_https_transfer("x",3,30001,capture,wire)==EINVAL && wire_size==0);
    assert(fy_linux_https_transfer("x",3,30000,NULL,wire)==EINVAL && wire_size==0);return 0;
}
`
