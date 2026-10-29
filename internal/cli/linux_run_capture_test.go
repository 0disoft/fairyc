package cli

import "testing"

func TestNativeRunControl(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"native_run_control", "windows_run_control", "windows_command", "windows_process", "project_json_string", "project_path", "project_path_unicode_data"}, "native_run_control", `
fn i32 main(){preserves_all_normal_exit_bits_and_requires_exact_trap_receipts();validates_extended_trap_location_records();formats_snapshot_locations_with_unicode_columns_and_safe_paths();keeps_windows_wrapper_results_and_locations();return 0;}
`, "")
}
func TestLinuxRunCapturePolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_run_capture", "native_run_control", "project_json_string", "project_path", "project_path_unicode_data"}, "linux_run_capture", `
@effects(filesystem,environment,console,clock) fn i32 main(){captures_exit_signal_limits_and_authenticated_receipts();return 0;}
`, linuxRunCapturePolicyStub)
}

const linuxRunCapturePolicyStub = `
#include <stdint.h>
#include <stddef.h>
#include <assert.h>
#include <string.h>
static uint32_t mode,drops;
void run_capture_mode(uint32_t value){mode=value;drops=0;}
uint32_t run_capture_drops(void){return drops;}
uint32_t fairyc_private_linux_run_capture_open_v1(const uint8_t *data,size_t size,size_t argc,const uint8_t *cwd,size_t cwd_size,void **output){
 assert(size==9 && argc==3 && memcmp(data,"app\0\0a b",9)==0 && cwd_size==5 && strcmp((const char *)cwd,"/tmp")==0);*output=(void *)(uintptr_t)4321;return mode==11 ? 2 : 0;
}
uint32_t fairyc_private_linux_capture_collect_v1(void *raw,uint32_t timeout,uint8_t *out,size_t out_cap,size_t *out_len,uint8_t *err,size_t err_cap,size_t *err_len,uint32_t *kind,uint32_t *value,uint32_t *pid){
 assert(raw==(void *)(uintptr_t)4321 && timeout==25 && out_cap==3 && err_cap==3);
 memcpy(out,"o\0",2);memcpy(err,"\377e",2);*out_len=*err_len=2;*pid=4321;
 *kind=mode==3 || mode==7 ? 2 : mode==4 ? 3 : mode==5 ? 4 : 1;
 *value=*kind==2 ? 9 : mode==4 ? 0 : mode==5 ? 1 : mode==0 ? 127 : 133;return mode==10 ? 5 : 0;
}
uint32_t fairyc_private_linux_run_capture_receipt_v1(void *raw,uint8_t *data,size_t capacity,size_t *length){
 assert(raw==(void *)(uintptr_t)4321 && capacity==276);
 *length=mode==1 || mode==6 || mode==7 ? 1 : mode==9 ? 277 : 0;data[0]=mode==6 ? 2 : 5;return mode==8 ? 5 : 0;
}
uint32_t fairyc_private_linux_capture_drop_v1(void *raw){if(raw!=NULL){assert(raw==(void *)(uintptr_t)4321);++drops;}return 0;}
`

func TestLinuxRunOptions(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"native_cli_options"}, "native_cli_options", `
fn i32 main(){preserves_flags_and_all_arguments_after_the_first_delimiter();rejects_unknown_duplicates_and_nul_with_reference_precedence();return 0;}
`, "")
}
