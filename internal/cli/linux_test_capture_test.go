package cli

import (
	"strings"
	"testing"
)

func TestLinuxTestCapturePolicy(t *testing.T) {
	stub := strings.ReplaceAll(linuxRunCapturePolicyStub, "fairyc_private_linux_run_capture_open_v1", "fairyc_private_linux_test_capture_open_v1")
	stub = strings.Replace(stub, "size_t cwd_size,void **output)", "size_t cwd_size,uint32_t input_mode,const uint8_t *path,size_t path_size,void **output)", 1)
	stub = strings.Replace(stub, "*output=(void *)(uintptr_t)4321;return mode==11", `assert(input_mode<=2);if(input_mode==2){assert(path_size==12 && memcmp(path,"/tmp/replay",12)==0);}else{assert(path_size==0);}*output=(void *)(uintptr_t)4321;return mode==11`, 1)
	stub = strings.ReplaceAll(stub, "out_cap==3 && err_cap==3", "out_cap==65536 && err_cap==65536")
	// run_for remains in the same source module; its unused opener has no reachable symbol.
	runLinuxCandidateFixture(t, []string{"linux_test_capture", "linux_run_capture", "linux_command", "native_run_control", "project_json_string", "project_path", "project_path_unicode_data"}, "linux_test_capture", `
@effects(filesystem,environment,console,clock) fn i32 main(){forwards_stdin_modes_and_owns_failed_open();return 0;}
`, stub)
}
