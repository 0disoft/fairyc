package packagecontent

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestFileAndLineMapEncodingsMatchCanonicalV1Vectors(t *testing.T) {
	t.Parallel()
	if FileDigestEncoding != "fairyc.source-file/v1" || FileDigestPrefix != "fyfile1-sha256-" ||
		LineMapEncoding != "fairyc.source-line-map/v1" || LineMapPrefix != "fylm1-sha256-" {
		t.Fatal("source file identity constants drifted")
	}
	contents := []byte("a\nb\r\n")
	fileBytes := EncodeFileDigestInput(contents)
	wantFileBytes := []byte("fairyc.source-file\x00\x01" +
		"\x00\x00\x00\x00\x00\x00\x00\x05a\nb\r\n")
	if string(fileBytes) != string(wantFileBytes) {
		t.Fatalf("EncodeFileDigestInput() = %s, want %s", hex.EncodeToString(fileBytes), hex.EncodeToString(wantFileBytes))
	}
	fileDigest := ComputeFileDigest(contents)
	wantFileDigest := FileDigest(sha256.Sum256(wantFileBytes))
	if fileDigest != wantFileDigest {
		t.Fatalf("ComputeFileDigest() = %s, want %s", fileDigest, wantFileDigest)
	}

	lineBytes := EncodeLineMapInput(contents)
	wantLineBytes := []byte("fairyc.source-line-map\x00\x01" +
		"\x00\x00\x00\x00\x00\x00\x00\x05" +
		"\x00\x00\x00\x00\x00\x00\x00\x03" +
		"\x00\x00\x00\x00\x00\x00\x00\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x02" +
		"\x00\x00\x00\x00\x00\x00\x00\x05")
	if string(lineBytes) != string(wantLineBytes) {
		t.Fatalf("EncodeLineMapInput() = %s, want %s", hex.EncodeToString(lineBytes), hex.EncodeToString(wantLineBytes))
	}
	lineMap := ComputeLineMapID(contents)
	wantLineMap := LineMapID(sha256.Sum256(wantLineBytes))
	if lineMap != wantLineMap {
		t.Fatalf("ComputeLineMapID() = %s, want %s", lineMap, wantLineMap)
	}
	combinedDigest, combinedLineMap := ComputeFileIdentities(contents)
	if combinedDigest != fileDigest || combinedLineMap != lineMap {
		t.Fatalf("ComputeFileIdentities() = %s, %s", combinedDigest, combinedLineMap)
	}
}

func TestLineMapSeparatesContentChangesFromByteGeometryChanges(t *testing.T) {
	t.Parallel()
	left := []byte("a\nb\n")
	sameGeometry := []byte("x\ny\n")
	crlf := []byte("a\r\nb\r\n")
	if ComputeFileDigest(left) == ComputeFileDigest(sameGeometry) {
		t.Fatal("content change preserved file digest")
	}
	if ComputeLineMapID(left) != ComputeLineMapID(sameGeometry) {
		t.Fatal("equal byte length and line starts changed line-map identity")
	}
	if ComputeLineMapID(left) == ComputeLineMapID(crlf) {
		t.Fatal("CRLF byte geometry preserved line-map identity")
	}
	if ComputeLineMapID([]byte("a")) == ComputeLineMapID([]byte("a\n")) {
		t.Fatal("final LF did not add the EOF line start")
	}
	emptyBytes := EncodeLineMapInput(nil)
	if len(emptyBytes) == 0 || ComputeLineMapID(nil) != LineMapID(sha256.Sum256(emptyBytes)) {
		t.Fatal("empty source line-map vector is inconsistent")
	}
}

func TestFileAndLineMapTextRequireCanonicalFullDigests(t *testing.T) {
	t.Parallel()
	fileDigest := ComputeFileDigest([]byte("module main;\n"))
	parsedFile, err := ParseFileDigest(fileDigest.String())
	if err != nil || parsedFile != fileDigest {
		t.Fatalf("ParseFileDigest() = %s, %v", parsedFile, err)
	}
	lineMap := ComputeLineMapID([]byte("module main;\n"))
	parsedMap, err := ParseLineMapID(lineMap.String())
	if err != nil || parsedMap != lineMap {
		t.Fatalf("ParseLineMapID() = %s, %v", parsedMap, err)
	}
	for _, value := range []string{
		strings.TrimPrefix(fileDigest.String(), FileDigestPrefix),
		FileDigestPrefix + strings.ToUpper(strings.TrimPrefix(fileDigest.String(), FileDigestPrefix)),
		fileDigest.String() + "00",
		FileDigestPrefix + strings.Repeat("z", sha256.Size*2),
	} {
		if _, parseErr := ParseFileDigest(value); parseErr == nil {
			t.Fatalf("ParseFileDigest(%q) succeeded", value)
		}
	}
	for _, value := range []string{
		strings.TrimPrefix(lineMap.String(), LineMapPrefix),
		LineMapPrefix + strings.ToUpper(strings.TrimPrefix(lineMap.String(), LineMapPrefix)),
		lineMap.String() + "00",
		LineMapPrefix + strings.Repeat("z", sha256.Size*2),
	} {
		if _, parseErr := ParseLineMapID(value); parseErr == nil {
			t.Fatalf("ParseLineMapID(%q) succeeded", value)
		}
	}
}
