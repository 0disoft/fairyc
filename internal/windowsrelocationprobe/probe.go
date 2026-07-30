// Package windowsrelocationprobe generates one closed Microsoft x64 REL32 link proof.
package windowsrelocationprobe

const (
	SourceSymbol         = "fy_rel32_source"
	TargetSymbol         = "fy_rel32_target"
	ReceiptSymbol        = "fy_rel32_receipt"
	SourceObject         = "caller.obj"
	TargetObject         = "target.obj"
	VariantObject        = "variants.obj"
	VariantSource        = "fy_vsrc"
	VariantTarget        = "fy_vdst"
	WeakObject           = "weak.obj"
	OverflowObject       = "overflow.obj"
	OverflowSource       = "fy_ovfl"
	BigObject            = "bigobj.obj"
	BigObjectSource      = "fy_bigob"
	WeakFallbackEntry    = "fy_wfb_e"
	WeakOverrideEntry    = "fy_wov_e"
	WeakMissing          = "fy_wmiss"
	WeakOverride         = "fy_wovr"
	WeakMissingFallback  = "fy_mfbk"
	WeakOverrideFallback = "fy_ofbk"
	ReceiptSection       = ".fyreloc"
	SourceLine           = uint32(11)
	ExpectedExit         = int32(0)
)

var ReceiptMagic = [16]byte{'F', 'Y', 'R', 'E', 'L', '3', '2', '-', 'L', 'I', 'N', 'K', '-', 'V', '0', '!'}

// Unit contains separately compiled caller and target translation units plus the link receipt
// identity consumed by the PE/COFF inspector.
type Unit struct {
	CallerSource         string
	TargetSource         string
	SourceSymbol         string
	TargetSymbol         string
	ReceiptSymbol        string
	SourceObject         string
	TargetObject         string
	VariantObject        string
	VariantSource        string
	VariantTarget        string
	WeakObject           string
	OverflowObject       string
	OverflowSource       string
	BigObject            string
	BigObjectSource      string
	WeakFallbackEntry    string
	WeakOverrideEntry    string
	WeakMissing          string
	WeakOverride         string
	WeakMissingFallback  string
	WeakOverrideFallback string
	ReceiptSection       string
	ReceiptMagic         [16]byte
	SourceLine           uint32
	ExpectedExit         int32
}

// Generate returns deterministic C11 sources. The caller retains one direct cross-object call and
// a read-only image receipt containing preferred-image addresses for both functions.
func Generate() Unit {
	return Unit{
		CallerSource: callerSource, TargetSource: targetSource,
		SourceSymbol: SourceSymbol, TargetSymbol: TargetSymbol,
		ReceiptSymbol: ReceiptSymbol, SourceObject: SourceObject, TargetObject: TargetObject,
		VariantObject: VariantObject, VariantSource: VariantSource, VariantTarget: VariantTarget,
		WeakObject: WeakObject, WeakFallbackEntry: WeakFallbackEntry, WeakOverrideEntry: WeakOverrideEntry,
		OverflowObject: OverflowObject, OverflowSource: OverflowSource,
		BigObject: BigObject, BigObjectSource: BigObjectSource,
		WeakMissing: WeakMissing, WeakOverride: WeakOverride,
		WeakMissingFallback: WeakMissingFallback, WeakOverrideFallback: WeakOverrideFallback,
		ReceiptSection: ReceiptSection, ReceiptMagic: ReceiptMagic, SourceLine: SourceLine, ExpectedExit: ExpectedExit,
	}
}

const callerSource = `/* fairyc Microsoft x64 REL32 link proof: caller and address receipt */
#include <stdint.h>

typedef int (__cdecl *fy_rel32_function)(int);

extern int fy_rel32_target(int value);
extern int fy_vsrc(void);
extern int fy_wfb_e(void);
extern int fy_wov_e(void);

__declspec(noinline) int fy_rel32_source(int value) {
    volatile int observed = fy_rel32_target(value);
    return observed + 1;
}

extern int fy_ovfl(void);
extern int fy_bigob(void);

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
    if (fy_rel32_source(40) != 42) {
        return 12;
    }
    if (fy_vsrc() != 42) {
        return 13;
    }
    if (fy_wfb_e() != 17) {
        return 14;
    }
    if (fy_wov_e() != 29) {
        return 15;
    }
    if (fy_ovfl() != 42) {
        return 16;
    }
    return fy_bigob() == 42 ? 0 : 17;
}
`

const targetSource = `/* fairyc Microsoft x64 REL32 link proof: separate target */
__declspec(noinline) int fy_rel32_target(int value) {
    return value + 1;
}

__declspec(noinline) int fy_vdst(int value) {
    return value + 2;
}

__declspec(noinline) int fy_wovr(void) {
    return 29;
}
`
