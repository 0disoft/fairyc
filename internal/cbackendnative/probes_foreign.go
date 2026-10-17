package cbackendnative

import (
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
)

func runForeignCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-call",
		foreignCallSmokeSource,
		foreignCallCompanionSource,
		42,
		[]string{
			"extern int32_t fairyc_fixture_add_one(int32_t fy_a_0);",
			"fairyc_fixture_add_one(INT32_C(41))",
		},
	)
}

func runForeignCIntCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-int-call",
		foreignCIntCallSmokeSource,
		foreignCIntCallCompanionSource,
		42,
		[]string{
			"_Static_assert(sizeof(int) == UINT64_C(4), \"fairyc target c.int size\");",
			"_Static_assert(_Alignof(int) == UINT64_C(4), \"fairyc target c.int alignment\");",
			"extern int fairyc_fixture_make_c_int(void);",
			"extern int32_t fairyc_fixture_consume_c_int(int fy_a_0);",
			"fairyc_fixture_make_c_int()",
			"fairyc_fixture_consume_c_int(fy_l_",
		},
	)
}

func runForeignCLongCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-long-call",
		foreignCLongCallSmokeSource,
		foreignCLongCallCompanionSource,
		42,
		[]string{
			"_Static_assert(sizeof(long) == UINT64_C(8), \"fairyc target c.long size\");",
			"_Static_assert(_Alignof(long) == UINT64_C(8), \"fairyc target c.long alignment\");",
			"_Static_assert(((long)-1) < (long)0, \"fairyc target c.long signed\");",
			"extern long fairyc_fixture_make_c_long(void);",
			"extern int32_t fairyc_fixture_consume_c_long(long fy_a_0);",
			"fairyc_fixture_make_c_long()",
			"fairyc_fixture_consume_c_long(fy_l_",
		},
	)
}

func runForeignCScalarFamilyCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-scalar-family-call",
		foreignCScalarFamilyCallSmokeSource,
		foreignCScalarFamilyCallCompanionSource,
		42,
		foreignCScalarFamilyExpectedCFragments,
	)
}

func runForeignCSizeTCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-size-t-call",
		foreignCSizeTCallSmokeSource,
		foreignCSizeTCallCompanionSource,
		42,
		[]string{
			"#include <stddef.h>",
			"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
			"_Static_assert(_Alignof(size_t) == UINT64_C(8), \"fairyc target c.size_t alignment\");",
			"extern size_t fairyc_fixture_make_c_size_t(void);",
			"extern int32_t fairyc_fixture_consume_c_size_t(size_t fy_a_0);",
			"fairyc_fixture_make_c_size_t()",
			"fairyc_fixture_consume_c_size_t(fy_l_",
		},
	)
}

func runForeignCSSizeTCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-ssize-t-call",
		foreignCSSizeTCallSmokeSource,
		foreignCSSizeTCallCompanionSource,
		42,
		[]string{
			"#define _POSIX_C_SOURCE 200809L",
			"#include <sys/types.h>",
			"_Static_assert(sizeof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t size\");",
			"_Static_assert(_Alignof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t alignment\");",
			"extern ssize_t fairyc_fixture_make_c_ssize_t(void);",
			"extern int32_t fairyc_fixture_consume_c_ssize_t(ssize_t fy_a_0);",
			"fairyc_fixture_make_c_ssize_t()",
			"fairyc_fixture_consume_c_ssize_t(fy_l_",
		},
	)
}

func runForeignPointerCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-pointer-call",
		foreignPointerCallSmokeSource,
		foreignPointerCallCompanionSource,
		42,
		[]string{
			"extern uint32_t fairyc_fixture_pointer_scalars(const uint8_t * fy_a_0, uint32_t * fy_a_1, uintptr_t fy_a_2, uint8_t fy_a_3);",
			"fairyc_fixture_pointer_scalars(((void *)0), ((void *)0), ((uintptr_t)UINT64_C(0)), UINT8_C(7))",
		},
	)
}

func runForeignCStructResultCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	unit := foreignstructprobe.Generate()
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-struct-result-call",
		unit.FairySource,
		unit.CompanionC,
		unit.ExpectedExit,
		unit.RequiredCFragments,
	)
}
