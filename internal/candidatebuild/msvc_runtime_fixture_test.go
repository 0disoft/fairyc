package candidatebuild

import (
	"os"
	"testing"
)

// Only verification setup selects runtime C with the Go reference. The authored
// builder receives immutable C bytes and never invokes this test or a Go helper.
func TestPacketMSVCRuntimeFixture(t *testing.T) {
	path := os.Getenv("FAIRYC_MSVC_BUILD_FIXTURE")
	if path == "" {
		t.Skip("explicit authored MSVC build fixture only")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	support, err := runtimeSource(p.Header, false)
	if err != nil {
		t.Fatal(err)
	}
	for suffix, data := range map[string][]byte{
		".runtime.c": []byte(support), ".compile-error": packetFixture("compile-error"), ".link-error": packetFixture("link-error"),
	} {
		if err := os.WriteFile(path+suffix, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
