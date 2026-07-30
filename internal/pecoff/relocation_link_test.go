package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testREL32ObjectRaw     = coffFileHeaderBytes + coffSectionHeaderBytes
	testREL32ObjectReloc   = testREL32ObjectRaw + 5
	testREL32ObjectSymbols = testREL32ObjectReloc + coffRelocationBytes
	testREL32ImageRaw      = 0x200
	testREL32ReceiptRaw    = 0x400
	testREL32SourceRVA     = 0x1000
	testREL32TargetRVA     = 0x1020
	testREL32LinkedSiteRVA = testREL32SourceRVA + 1
	testREL32LinkedDisp    = testREL32TargetRVA - (testREL32LinkedSiteRVA + 4)
)

var testREL32Magic = [16]byte{'F', 'Y', 'R', 'E', 'L', '3', '2', '-', 'L', 'I', 'N', 'K', '-', 'V', '0', '!'}

func TestInspectAMD64REL32LinkProvesObjectFormulaAndLinkedDisplacement(t *testing.T) {
	t.Parallel()
	report, err := InspectAMD64REL32Link(testREL32Object(), testREL32Image(), testREL32Spec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ObjectSection != ".text" || report.ObjectVirtualAddress != 1 || report.ObjectAddend != 0 ||
		report.SourceRVA != testREL32SourceRVA || report.TargetRVA != testREL32TargetRVA ||
		report.LinkedSiteRVA != testREL32LinkedSiteRVA || report.LinkedDisplacement != testREL32LinkedDisp ||
		report.ExpectedDisplacement != testREL32LinkedDisp {
		t.Fatalf("InspectAMD64REL32Link() report = %+v", report)
	}
}

func TestInspectAMD64REL32LinkRejectsIndependentObjectAndImageMutations(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name       string
		editObject func([]byte)
		editImage  func([]byte)
		want       string
	}{
		{name: "wrong relocation type", editObject: func(object []byte) {
			binary.LittleEndian.PutUint16(object[testREL32ObjectReloc+8:], ObjectRelocationAMD64Addr32)
		}, want: "no exact source-to-target relocation"},
		{name: "wrong relocation target", editObject: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testREL32ObjectReloc+4:], 0)
		}, want: "no exact source-to-target relocation"},
		{name: "source is not a function", editObject: func(object []byte) {
			binary.LittleEndian.PutUint16(object[testREL32ObjectSymbols+14:], 0)
		}, want: "source symbol"},
		{name: "target is not external", editObject: func(object []byte) {
			object[testREL32ObjectSymbols+18+16] = coffStorageClassStatic
		}, want: "target symbol"},
		{name: "non-call object opcode", editObject: func(object []byte) {
			object[testREL32ObjectRaw] = 0xe9
		}, want: "not a direct CALL"},
		{name: "object addend drift", editObject: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testREL32ObjectRaw+1:], 1)
		}, want: "linked displacement"},
		{name: "linked displacement drift", editImage: func(image []byte) {
			binary.LittleEndian.PutUint32(image[testREL32ImageRaw+1:], testREL32LinkedDisp+1)
		}, want: "linked displacement"},
		{name: "non-call linked opcode", editImage: func(image []byte) {
			image[testREL32ImageRaw] = 0xe9
		}, want: "not a direct CALL"},
		{name: "target receipt drift", editImage: func(image []byte) {
			binary.LittleEndian.PutUint64(image[testREL32ReceiptRaw+24:], uint64(0x140000000+testREL32TargetRVA+1))
		}, want: "linked displacement"},
		{name: "executable receipt section", editImage: func(image []byte) {
			binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+coffSectionHeaderBytes+36:], sectionContainsCode|sectionExecutable|sectionReadable)
		}, want: "not read-only data"},
		{name: "source outside image", editImage: func(image []byte) {
			binary.LittleEndian.PutUint64(image[testREL32ReceiptRaw+16:], 1)
		}, want: "precedes image base"},
		{name: "ambiguous receipt", editImage: func(image []byte) {
			copy(image[testREL32ReceiptRaw+0x40:], testREL32Magic[:])
		}, want: "receipt magic is ambiguous"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object := testREL32Object()
			image := testREL32Image()
			if testCase.editObject != nil {
				testCase.editObject(object)
			}
			if testCase.editImage != nil {
				testCase.editImage(image)
			}
			if _, err := InspectAMD64REL32Link(object, image, testREL32Spec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64REL32Link() error = %v", err)
			}
		})
	}
}

