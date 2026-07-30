package pecoff

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectAMD64WeakSearchLinkProvesArchiveSuppressionAndExtraction(t *testing.T) {
	t.Parallel()
	report, err := InspectAMD64WeakSearchLink(testWeakSearchObject(), testWeakSearchArchive(), testWeakSearchImage(), testWeakSearchMap(), testWeakSearchSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ArchiveMembers != 2 || !report.NoLibrary.UsedFallback || report.NoLibrary.SelectedSymbol != "fy_nl_f" ||
		report.Library.UsedFallback || report.Library.SelectedSymbol != "fy_lib_w" ||
		report.NoLibrary.LinkedDisplacement != report.NoLibrary.ExpectedDisplacement ||
		report.Library.LinkedDisplacement != report.Library.ExpectedDisplacement {
		t.Fatalf("weak-search report = %+v", report)
	}
}

func TestInspectAMD64WeakSearchLinkRejectsIndependentMutations(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "no-library search becomes alias", edit: func(object, _, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testWeakSymbols+3*18+4:], ObjectWeakSearchAlias)
		}, want: "invalid search metadata"},
		{name: "library search becomes alias", edit: func(object, _, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testWeakSymbols+5*18+4:], ObjectWeakSearchAlias)
		}, want: "invalid search metadata"},
		{name: "no-library member extracted", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = append(*linkMap, []byte(" 0001:00000120 fy_nl_s 0000000140001120 f weakmode.lib:nolib.obj\r\n")...)
		}, want: "unexpectedly extracted archive sentinel"},
		{name: "library owner becomes direct", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = bytes.ReplaceAll(*linkMap, []byte("weakmode.lib:library.obj"), []byte("library.obj             "))
		}, want: "selected target owner"},
		{name: "linked library displacement", edit: func(_, _, image, _ *[]byte) {
			(*image)[testWeakSourceRaw+6]++
		}, want: "linked weak displacement"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object := testWeakSearchObject()
			archive := testWeakSearchArchive()
			image := testWeakSearchImage()
			linkMap := testWeakSearchMap()
			testCase.edit(&object, &archive, &image, &linkMap)
			if _, err := InspectAMD64WeakSearchLink(object, archive, image, linkMap, testWeakSearchSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64WeakSearchLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testWeakSearchObject() []byte {
	object, err := BuildAMD64WeakSearchProbeObject("fy_nl_e", "fy_lib_e", "fy_nl_w", "fy_lib_w", "fy_nl_f", "fy_lib_f")
	if err != nil {
		panic(err)
	}
	return object
}

func testWeakSearchImage() []byte {
	image := testREL32Image()
	object := testWeakSearchObject()
	copy(image[testWeakSourceRaw:testWeakSourceRaw+22], object[testWeakRawOffset:testWeakRawOffset+22])
	for _, value := range []struct {
		site   uint32
		target uint32
	}{{site: 1, target: testWeakSourceRVA + 10}, {site: 6, target: testVariantTargetRVA}} {
		displacement, err := amd64REL32Displacement(value.target, 0, testWeakSourceRVA+value.site)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint32(image[testWeakSourceRaw+int(value.site):], uint32(displacement))
	}
	return image
}

func testWeakSearchMap() []byte {
	return []byte(" 0001:00000080 fy_nl_e 0000000140001080 f weakmode.obj\r\n" +
		" 0001:00000085 fy_lib_e 0000000140001085 f weakmode.obj\r\n" +
		" 0001:0000008a fy_nl_f 000000014000108a f weakmode.obj\r\n" +
		" 0001:00000100 fy_lib_w 0000000140001100 f weakmode.lib:library.obj\r\n")
}

func testWeakSearchSpec() AMD64WeakSearchSpec {
	return AMD64WeakSearchSpec{
		ObjectName: "weakmode.obj", ArchiveName: "weakmode.lib", NoLibraryMember: "nolib.obj", LibraryMember: "library.obj",
		NoLibraryEntry: "fy_nl_e", LibraryEntry: "fy_lib_e", NoLibraryWeak: "fy_nl_w", NoLibrarySentinel: "fy_nl_s", LibraryWeak: "fy_lib_w",
		NoLibraryFallback: "fy_nl_f", LibraryFallback: "fy_lib_f",
	}
}

func testWeakSearchArchive() []byte {
	noLibrary := testWeakSearchArchiveObject("fy_nl_w", "fy_nl_s")
	library := testWeakSearchArchiveObject("fy_lib_w")
	names := []string{"fy_lib_w", "fy_nl_s", "fy_nl_w"}
	firstSize := 4 + len(names)*4
	secondSize := 4 + 2*4 + 4 + len(names)*2
	for _, name := range names {
		firstSize += len(name) + 1
		secondSize += len(name) + 1
	}
	firstHeader := len(coffArchiveSignature)
	secondHeader := nextSyntheticArchiveMemberOffset(firstHeader, firstSize)
	noLibraryHeader := nextSyntheticArchiveMemberOffset(secondHeader, secondSize)
	libraryHeader := nextSyntheticArchiveMemberOffset(noLibraryHeader, len(noLibrary))

	first := make([]byte, firstSize)
	binary.BigEndian.PutUint32(first, uint32(len(names)))
	position := 4
	for _, offset := range []int{libraryHeader, noLibraryHeader, noLibraryHeader} {
		binary.BigEndian.PutUint32(first[position:], uint32(offset))
		position += 4
	}
	for _, name := range names {
		copy(first[position:], name)
		position += len(name) + 1
	}
	second := make([]byte, secondSize)
	binary.LittleEndian.PutUint32(second, 2)
	binary.LittleEndian.PutUint32(second[4:], uint32(noLibraryHeader))
	binary.LittleEndian.PutUint32(second[8:], uint32(libraryHeader))
	binary.LittleEndian.PutUint32(second[12:], uint32(len(names)))
	position = 16
	for _, memberIndex := range []uint16{2, 1, 1} {
		binary.LittleEndian.PutUint16(second[position:], memberIndex)
		position += 2
	}
	for _, name := range names {
		copy(second[position:], name)
		position += len(name) + 1
	}
	var archive bytes.Buffer
	archive.Write(coffArchiveSignature[:])
	writeSyntheticArchiveMember(&archive, "/", first)
	writeSyntheticArchiveMember(&archive, "/", second)
	writeSyntheticArchiveMember(&archive, "nolib.obj/", noLibrary)
	writeSyntheticArchiveMember(&archive, "library.obj/", library)
	return archive.Bytes()
}

func testWeakSearchArchiveObject(names ...string) []byte {
	const rawOffset = coffFileHeaderBytes + coffSectionHeaderBytes
	rawSize := len(names)
	symbolOffset := rawOffset + rawSize
	objectSize := symbolOffset + len(names)*18 + 4
	if objectSize < 128 {
		// debug/pe reads a 96-byte discriminator block before selecting raw COFF parsing.
		objectSize = 128
	}
	object := make([]byte, objectSize)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], uint32(len(names)))
	copy(object[coffFileHeaderBytes:], ".text")
	binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+16:], uint32(rawSize))
	binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+20:], rawOffset)
	binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	for index, name := range names {
		object[rawOffset+index] = 0xc3
		symbol := symbolOffset + index*18
		copy(object[symbol:], name)
		binary.LittleEndian.PutUint32(object[symbol+8:], uint32(index))
		binary.LittleEndian.PutUint16(object[symbol+12:], 1)
		binary.LittleEndian.PutUint16(object[symbol+14:], coffSymbolTypeFunction)
		object[symbol+16] = coffStorageClassExternal
	}
	binary.LittleEndian.PutUint32(object[symbolOffset+len(names)*18:], 4)
	return object
}
