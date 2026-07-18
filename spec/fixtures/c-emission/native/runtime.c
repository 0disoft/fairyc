#include <stdint.h>
#include <stdlib.h>

#ifndef FY_TEST_TRAP_EXIT_BASE
#error "FY_TEST_TRAP_EXIT_BASE must be supplied by the native fixture runner"
#endif

_Noreturn void fy_runtime_trap(uint32_t category) {
    if (category < UINT32_C(1) || category > UINT32_C(7)) {
        _Exit(255);
    }
    _Exit(FY_TEST_TRAP_EXIT_BASE + (int)category);
}
