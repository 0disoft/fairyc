package cli

import "testing"

func TestLinuxProcessPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_process"}, "linux_process", `
@effects(filesystem,environment,console) fn i32 main() {
    separates_launch_failures_exit_status_signals_and_cleanup(); return 0;
}
`, linuxProcessPolicyStub)
}

const linuxProcessPolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t scenario,starts,waits,discards;
void probe_mode(uint32_t value) { scenario=value;starts=0;waits=0;discards=0; }
uint32_t probe_counts(void) { return starts*100+waits*10+discards; }
uint32_t fairyc_private_linux_process_start_v1(const uint8_t *data,size_t size,size_t argc,
        const uint8_t *cwd,size_t cwd_size,uint32_t *pid) {
    assert(size==9 && argc==3 && memcmp(data,"app\0\0a b",9)==0 && cwd_size==5 && strcmp((const char *)cwd,"/tmp")==0);
    ++starts; *pid=0;
    if(scenario==3) { return 2; } if(scenario==4) { return 13; }
    if(scenario!=6) { *pid=4321; } return 0;
}
uint32_t fairyc_private_linux_process_wait_v1(uint32_t pid,uint32_t *kind,uint32_t *value) {
    assert(pid==4321); ++waits;
    if(scenario==5) { return 22; }
    *kind=scenario==2 ? 2 : 1; *value=scenario==0 ? 127 : scenario==1 ? 41 : 15;
    if(scenario==7) { *kind=2; *value=0; } if(scenario==8) { *value=256; } return 0;
}
uint32_t fairyc_private_linux_process_discard_v1(uint32_t pid) {
    if(pid!=0) { assert(pid==4321); ++discards; } return 0;
}
`
