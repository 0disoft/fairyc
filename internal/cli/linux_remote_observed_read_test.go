package cli

import "testing"

func TestLinuxRemoteObservedReadPolicy(t *testing.T) {
	runLinuxCandidateFixture(t, []string{"linux_command", "linux_rooted_file", "linux_directory", "linux_package_directory", "linux_package_file", "linux_remote_fs"}, "linux_remote_observed_read", `
@effects(filesystem) fn i32 main(){
    matches_every_observed_field_before_reading();
    rejects_invalid_facts_paths_and_bounds_without_reading();
    retains_complete_read_and_single_link_checks();return 0;
}
`, linuxRemoteFileSystemStub)
}
