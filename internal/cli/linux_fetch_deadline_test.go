package cli

import "testing"

func TestLinuxFetchDeadlinePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_fetch_deadline"}, "linux_fetch_deadline", `
@effects(clock) fn i32 main(){observes_exact_deadline_and_overflow_boundaries();fails_closed_on_clock_errors_or_reverse_time();return 0;}
`, `
#include <assert.h>
#include <stdint.h>
#include <stddef.h>
static uint32_t clock_case,clock_calls;
void fetch_clock_probe_mode(uint32_t value){clock_case=value;clock_calls=0;}
uint32_t fetch_clock_probe_calls(void){return clock_calls;}
uint32_t fairyc_private_linux_monotonic_ms_v1(size_t *output){
    ++clock_calls;*output=0;if(clock_case==1 || (clock_case==3 && clock_calls>1)){return 5;}*output=1000;
    if(clock_calls>1){*output=clock_case==2?999:clock_case==4?121000:clock_case==5?120999:1000+clock_calls-1;}return 0;
}
`)
}
