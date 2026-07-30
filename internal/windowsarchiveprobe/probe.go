// Package windowsarchiveprobe generates one closed Microsoft archive-extraction and COMDAT proof.
package windowsarchiveprobe

const (
	ArchiveName      = "choice.lib"
	SelectedMember   = "chosen.obj"
	UnusedMember     = "unused.obj"
	DirectObject     = "direct.obj"
	DemandSymbol     = "fy_archive_target"
	UnusedSymbol     = "fy_archive_unused"
	COMDATSymbol     = "fy_pick_any_value"
	ReceiptSection   = ".fysel"
	ArchiveCandidate = int32(17)
	DirectCandidate  = int32(23)
	ExpectedExit     = int32(0)
)

var ReceiptMagic = [16]byte{'F', 'Y', 'A', 'R', '-', 'C', 'O', 'M', 'D', 'A', 'T', '-', 'V', '0', '0', '!'}

// Unit contains the four separately compiled C11 translation units and exact proof identities.
type Unit struct {
	CallerSource     string
	ChosenSource     string
	UnusedSource     string
	DirectSource     string
	ArchiveName      string
	SelectedMember   string
	UnusedMember     string
	DirectObject     string
	DemandSymbol     string
	UnusedSymbol     string
	COMDATSymbol     string
	ReceiptSection   string
	ReceiptMagic     [16]byte
	ArchiveCandidate int32
	DirectCandidate  int32
	ExpectedExit     int32
}

// Generate returns deterministic C11 inputs. choice.lib contains chosen.obj and unused.obj;
// direct.obj contributes the competing pick-any read-only COMDAT.
func Generate() Unit {
	return Unit{
		CallerSource: callerSource, ChosenSource: chosenSource, UnusedSource: unusedSource,
		DirectSource: directSource, ArchiveName: ArchiveName, SelectedMember: SelectedMember,
		UnusedMember: UnusedMember, DirectObject: DirectObject, DemandSymbol: DemandSymbol,
		UnusedSymbol: UnusedSymbol, COMDATSymbol: COMDATSymbol, ReceiptSection: ReceiptSection,
		ReceiptMagic: ReceiptMagic, ArchiveCandidate: ArchiveCandidate,
		DirectCandidate: DirectCandidate, ExpectedExit: ExpectedExit,
	}
}

const callerSource = `/* fairyc Microsoft archive and COMDAT proof: caller and final-image receipt */
#include <stdint.h>

typedef int (__cdecl *fy_archive_function)(int);

extern int fy_archive_target(int value);
extern const int fy_pick_any_value;

struct fy_archive_selection_receipt {
    unsigned char magic[16];
    fy_archive_function target;
    const int *selected_value;
};

#pragma section(".fysel", read)
__declspec(allocate(".fysel")) const struct fy_archive_selection_receipt fy_archive_receipt = {
    { 'F', 'Y', 'A', 'R', '-', 'C', 'O', 'M', 'D', 'A', 'T', '-', 'V', '0', '0', '!' },
    fy_archive_target,
    &fy_pick_any_value
};

int main(void) {
    int selected;
    if (fy_archive_receipt.target != fy_archive_target ||
        fy_archive_receipt.selected_value != &fy_pick_any_value ||
        fy_archive_receipt.magic[0] != 'F' || fy_archive_receipt.magic[15] != '!') {
        return 11;
    }
    selected = *fy_archive_receipt.selected_value;
    if (selected != 17 && selected != 23) {
        return 12;
    }
    return fy_archive_receipt.target(40) == 41 ? 0 : 13;
}
`

const chosenSource = `/* fairyc Microsoft archive and COMDAT proof: demanded archive member */
extern const __declspec(selectany) int fy_pick_any_value = 17;

__declspec(noinline) int fy_archive_target(int value) {
    return value + 1;
}
`

const unusedSource = `/* fairyc Microsoft archive and COMDAT proof: indexed but unrequested member */
__declspec(noinline) int fy_archive_unused(int value) {
    return value - 1;
}
`

const directSource = `/* fairyc Microsoft archive and COMDAT proof: direct competing COMDAT */
extern const __declspec(selectany) int fy_pick_any_value = 23;

__declspec(noinline) int fy_direct_anchor(void) {
    return fy_pick_any_value;
}
`
