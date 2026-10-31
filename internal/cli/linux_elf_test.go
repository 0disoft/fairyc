package cli

import "testing"

func TestLinuxGNUCompanionELF(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_elf"}, "linux_elf", `
fn i32 main(){
    validates_relocatable_and_executable_ranges();
    rejects_truncated_foreign_and_overflowed_headers();
    return 0;
}
`, "")
}
