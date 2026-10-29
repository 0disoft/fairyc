package cli

import "testing"

func TestLinuxBuildScratchPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_build_scratch", "linux_command"}, "linux_build_scratch", `@effects(filesystem) fn i32 main(){owns_exact_path_and_consuming_cleanup();return 0;}`, `
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <assert.h>
static uint32_t scenario,closes;
void build_scratch_mode(uint32_t mode){scenario=mode;closes=0;}
uint32_t build_scratch_closes(void){return closes;}
uint32_t fairyc_private_linux_build_scratch_open_v1(const uint8_t *root,size_t size,uint8_t *path,size_t capacity,size_t *length,size_t *out){assert(size==5 && strcmp((const char *)root,"/tmp")==0 && capacity==4096);*out=8;memcpy(path,"/tmp/scratch",13);*length=scenario==2?4096:12;return scenario==3?13:0;}
uint32_t fairyc_private_linux_build_scratch_drop_v1(size_t raw){if(raw==0){return 0;}assert(raw==8);++closes;return scenario==1?5:0;}
`)
}
