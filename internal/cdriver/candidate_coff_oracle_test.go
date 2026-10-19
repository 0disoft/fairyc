package cdriver

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCandidateCOFFBuildObjects(t *testing.T) {
	directory := os.Getenv("FAIRYC_COFF_OBJECTS")
	if directory == "" {
		t.Skip("explicit authored COFF real objects only")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"empty":    "/* include-only unit */\n#include <stdint.h>\n",
		"literals": "#include <stdint.h>\nconst uint8_t sample_literal[]={1,2,3};\n",
		"code":     "#include <stdlib.h>\n__declspec(selectany) int selected=42;\nint sample(int x){void *p=malloc(16);free(p);return selected+x;}\n",
	} {
		path := filepath.Join(directory, name+".c")
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		for _, profile := range []string{"Od", "O2"} {
			for _, big := range []bool{false, true} {
				mode := "regular"
				if big {
					mode = "big"
				}
				object := filepath.Join(directory, name+"-"+profile+"-"+mode+".obj")
				args := []string{"/nologo", "/TC", "/std:c11", "/MT", "/Brepro", "/" + profile, "/Gy", "/Gw", "/c", path, "/Fo:" + object}
				if big {
					args = append(args, "/bigobj")
				}
				if _, err := runWindowsHostCommandWithLimitAndTimeout(toolchain.Path, toolchain.Environment, directory, maxProcessOutput, commandTimeout, args...); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(object)
				if err != nil {
					t.Fatal(err)
				}
				// Include-only ordinary objects have no code and use the data-only
				// fallback. Bigobj data-only files are intentionally rejected by Go.
				if !big {
					if err := inspectProjectObject(data); err != nil {
						t.Fatal(name, profile, err)
					}
				}
			}
		}
	}
}

func TestCandidateCOFFProjectOracle(t *testing.T) {
	path := os.Getenv("FAIRYC_COFF_ORACLE")
	if path == "" {
		t.Skip("explicit project COFF oracle only")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	input := bytes.NewReader(data)
	magic := make([]byte, len("FYCCOFFVERIFY1\n"))
	if _, err := io.ReadFull(input, magic); err != nil || string(magic) != "FYCCOFFVERIFY1\n" {
		t.Fatal("COFF fixture magic", err)
	}
	read := func() uint32 {
		var n uint32
		if err := binary.Read(input, binary.LittleEndian, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	part := func() []byte {
		n := read()
		if n > 64*1024*1024 {
			t.Fatal("unbounded COFF fixture")
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(input, b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	count := read()
	accepted := 0
	for i := uint32(0); i < count; i++ {
		_ = part()
		object := part()
		_ = part()
		offset := len(data) - input.Len()
		_ = read()
		if inspectProjectObject(object) == nil {
			binary.LittleEndian.PutUint32(data[offset:], 1)
			accepted++
		}
	}
	if input.Len() != 0 {
		t.Fatal("COFF fixture trailing data")
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("project driver admits %d of %d COFF inputs", accepted, count)
}
