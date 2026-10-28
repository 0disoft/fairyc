package cli

import (
	"strings"
	"testing"
)

func TestStableWorkspaceTestRegionByteBacking(t *testing.T) {
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/bytes.fy", `module tests::bytes;
import std::alloc as alloc;
fn alloc::Arena make() {
    return switch (alloc::create()) {
        case .ok(arena): { arena }
        case .err(_): { assert(false); make() }
    };
}
fn Result<i32, alloc::AllocError> exercise(Region r) {
    let ref mut u8[] bytes = r.alloc_zeroed_bytes(8)?;
    assert(bytes[3] == 0);
    {
        let ref mut u8[] sub = bytes[2..6];
        let ref mut u8 cell = ref mut sub[1];
        *cell = 42;
    }
    assert(bytes[3] == 42);
    let ref mut u8[] copied = r.copy_bytes(bytes[2..6])?;
    copied[1] = 7;
    assert(bytes[3] == 42);
    assert(copied[1] == 7);
    return .ok(42);
}
@test fn void region_bytes() {
    var alloc::Arena arena = make();
    region r in ref mut arena {
        assert(switch (exercise(r)) { case .ok(n): { n == 42 } case .err(_): { false } });
    }
}
`)
	code, stdout, stderr := runForTest([]string{"test"}, root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "1 passed; 0 failed") {
		t.Fatalf("Region byte test: exit=%d, stdout=%s, stderr=%s", code, stdout, stderr)
	}
}
