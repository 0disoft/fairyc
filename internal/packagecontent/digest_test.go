package packagecontent

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestEncodeMatchesCanonicalV1Vector(t *testing.T) {
	t.Parallel()
	files := []File{{Path: "src/api.fy", Bytes: []byte("module api;\n")}}
	encoded, err := Encode(files)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	want := []byte("fairyc.package-source\x00\x01" +
		"\x00\x00\x00\x00\x00\x00\x00\x01" +
		"\x00\x00\x00\x00\x00\x00\x00\x0asrc/api.fy" +
		"\x00\x00\x00\x00\x00\x00\x00\x0cmodule api;\x0a")
	if string(encoded) != string(want) {
		t.Fatalf("Encode() = %s, want %s", hex.EncodeToString(encoded), hex.EncodeToString(want))
	}
	digest, err := Compute(files)
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	wantDigest := sha256.Sum256(want)
	if digest != Digest(wantDigest) {
		t.Fatalf("Compute() = %s, want %s", digest, Digest(wantDigest))
	}
}

func TestDigestSortsFilesButPreservesExactPathsAndBytes(t *testing.T) {
	t.Parallel()
	left := []File{{Path: "src/z.fy", Bytes: []byte("z")}, {Path: "src/a.fy", Bytes: []byte("a")}}
	right := []File{{Path: "src/a.fy", Bytes: []byte("a")}, {Path: "src/z.fy", Bytes: []byte("z")}}
	leftDigest, err := Compute(left)
	if err != nil {
		t.Fatalf("Compute(left) error = %v", err)
	}
	rightDigest, err := Compute(right)
	if err != nil {
		t.Fatalf("Compute(right) error = %v", err)
	}
	if leftDigest != rightDigest {
		t.Fatalf("input order changed digest: %s != %s", leftDigest, rightDigest)
	}

	changes := [][]File{
		{{Path: "src/a.fy", Bytes: []byte("b")}, {Path: "src/z.fy", Bytes: []byte("z")}},
		{{Path: "src/b.fy", Bytes: []byte("a")}, {Path: "src/z.fy", Bytes: []byte("z")}},
		{{Path: "src/a.fy", Bytes: []byte("a\r\n")}, {Path: "src/z.fy", Bytes: []byte("z")}},
	}
	for index, changed := range changes {
		digest, err := Compute(changed)
		if err != nil {
			t.Fatalf("Compute(change %d) error = %v", index, err)
		}
		if digest == leftDigest {
			t.Fatalf("exact change %d preserved digest %s", index, digest)
		}
	}
}

func TestDigestRejectsInvalidGraphs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files []File
	}{
		{name: "empty"},
		{name: "duplicate", files: []File{{Path: "a.fy"}, {Path: "a.fy"}}},
		{name: "absolute", files: []File{{Path: "/a.fy"}}},
		{name: "backslash", files: []File{{Path: `src\a.fy`}}},
		{name: "traversal", files: []File{{Path: "../a.fy"}}},
		{name: "unclean", files: []File{{Path: "src/./a.fy"}}},
		{name: "drive", files: []File{{Path: "C:/a.fy"}}},
		{name: "wrong extension", files: []File{{Path: "src/a.c"}}},
		{name: "invalid utf8", files: []File{{Path: string([]byte{0xff, '.', 'f', 'y'})}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Compute(test.files); err == nil {
				t.Fatalf("Compute(%+v) succeeded", test.files)
			}
		})
	}
}

func TestParseDigestRequiresCanonicalText(t *testing.T) {
	t.Parallel()
	digest, err := Compute([]File{{Path: "a.fy", Bytes: []byte("a")}})
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	parsed, err := ParseDigest(digest.String())
	if err != nil || parsed != digest {
		t.Fatalf("ParseDigest() = %s, %v; want %s", parsed, err, digest)
	}
	for _, value := range []string{
		strings.TrimPrefix(digest.String(), DigestPrefix),
		DigestPrefix + strings.ToUpper(strings.TrimPrefix(digest.String(), DigestPrefix)),
		digest.String() + "00",
		DigestPrefix + strings.Repeat("z", sha256.Size*2),
	} {
		if _, err := ParseDigest(value); err == nil {
			t.Fatalf("ParseDigest(%q) succeeded", value)
		}
	}
}