func TestInspectAMD64REL32LinkAcceptsNegativeAddendAndBackwardDisplacement(t *testing.T) {
	t.Parallel()
	addend := int32(-7)
	object := testREL32Object()
	binary.LittleEndian.PutUint32(object[testREL32ObjectRaw+1:], uint32(addend))
	image, sourceRVA, targetRVA, displacement := testREL32BackwardImage(addend)
	report, err := InspectAMD64REL32Link(object, image, testREL32Spec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ObjectAddend != addend || report.SourceRVA != sourceRVA || report.TargetRVA != targetRVA ||
		report.LinkedDisplacement != displacement || report.ExpectedDisplacement != displacement || displacement >= 0 {
		t.Fatalf("InspectAMD64REL32Link() backward report = %+v", report)
	}
}

func TestInspectAMD64REL32LinkIgnoresReceiptMagicInRawAlignmentPadding(t *testing.T) {
	t.Parallel()
	image := testREL32Image()
	copy(image[testREL32ReceiptRaw+0x80:], testREL32Magic[:])
	if _, err := InspectAMD64REL32Link(testREL32Object(), image, testREL32Spec()); err != nil {
		t.Fatalf("InspectAMD64REL32Link() padding error = %v", err)
	}
}

func TestAMD64REL32DisplacementSignedBoundaries(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name   string
		target uint32
		addend int32
		site   uint32
		want   int32
	}{
		{name: "positive maximum", target: 0x80000003, site: 0, want: 1<<31 - 1},
		{name: "negative minimum", target: 0, site: 0x7ffffffc, want: -1 << 31},
		{name: "negative addend", target: 0x1000, addend: -7, site: 0x1041, want: -76},
	} {
		displacement, err := amd64REL32Displacement(testCase.target, testCase.addend, testCase.site)
		if err != nil || displacement != testCase.want {
			t.Fatalf("%s: amd64REL32Displacement() = %d, %v; want %d", testCase.name, displacement, err, testCase.want)
		}
	}
	for _, testCase := range []struct {
		target uint32
		site   uint32
	}{
		{target: 0x80000004, site: 0},
		{target: 0, site: 0x7ffffffd},
	} {
		if _, err := amd64REL32Displacement(testCase.target, 0, testCase.site); err == nil || !strings.Contains(err.Error(), "out of range") {
			t.Fatalf("amd64REL32Displacement(%#x, %#x) error = %v", testCase.target, testCase.site, err)
		}
	}
}

func TestRequireREL32SourceSectionRejectsDataEvenWhenAnotherTextSectionCouldExist(t *testing.T) {
	t.Parallel()
	if err := requireREL32SourceSection(ObjectSection{Name: ".data", Characteristics: sectionReadable}); err == nil || !strings.Contains(err.Error(), "executable .text code") {
		t.Fatalf("requireREL32SourceSection() error = %v", err)
	}
}

func TestInspectAMD64REL32LinkRejectsInvalidSpec(t *testing.T) {
	t.Parallel()
	for _, spec := range []AMD64REL32LinkSpec{
		{},
		{SourceSymbol: "same", TargetSymbol: "same", ReceiptSection: ".text", ReceiptMagic: testREL32Magic},
		{SourceSymbol: "fy_src", TargetSymbol: "fy_dst", ReceiptSection: ".text"},
	} {
		if _, err := InspectAMD64REL32Link(testREL32Object(), testREL32Image(), spec); err == nil {
			t.Fatalf("InspectAMD64REL32Link() accepted spec %+v", spec)
		}
	}
}

