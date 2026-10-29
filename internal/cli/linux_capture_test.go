package cli

import "testing"

func TestLinuxCapturePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_capture"}, "linux_capture", `
@effects(filesystem,environment,console,clock) fn i32 main() {
    owns_binary_output_and_classifies_limits_failures_and_timeout(); return 0;
}
`, linuxCapturePolicyStub)
}

const linuxCapturePolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t mode,opens,collects,drops;
void capture_mode(uint32_t value) { mode=value;opens=0;collects=0;drops=0; }
uint32_t capture_counts(void) { return opens*100+collects*10+drops; }
uint32_t fairyc_private_linux_capture_open_v1(const uint8_t *data,size_t size,size_t argc,const uint8_t *cwd,size_t cwd_size,void **output) {
    assert(size==9 && argc==3 && memcmp(data,"app\0\0a b",9)==0 && cwd_size==5 && strcmp((const char *)cwd,"/tmp")==0);
    ++opens;*output=NULL;if(mode==5) {return 2;}if(mode!=10) {*output=(void *)(uintptr_t)4321;}return 0;
}
uint32_t fairyc_private_linux_capture_collect_v1(void *raw,uint32_t timeout_ms,
        uint8_t *out,size_t out_cap,size_t *out_len,uint8_t *err,size_t err_cap,size_t *err_len,
        uint32_t *kind,uint32_t *value,uint32_t *pid) {
    assert(raw==(void *)(uintptr_t)4321 && timeout_ms==25);++collects;
    *pid=4321;*kind=mode<3 ? mode+1 : 4;*value=mode==0 ? 127 : mode==1 ? 9 : mode==2 ? 0 : mode-2;
    if(mode==15) {assert(out_cap==0 && err_cap==0 && out==NULL && err==NULL);*out_len=0;*err_len=0;*kind=1;*value=0;return 0;}
    assert(out_cap==3 && err_cap==3);memcpy(out,"o\0x",3);memcpy(err,"\377ex",3);
    *out_len=mode==3 ? 3 : 2;*err_len=mode==4 ? 3 : 2;
    if(mode==6) {return 5;}if(mode==7) {*out_len=4;}
    if(mode==8) {*kind=4;*value=3;}if(mode==9) {*pid=1;}
    if(mode==11) {*kind=3;*value=7;}if(mode==12) {*kind=1;*value=256;}
    if(mode==13 || mode==14) {*kind=2;*value=mode==13 ? 0 : 65;}return 0;
}
uint32_t fairyc_private_linux_capture_drop_v1(void *raw) {
    if(raw!=NULL) {assert(raw==(void *)(uintptr_t)4321);++drops;}return 0;
}
`
