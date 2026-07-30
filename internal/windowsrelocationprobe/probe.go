// Package windowsrelocationprobe generates one closed Microsoft x64 REL32 link proof.
package windowsrelocationprobe

const (
	SourceSymbol   = "fy_rel32_source"
	TargetSymbol   = "fy_rel32_target"
	ReceiptSymbol  = "fy_rel32_receipt"
	SourceObject   = "caller.obj"
	TargetObject   = "target.obj"
	ReceiptSection = ".fyreloc"
	ExpectedExit   = int32(0)
)

var ReceiptMagic = [16]byte{'F', 'Y', 'R', 'E', 'L', '3', '2', '-', 'L', 'I', 'N', 'K', '-', 'V', '0', '!'}

// Unit contains separately compiled caller and target translation units plus the link receipt
// identity consumed by the PE/COFF inspector.
type Unit struct {
	CallerSource   string
	TargetSource   string
	SourceSymbol   string
	TargetSymbol   string
	ReceiptSymbol  string
	SourceObject   string
	TargetObject   string
	ReceiptSection string
	ReceiptMagic   [16]byte
	ExpectedExit   int32
}

// Generate returns deterministic C11 sources. The caller retains one direct cross-object call and
// a read-only image receipt containing preferred-image addresses for both functions.
func Generate() Unit {
	return Unit{
		CallerSource: callerSource, TargetSource: targetSource,
		SourceSymbol: SourceSymbol, TargetSymbol: TargetSymbol,
		ReceiptSymbol: ReceiptSymbol, SourceObject: SourceObject, TargetObject: TargetObject,
		ReceiptSection: ReceiptSection, ReceiptMagic: ReceiptMagic, ExpectedExit: ExpectedExit,
	}
}

const callerSource = `/* fairyc Microsoft x64 REL32 link proof: caller and address receipt */
#include <stdint.h>

typedef int (__cdecl *fy_rel32_function)(int);

extern int fy_rel32_target(int value);

__declspec(noinline) int fy_rel32_source(int value) {
    volatile int observed = fy_rel32_target(value);
    return observed + 1;
}

struct fy_rel32_link_receipt {
    unsigned char magic[16];
    fy_rel32_function source;
    fy_rel32_function target;
};

#pragma section(".fyreloc", read)
__declspec(allocate(".fyreloc")) const struct fy_rel32_link_receipt fy_rel32_receipt = {
    { 'F', 'Y', 'R', 'E', 'L', '3', '2', '-', 'L', 'I', 'N', 'K', '-', 'V', '0', '!' },
    fy_rel32_source,
    fy_rel32_target
};

int main(void) {
    if (fy_rel32_receipt.source != fy_rel32_source || fy_rel32_receipt.target != fy_rel32_target ||
        fy_rel32_receipt.magic[0] != 'F' || fy_rel32_receipt.magic[15] != '!') {
        return 11;
    }
    return fy_rel32_source(40) == 42 ? 0 : 12;
}
`

const targetSource = `/* fairyc Microsoft x64 REL32 link proof: separate target */
__declspec(noinline) int fy_rel32_target(int value) {
    return value + 1;
}
`
