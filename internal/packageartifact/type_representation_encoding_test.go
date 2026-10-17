package packageartifact

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/mircodec"
)

// The golden bytes were measured from b22396e6, before Type became an interned
// handle. Changing an in-memory representation must not change artifact schemas.
func TestTypeRepresentationPreservesArtifactEncoding(t *testing.T) {
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	encoded, err := mircodec.Encode(program, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		bytes int
		hash  string
	}{
		"whole-mir": {7441, "b13bc264b1ebf6a101d36ce7cc3c5a64a43972b15236303e57569c6ea817bfeb"},
		"package-1": {2772, "bf1ed839cca485af8e6f7903b21b7165103d30be302b80d324419817554c9eac"},
		"package-2": {6088, "7cb67cc6bf1506314c073f6f63e9de7a9ddfdf173b9097d76f8e935362b44ee0"},
	}
	check := func(name string, data []byte) {
		expected, ok := want[name]
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		if !ok || expected.bytes != len(data) || expected.hash != hash {
			t.Errorf("%s encoded %d bytes, SHA-256 %s; want %+v", name, len(data), hash, expected)
		}
		delete(want, name)
	}
	check("whole-mir", encoded)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		check(fmt.Sprintf("package-%d", artifact.PackageID), encoded)
	}
	if len(want) != 0 {
		t.Fatalf("missing artifact encodings: %+v", want)
	}
}
