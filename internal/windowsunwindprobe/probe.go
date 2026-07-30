// Package windowsunwindprobe generates a closed Microsoft x64 structured-
// exception fixture for native Windows execution.
package windowsunwindprobe

import "slices"

const (
	FrameSymbol            = "fy_unwind_frame"
	FrameObject            = "frame.obj"
	LowHandlerSymbol       = "__GSHandlerCheck_SEH"
	OptimizedHandlerSymbol = "__C_specific_handler"
)

// Unit contains three separately compiled C translation units and the
// observable runtime contracts they exercise.
type Unit struct {
	Cases                  []string
	CatcherSource          string
	FrameSource            string
	RaiserSource           string
	FrameSymbol            string
	FrameObject            string
	LowHandlerSymbol       string
	OptimizedHandlerSymbol string
}

var cases = []string{
	"abnormal-finally",
	"exception-filter-payload",
	"virtual-unwind-walk",
}

// Generate returns deterministic MSVC C sources that raise a private software
// exception, walk its captured context, and then perform the real unwind.
func Generate() Unit {
	return Unit{
		Cases:                  slices.Clone(cases),
		CatcherSource:          catcherSource,
		FrameSource:            frameSource,
		RaiserSource:           raiserSource,
		FrameSymbol:            FrameSymbol,
		FrameObject:            FrameObject,
		LowHandlerSymbol:       LowHandlerSymbol,
		OptimizedHandlerSymbol: OptimizedHandlerSymbol,
	}
}

// HandlerSymbolForProfile returns the exact MSVC handler emitted for the closed probe profile.
func (unit Unit) HandlerSymbolForProfile(profile string) (string, bool) {
	switch profile {
	case "low":
		return unit.LowHandlerSymbol, unit.LowHandlerSymbol != ""
	case "optimized":
		return unit.OptimizedHandlerSymbol, unit.OptimizedHandlerSymbol != ""
	default:
		return "", false
	}
}

// CloneCases returns an isolated case-ID copy for callers that retain receipts.
func CloneCases(unit Unit) []string {
	return slices.Clone(unit.Cases)
}

const catcherSource = `/* fairyc Windows x64 unwind probe: catcher and context walker */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>

#define FY_UNWIND_EXCEPTION_CODE ((DWORD)0xE042F10Du)
#define FY_UNWIND_TOKEN UINT64_C(0x26374A5B6C7D8E9F)
#define FY_UNWIND_XOR UINT64_C(0xA5C39E17D42B680F)

extern void fy_unwind_frame(uint64_t token);
extern void fy_unwind_raise(uint64_t token);
static int fy_unwind_catch(void);

volatile LONG fy_unwind_finally_count = 0;
volatile LONG fy_unwind_abnormal_count = 0;
volatile LONG fy_unwind_normal_count = 0;
volatile LONG fy_unwind_failure = 0;

static int fy_contains_pc(DWORD64 pc, DWORD64 function_address) {
    DWORD64 image_base = 0;
    PRUNTIME_FUNCTION entry = RtlLookupFunctionEntry(function_address, &image_base, NULL);
    if (entry == NULL) {
        return 0;
    }
    return pc >= image_base + entry->BeginAddress && pc < image_base + entry->EndAddress;
}

__declspec(noinline) static int fy_walk_exception_context(const CONTEXT *initial) {
    CONTEXT context = *initial;
    int stage = 0;
    unsigned int depth = 0;
    DWORD64 raiser = (DWORD64)(uintptr_t)&fy_unwind_raise;
    DWORD64 frame = (DWORD64)(uintptr_t)&fy_unwind_frame;
    DWORD64 catcher = (DWORD64)(uintptr_t)&fy_unwind_catch;

    for (depth = 0; depth < 64 && context.Rip != 0 && context.Rsp != 0; ++depth) {
        if (stage == 0 && fy_contains_pc(context.Rip, raiser)) {
            stage = 1;
        } else if (stage == 1 && fy_contains_pc(context.Rip, frame)) {
            stage = 2;
        } else if (stage == 2 && fy_contains_pc(context.Rip, catcher)) {
            stage = 3;
        }
        if (stage == 3) {
            return 1;
        }

        {
            DWORD64 image_base = 0;
            PRUNTIME_FUNCTION entry = RtlLookupFunctionEntry(context.Rip, &image_base, NULL);
            if (entry != NULL) {
                PVOID handler_data = NULL;
                DWORD64 establisher_frame = 0;
                (void)RtlVirtualUnwind(
                    UNW_FLAG_NHANDLER,
                    image_base,
                    context.Rip,
                    entry,
                    &context,
                    &handler_data,
                    &establisher_frame,
                    NULL);
            } else {
                context.Rip = *(const DWORD64 *)(uintptr_t)context.Rsp;
                context.Rsp += sizeof(DWORD64);
            }
        }
    }
    return 0;
}

static LONG fy_unwind_filter(EXCEPTION_POINTERS *information) {
    if (information == NULL || information->ExceptionRecord == NULL || information->ContextRecord == NULL) {
        InterlockedCompareExchange(&fy_unwind_failure, 31, 0);
        return EXCEPTION_EXECUTE_HANDLER;
    }
    if (information->ExceptionRecord->ExceptionCode != FY_UNWIND_EXCEPTION_CODE ||
        information->ExceptionRecord->NumberParameters != 2 ||
        information->ExceptionRecord->ExceptionInformation[0] != (ULONG_PTR)FY_UNWIND_TOKEN ||
        information->ExceptionRecord->ExceptionInformation[1] != (ULONG_PTR)(FY_UNWIND_TOKEN ^ FY_UNWIND_XOR)) {
        InterlockedCompareExchange(&fy_unwind_failure, 32, 0);
    }
    if (!fy_walk_exception_context(information->ContextRecord)) {
        InterlockedCompareExchange(&fy_unwind_failure, 33, 0);
    }
    return EXCEPTION_EXECUTE_HANDLER;
}

__declspec(noinline) static int fy_unwind_catch(void) {
    int caught = 0;
    __try {
        fy_unwind_frame(FY_UNWIND_TOKEN);
        InterlockedCompareExchange(&fy_unwind_failure, 34, 0);
    }
    __except (fy_unwind_filter(GetExceptionInformation())) {
        caught = 1;
    }
    if (!caught) {
        return 35;
    }
    if (fy_unwind_failure != 0) {
        return (int)fy_unwind_failure;
    }
    if (fy_unwind_finally_count != 1 || fy_unwind_abnormal_count != 1 || fy_unwind_normal_count != 0) {
        return 36;
    }
    return 0;
}

int main(void) {
    return fy_unwind_catch();
}
`

