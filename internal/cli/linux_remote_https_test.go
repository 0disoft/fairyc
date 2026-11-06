package cli

import "testing"

func TestLinuxRemoteHTTPSPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "project_source_url", "linux_remote_https"}, "linux_remote_https", `
@effects(network,clock) fn i32 main(){
    rejects_invalid_inputs_without_starting_requests();
    preserves_native_errors_and_closes_partial_outputs_once();
    checks_headers_body_and_exact_byte_boundaries();
    preserves_binary_chunks_and_exact_url_bytes();return 0;
}
`, linuxRemoteHTTPSStub)
}

// Native I/O is deliberately substituted: this observes the actual Fairy
// response/budget/lifetime policy, not a TLS library or real Linux sockets.
const linuxRemoteHTTPSStub = `
#include <assert.h>
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
static uint32_t https_case,https_opens,https_headers,https_reads,https_closes,https_live;
static size_t https_position,https_maximum,https_timeout;
void linux_https_probe_mode(uint32_t value){
    assert(https_live==0);const char *path=getenv("PATH");assert(path==NULL || path[0]==0);
    https_case=value;https_opens=https_headers=https_reads=https_closes=0;https_position=https_maximum=https_timeout=0;
}
uint32_t linux_https_probe_count(uint32_t kind){return kind==0?https_opens:kind==1?https_headers:kind==2?https_reads:kind==3?https_closes:https_live;}
size_t linux_https_probe_budget(uint32_t kind){return kind==0?https_maximum:https_timeout;}
uint32_t fairyc_private_linux_https_open_v1(const uint8_t *address,size_t size,size_t maximum,size_t timeout,size_t *output){
    assert(address!=NULL && size==strlen((const char *)address)+1 && maximum>0 && maximum<=16777216 && timeout>0 && timeout<=30000);
    ++https_opens;https_maximum=maximum;https_timeout=timeout;*output=0;
    if(https_case==26){assert(strcmp((const char *)address,"https://packages.invalid/a%2Fb.zip")==0);}
    if(https_case==27){assert(strcmp((const char *)address,"https://packages.invalid/\xed\x95\x9c\xea\xb8\x80.zip")==0);}
    if(https_case==1){return 5;}if(https_case==3){return 0;}if(https_case==24){return 12;}if(https_case==25){return 110;}
    *output=77;https_live=1;return https_case==2 || https_case==16?5:0;
}
uint32_t fairyc_private_linux_https_headers_v1(size_t raw,uint32_t *status,uint8_t *encoding,size_t capacity,size_t *length,size_t *declared){
    assert(raw==77 && https_live==1 && capacity==128);++https_headers;*status=200;*length=0;*declared=3;
    if(https_case==4){return 61;}if(https_case==5){*length=129;}
    if(https_case==6 || https_case==23){*status=302;}if(https_case==7){memcpy(encoding,"gzip",4);*length=4;}
    if(https_case==8){*declared=4;}if(https_case==11 || https_case==15 || https_case==20 || https_case==21){*declared=SIZE_MAX;}
    if(https_case==17){*declared=0;}if(https_case==18){memcpy(encoding,"identity",8);*length=8;}
    if(https_case==19){memcpy(encoding,"Identity",8);*length=8;}if(https_case==22){*declared=2;}return 0;
}
uint32_t fairyc_private_linux_https_read_v1(size_t raw,uint8_t *data,size_t capacity,size_t *copied){
    assert(raw==77 && https_live==1 && capacity>0 && capacity<=65536);++https_reads;*copied=0;
    if(https_case==9 || https_case==14){return 62;}if(https_case==10){*copied=capacity+1;return 0;}
    size_t total=https_case==11?4:https_case==12?2:https_case==17?0:https_case==20?65536:https_case==21?65537:3;
    assert(https_position<=total);size_t count=total-https_position;if(count>capacity){count=capacity;}
    for(size_t i=0;i<count;++i){size_t at=https_position+i;data[i]=https_case==20 || https_case==21?(uint8_t)(at%251):at==0?'a':at==1?0:255;}
    https_position+=count;*copied=count;return 0;
}
uint32_t fairyc_private_linux_https_close_v1(size_t raw){
    assert(raw==77 && https_live==1);https_live=0;++https_closes;
    return https_case==13 || https_case==14 || https_case==16 || https_case==23?63:0;
}
`