func TestExactREL32RelocationScopesDuplicatesToTheSourceSymbolRange(t *testing.T) {
	t.Parallel()
	relocations := []ObjectRelocation{
		{Section: ".text", VirtualAddress: 4, SymbolIndex: 7, Type: ObjectRelocationAMD64Rel32},
		{Section: ".text", VirtualAddress: 24, SymbolIndex: 7, Type: ObjectRelocationAMD64Rel32},
	}
	relocation, err := exactREL32Relocation(relocations, ".text", 7, 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	if relocation.VirtualAddress != 4 {
		t.Fatalf("exactREL32Relocation() = %+v", relocation)
	}
	if _, err := exactREL32Relocation(relocations, ".text", 7, 0, 32); err == nil || !strings.Contains(err.Error(), "repeats") {
		t.Fatalf("exactREL32Relocation() duplicate error = %v", err)
	}
}

func testREL32Spec() AMD64REL32LinkSpec {
	return AMD64REL32LinkSpec{
		SourceSymbol: "fy_src", TargetSymbol: "fy_dst", ReceiptSection: ".fyreloc", ReceiptMagic: testREL32Magic,
	}
}

func testREL32Object() []byte {
	object := make([]byte, testREL32ObjectSymbols+2*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], testREL32ObjectSymbols)
	binary.LittleEndian.PutUint32(object[12:], 2)

	section := coffFileHeaderBytes
	copy(object[section:], ".text")
	binary.LittleEndian.PutUint32(object[section+16:], 5)
	binary.LittleEndian.PutUint32(object[section+20:], testREL32ObjectRaw)
	binary.LittleEndian.PutUint32(object[section+24:], testREL32ObjectReloc)
	binary.LittleEndian.PutUint16(object[section+32:], 1)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[testREL32ObjectRaw] = 0xe8

	binary.LittleEndian.PutUint32(object[testREL32ObjectReloc:], 1)
	binary.LittleEndian.PutUint32(object[testREL32ObjectReloc+4:], 1)
	binary.LittleEndian.PutUint16(object[testREL32ObjectReloc+8:], ObjectRelocationAMD64Rel32)

	copy(object[testREL32ObjectSymbols:], "fy_src")
	binary.LittleEndian.PutUint16(object[testREL32ObjectSymbols+12:], 1)
	binary.LittleEndian.PutUint16(object[testREL32ObjectSymbols+14:], 0x20)
	object[testREL32ObjectSymbols+16] = 2
	target := testREL32ObjectSymbols + 18
	copy(object[target:], "fy_dst")
	binary.LittleEndian.PutUint16(object[target+14:], 0x20)
	object[target+16] = 2
	binary.LittleEndian.PutUint32(object[target+18:], 4)
	return object
}

func testREL32Image() []byte {
	base := minimalAMD64PE()
	image := make([]byte, 0x600)
	copy(image, base)
	coff := testPEOffset + 4
	binary.LittleEndian.PutUint16(image[coff+2:], 2)
	binary.LittleEndian.PutUint32(image[testOptionalHeaderOffset+56:], 0x3000)
	receiptSection := testSectionHeaderOffset + coffSectionHeaderBytes
	copy(image[receiptSection:], ".fyreloc")
	binary.LittleEndian.PutUint32(image[receiptSection+8:], 0x80)
	binary.LittleEndian.PutUint32(image[receiptSection+12:], 0x2000)
	binary.LittleEndian.PutUint32(image[receiptSection+16:], 0x200)
	binary.LittleEndian.PutUint32(image[receiptSection+20:], testREL32ReceiptRaw)
	binary.LittleEndian.PutUint32(image[receiptSection+36:], 0x00000040|sectionReadable)
	image[testREL32ImageRaw] = 0xe8
	binary.LittleEndian.PutUint32(image[testREL32ImageRaw+1:], testREL32LinkedDisp)
	image[testREL32ImageRaw+0x20] = 0xc3
	copy(image[testREL32ReceiptRaw:], testREL32Magic[:])
	binary.LittleEndian.PutUint64(image[testREL32ReceiptRaw+16:], uint64(0x140000000+testREL32SourceRVA))
	binary.LittleEndian.PutUint64(image[testREL32ReceiptRaw+24:], uint64(0x140000000+testREL32TargetRVA))
	return image
}

func testREL32BackwardImage(addend int32) ([]byte, uint32, uint32, int32) {
	const sourceRVA = uint32(0x1040)
	const targetRVA = uint32(0x1000)
	image := testREL32Image()
	for index := 0; index < 5; index++ {
		image[testREL32ImageRaw+index] = 0
	}
	sourceRaw := testREL32ImageRaw + int(sourceRVA-testREL32SourceRVA)
	image[sourceRaw] = 0xe8
	displacement, err := amd64REL32Displacement(targetRVA, addend, sourceRVA+1)
	if err != nil {
		panic(err)
	}
	binary.LittleEndian.PutUint32(image[sourceRaw+1:], uint32(displacement))
	image[testREL32ImageRaw] = 0xc3
	binary.LittleEndian.PutUint64(image[testREL32ReceiptRaw+16:], uint64(0x140000000)+uint64(sourceRVA))
	binary.LittleEndian.PutUint64(image[testREL32ReceiptRaw+24:], uint64(0x140000000)+uint64(targetRVA))
	return image, sourceRVA, targetRVA, displacement
}
