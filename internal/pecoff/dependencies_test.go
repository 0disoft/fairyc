package pecoff

import (
	"encoding/binary"
	"testing"
)

func dependencyTestImage() []byte {
	image := minimalAMD64PE()
	binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+8:], 0x200)
	dir := testOptionalHeaderOffset + 112 + 8
	binary.LittleEndian.PutUint32(image[dir:], 0x1080)
	binary.LittleEndian.PutUint32(image[dir+4:], 40)
	binary.LittleEndian.PutUint32(image[0x280:], 0x10b0)
	binary.LittleEndian.PutUint32(image[0x28c:], 0x10e0)
	binary.LittleEndian.PutUint32(image[0x290:], 0x10c8)
	binary.LittleEndian.PutUint64(image[0x2b0:], 0x10f0)
	binary.LittleEndian.PutUint64(image[0x2b8:], uint64(1)<<63|7)
	binary.LittleEndian.PutUint64(image[0x2c8:], 0x10f0)
	binary.LittleEndian.PutUint64(image[0x2d0:], uint64(1)<<63|7)
	copy(image[0x2e0:], "KERNEL32.dll\x00")
	copy(image[0x2f2:], "ExitProcess\x00")
	return image
}

func TestInspectAMD64Dependencies(t *testing.T) {
	r, err := InspectAMD64Dependencies(dependencyTestImage())
	if err != nil || len(r.Libraries) != 1 || r.Libraries[0].Name != "kernel32.dll" || len(r.Libraries[0].Functions) != 2 || r.Libraries[0].Functions[0].Name != "ExitProcess" || r.Libraries[0].Functions[1].Ordinal != 7 {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := InspectAMD64Dependencies(minimalAMD64PE()); err != nil {
		t.Fatal(err)
	}
	image := dependencyTestImage()
	binary.LittleEndian.PutUint32(image[0x280:], 0)
	if _, err := InspectAMD64Dependencies(image); err != nil {
		t.Fatal("missing OFT fallback:", err)
	}
}

func TestInspectAMD64DependenciesRejectsMutations(t *testing.T) {
	for name, edit := range map[string]func([]byte){
		"name escapes":             func(b []byte) { copy(b[0x2e0:], "../bad.dll\x00") },
		"name empty":               func(b []byte) { b[0x2e0] = 0 },
		"name RVA":                 func(b []byte) { binary.LittleEndian.PutUint32(b[0x28c:], 0xfffffffe) },
		"lookup RVA":               func(b []byte) { binary.LittleEndian.PutUint32(b[0x280:], 0xfffffffc) },
		"IAT RVA":                  func(b []byte) { binary.LittleEndian.PutUint32(b[0x290:], 0xfffffffc) },
		"name overflow":            func(b []byte) { binary.LittleEndian.PutUint64(b[0x2b0:], 0x100000000) },
		"ordinal bits":             func(b []byte) { binary.LittleEndian.PutUint64(b[0x2b8:], uint64(1)<<63|0x10001) },
		"ordinal zero":             func(b []byte) { binary.LittleEndian.PutUint64(b[0x2b8:], uint64(1)<<63) },
		"bound stamp":              func(b []byte) { binary.LittleEndian.PutUint32(b[0x284:], 1) },
		"delay directory":          func(b []byte) { binary.LittleEndian.PutUint32(b[testOptionalHeaderOffset+112+13*8:], 0x1080) },
		"bound directory":          func(b []byte) { binary.LittleEndian.PutUint32(b[testOptionalHeaderOffset+112+11*8:], 0x1080) },
		"no descriptor terminator": func(b []byte) { binary.LittleEndian.PutUint32(b[testOptionalHeaderOffset+112+12:], 20) },
		"IAT terminator mismatch":  func(b []byte) { binary.LittleEndian.PutUint64(b[0x2d8:], 1) },
		"empty thunk": func(b []byte) {
			binary.LittleEndian.PutUint64(b[0x2b0:], 0)
			binary.LittleEndian.PutUint64(b[0x2c8:], 0)
		},
	} {
		t.Run(name, func(t *testing.T) {
			image := dependencyTestImage()
			edit(image)
			if _, err := InspectAMD64Dependencies(image); err == nil {
				t.Fatal("mutation accepted")
			}
		})
	}
}
