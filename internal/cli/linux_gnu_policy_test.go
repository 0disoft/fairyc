package cli

import "testing"

func TestLinuxGNUCompanionPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_gnu_policy"}, "linux_gnu_policy", `
fn i32 main(){
    admits_supported_gnu_identity_and_profiles();
    rejects_wrong_compiler_versions_and_targets();
    return 0;
}
`, "")
}