const frameSource = `/* fairyc Windows x64 unwind probe: non-leaf termination frame */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stddef.h>

#define FY_UNWIND_XOR UINT64_C(0xA5C39E17D42B680F)

extern void fy_unwind_raise(uint64_t token);
extern volatile LONG fy_unwind_finally_count;
extern volatile LONG fy_unwind_abnormal_count;
extern volatile LONG fy_unwind_normal_count;
extern volatile LONG fy_unwind_failure;

__declspec(noinline) void fy_unwind_frame(uint64_t token) {
    volatile uint64_t stack_state[64];
    size_t index = 0;
    for (index = 0; index < 64; ++index) {
        stack_state[index] = token ^ (uint64_t)index;
    }
    __try {
        fy_unwind_raise(token);
        InterlockedCompareExchange(&fy_unwind_failure, 41, 0);
    }
    __finally {
        InterlockedIncrement(&fy_unwind_finally_count);
        if (AbnormalTermination()) {
            InterlockedIncrement(&fy_unwind_abnormal_count);
        } else {
            InterlockedIncrement(&fy_unwind_normal_count);
        }
        if (stack_state[0] != token || stack_state[63] != (token ^ UINT64_C(63)) ||
            (token ^ FY_UNWIND_XOR) == 0) {
            InterlockedCompareExchange(&fy_unwind_failure, 42, 0);
        }
    }
}
`

const raiserSource = `/* fairyc Windows x64 unwind probe: private software exception raiser */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>

#define FY_UNWIND_EXCEPTION_CODE ((DWORD)0xE042F10Du)
#define FY_UNWIND_XOR UINT64_C(0xA5C39E17D42B680F)

__declspec(noinline) void fy_unwind_raise(uint64_t token) {
    ULONG_PTR arguments[2];
    volatile uint64_t guard = token ^ FY_UNWIND_XOR;
    arguments[0] = (ULONG_PTR)token;
    arguments[1] = (ULONG_PTR)guard;
    RaiseException(FY_UNWIND_EXCEPTION_CODE, EXCEPTION_NONCONTINUABLE, 2, arguments);
    ExitProcess(51);
}
`
