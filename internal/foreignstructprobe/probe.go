// Package foreignstructprobe owns the shared generated-Fairy aggregate-result
// crossing used by native C backend conformance runners.
package foreignstructprobe

// Unit describes one Fairy caller, its independently compiled C callee, and
// the generated-C fragments that must survive lowering and emission.
type Unit struct {
	FairySource        string
	CompanionC         string
	ExpectedExit       int
	RequiredCFragments []string
}

// Generate returns a fresh aggregate-result crossing. The two records force
// both the register-sized and hidden-result-pointer paths on supported AMD64
// C ABIs while keeping the observable result target-independent.
func Generate() Unit {
	return Unit{
		FairySource:  fairySource,
		CompanionC:   companionC,
		ExpectedExit: 42,
		RequiredCFragments: []string{
			"struct fairyc_fixture_status_v0 {",
			"uint32_t status;",
			"uint32_t allocation_status;",
			"_Static_assert(sizeof(struct fairyc_fixture_status_v0) == UINT64_C(8)",
			"_Static_assert(_Alignof(struct fairyc_fixture_status_v0) == UINT64_C(4)",
			"offsetof(struct fairyc_fixture_status_v0, status) == UINT64_C(0)",
			"offsetof(struct fairyc_fixture_status_v0, allocation_status) == UINT64_C(4)",
			"extern struct fairyc_fixture_status_v0 fairyc_fixture_make_status_v0(void);",
			" = fairyc_fixture_make_status_v0();",
			"struct fairyc_fixture_large_status_v0 {",
			"_Static_assert(sizeof(struct fairyc_fixture_large_status_v0) == UINT64_C(20)",
			"offsetof(struct fairyc_fixture_large_status_v0, fifth) == UINT64_C(16)",
			"extern struct fairyc_fixture_large_status_v0 fairyc_fixture_make_large_status_v0(void);",
			" = fairyc_fixture_make_large_status_v0();",
		},
	}
}

const fairySource = `module main;
@repr(C)
struct fairyc_fixture_status_v0 {
    u32 status;
    u32 allocation_status;
}
@repr(C)
struct fairyc_fixture_large_status_v0 {
    u32 first;
    u32 second;
    u32 third;
    u32 fourth;
    u32 fifth;
}
@link_name("fairyc_fixture_make_status_v0")
unsafe extern C fn fairyc_fixture_status_v0 make_status() ;
@link_name("fairyc_fixture_make_large_status_v0")
unsafe extern C fn fairyc_fixture_large_status_v0 make_large_status() ;
fn i32 main() {
    let fairyc_fixture_status_v0 result = unsafe { make_status() };
    let fairyc_fixture_large_status_v0 large = unsafe { make_large_status() };
    return if (result.status == 41 && result.allocation_status == 1 &&
        large.first == 1 && large.second == 2 && large.third == 3 &&
        large.fourth == 4 && large.fifth == 5) {
        42
    } else {
        0
    };
}
`

const companionC = `#include <stdint.h>

struct fairyc_fixture_status_v0 {
    uint32_t status;
    uint32_t allocation_status;
};

struct fairyc_fixture_large_status_v0 {
    uint32_t first;
    uint32_t second;
    uint32_t third;
    uint32_t fourth;
    uint32_t fifth;
};

struct fairyc_fixture_status_v0 fairyc_fixture_make_status_v0(void) {
    struct fairyc_fixture_status_v0 result = {UINT32_C(41), UINT32_C(1)};
    return result;
}

struct fairyc_fixture_large_status_v0 fairyc_fixture_make_large_status_v0(void) {
    struct fairyc_fixture_large_status_v0 result = {
        UINT32_C(1), UINT32_C(2), UINT32_C(3), UINT32_C(4), UINT32_C(5)
    };
    return result;
}
`
