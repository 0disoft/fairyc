@TRAP_RECEIPT_INCLUDE@#include <stdint.h>
#include <stdlib.h>

#define FY_RUNTIME_TRAP_EXIT_BASE UINT32_C(@TRAP_EXIT_BASE@)

_Noreturn void fy_runtime_trap(uint32_t category) {
    if (category < UINT32_C(1) || category > UINT32_C(7)) {
        _Exit(255);
    }
@TRAP_RECEIPT_WRITE@    _Exit((int)(FY_RUNTIME_TRAP_EXIT_BASE + category));
}
