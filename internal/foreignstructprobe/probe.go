// Package foreignstructprobe owns the shared generated-Fairy aggregate argument/result
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

// Generate returns a fresh aggregate argument/result crossing. The records force
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
			"extern struct fairyc_fixture_status_v0 fairyc_fixture_change_status_v0(struct fairyc_fixture_status_v0 fy_a_0, int32_t fy_a_1);",
			"extern struct fairyc_fixture_large_status_v0 fairyc_fixture_change_large_v0(int32_t fy_a_0, struct fairyc_fixture_large_status_v0 fy_a_1, int32_t fy_a_2);",
			"struct fairyc_fixture_mixed_status_v0 {",
			"double ratio;",
			"extern int32_t fairyc_fixture_exhausted_v0(int32_t fy_a_0, int32_t fy_a_1, int32_t fy_a_2, int32_t fy_a_3, int32_t fy_a_4, int32_t fy_a_5, struct fairyc_fixture_mixed_status_v0 fy_a_6, int32_t fy_a_7);",
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
@repr(C)
struct fairyc_fixture_mixed_status_v0 {
    fairyc_fixture_status_v0 detail;
    c::double ratio;
}
@link_name("fairyc_fixture_make_status_v0")
unsafe extern C fn fairyc_fixture_status_v0 make_status() ;
@link_name("fairyc_fixture_make_large_status_v0")
unsafe extern C fn fairyc_fixture_large_status_v0 make_large_status() ;
@link_name("fairyc_fixture_change_status_v0")
unsafe extern C fn fairyc_fixture_status_v0 change_status(fairyc_fixture_status_v0 value, i32 marker);
@link_name("fairyc_fixture_change_large_v0")
unsafe extern C fn fairyc_fixture_large_status_v0 change_large(i32 before, fairyc_fixture_large_status_v0 value, i32 after);
@link_name("fairyc_fixture_make_mixed_v0")
unsafe extern C fn fairyc_fixture_mixed_status_v0 make_mixed();
@link_name("fairyc_fixture_exhausted_v0")
unsafe extern C fn i32 exhausted(i32 a, i32 b, i32 c, i32 d, i32 e, i32 f, fairyc_fixture_mixed_status_v0 value, i32 tail);
fn i32 main() {
    let fairyc_fixture_status_v0 result = unsafe { make_status() };
    let fairyc_fixture_large_status_v0 large = unsafe { make_large_status() };
    let fairyc_fixture_status_v0 changed = unsafe { change_status(result, 7) };
    let fairyc_fixture_large_status_v0 changed_large = unsafe { change_large(11, large, 13) };
    let fairyc_fixture_mixed_status_v0 mixed = unsafe { make_mixed() };
    let i32 exhausted_result = unsafe { exhausted(1, 2, 3, 4, 5, 6, mixed, 8) };
    return if (result.status == 41 && result.allocation_status == 1 &&
        large.first == 1 && large.second == 2 && large.third == 3 &&
        large.fourth == 4 && large.fifth == 5 &&
        changed.status == 99 && changed.allocation_status == 7 &&
        changed_large.first == 101 && changed_large.second == 2 &&
        changed_large.third == 3 && changed_large.fourth == 4 && changed_large.fifth == 105 &&
        mixed.detail.status == 17 && mixed.detail.allocation_status == 19 && exhausted_result == 42) {
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

struct fairyc_fixture_mixed_status_v0 {
    struct fairyc_fixture_status_v0 detail;
    double ratio;
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

struct fairyc_fixture_status_v0 fairyc_fixture_change_status_v0(
    struct fairyc_fixture_status_v0 value, int32_t marker
) {
    if (value.status != UINT32_C(41) || value.allocation_status != UINT32_C(1) || marker != 7) {
        struct fairyc_fixture_status_v0 failed = {0, 0};
        return failed;
    }
    value.status = UINT32_C(99);
    value.allocation_status = UINT32_C(7);
    return value;
}

struct fairyc_fixture_large_status_v0 fairyc_fixture_change_large_v0(
    int32_t before, struct fairyc_fixture_large_status_v0 value, int32_t after
) {
    if (before != 11 || after != 13 || value.first != 1 || value.second != 2 ||
        value.third != 3 || value.fourth != 4 || value.fifth != 5) {
        struct fairyc_fixture_large_status_v0 failed = {0, 0, 0, 0, 0};
        return failed;
    }
    value.first = UINT32_C(101);
    value.fifth = UINT32_C(105);
    return value;
}

struct fairyc_fixture_mixed_status_v0 fairyc_fixture_make_mixed_v0(void) {
    struct fairyc_fixture_mixed_status_v0 value = {{17, 19}, -1.5};
    return value;
}

int32_t fairyc_fixture_exhausted_v0(
    int32_t a, int32_t b, int32_t c, int32_t d, int32_t e, int32_t f,
    struct fairyc_fixture_mixed_status_v0 value, int32_t tail
) {
    /* All six SysV INTEGER argument registers are occupied before the mixed record. */
    return a == 1 && b == 2 && c == 3 && d == 4 && e == 5 && f == 6 && tail == 8 &&
        value.detail.status == 17 && value.detail.allocation_status == 19 && value.ratio == -1.5
        ? INT32_C(42) : INT32_C(0);
}
`
