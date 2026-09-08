package sdkstdlib

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVectorCandidateIsExplicitAndDetached(t *testing.T) {
	stable, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := LoadVectorCandidate()
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Sources) != len(stable.Sources)+1 || len(candidate.Modules) != len(stable.Modules)+1 ||
		candidate.Digest == stable.Digest || candidate.SourceIdentity == stable.SourceIdentity ||
		!strings.HasPrefix(candidate.SourceIdentity, "fairyc-sdk-vector-candidate/0;") {
		t.Fatal("candidate bundle is not isolated")
	}
	for i, file := range stable.Sources {
		if file.Module.Path == "std::vec" || file.Path != candidate.Sources[i].Path || !bytes.Equal(file.Bytes, candidate.Sources[i].Bytes) {
			t.Fatal("stable source changed")
		}
	}
	last := candidate.Sources[len(candidate.Sources)-1]
	if last.Path != "std/vec.fy" || last.Module.Path != "std::vec" {
		t.Fatal("candidate source missing")
	}
	want, _ := json.Marshal(candidate)
	last.Bytes[0] ^= 1
	last.Module.Path = "changed"
	candidate.Sources[0].Bytes[0] ^= 1
	repeated, err := LoadVectorCandidate()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(repeated)
	if !bytes.Equal(want, got) {
		t.Fatal("candidate loading is not deterministic and detached")
	}
	stableAgain, err := Load()
	if err != nil || stableAgain.SourceIdentity != stable.SourceIdentity {
		t.Fatal("candidate modified installed SDK identity")
	}
}
