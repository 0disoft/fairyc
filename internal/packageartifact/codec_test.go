package packageartifact

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const processArtifactLimit = 8 << 20

func TestProcessCodecRoundTripsVerifiedArtifact(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	codec := fixedProcessCodec(t, 1)
	encoded, err := codec.Encode(artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(encoded, []byte(`{"schema":"fairyc.package-mir-process/v3"`)) {
		t.Fatalf("encoded prefix = %q", encoded[:min(len(encoded), 80)])
	}
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, artifacts[0]) {
		t.Fatal("decoded process-local artifact differs from source")
	}
	old := bytes.Replace(encoded, []byte(ProcessCodecVersion), []byte("fairyc-package-mir-process/2"), 1)
	if _, err := codec.Decode(old); err == nil {
		t.Fatal("accepted previous lossy process artifact version")
	}
}

func TestProcessCodecRejectsForeignNoncanonicalAndCorruptBytes(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	codec := fixedProcessCodec(t, 1)
	encoded, err := codec.Encode(artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	foreign := fixedProcessCodec(t, 2)
	if _, err := foreign.Decode(encoded); !errors.Is(err, ErrProcessArtifactForeign) {
		t.Fatalf("foreign codec error = %v", err)
	}
	withWhitespace := append(append([]byte(nil), encoded...), '\n')
	if _, err := codec.Decode(withWhitespace); !errors.Is(err, ErrProcessArtifactEncoding) {
		t.Fatalf("noncanonical bytes error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)
	if _, err := codec.Decode(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
	corrupt := bytes.Replace(encoded, []byte(`"Entry":0`), []byte(`"Entry":999`), 1)
	if bytes.Equal(corrupt, encoded) {
		t.Fatal("encoded artifact did not contain an entry block")
	}
	if _, err := codec.Decode(corrupt); err == nil || !strings.Contains(err.Error(), "verify decoded") {
		t.Fatalf("corrupt body error = %v", err)
	}
}

func TestProcessCodecRequiresIdentityAndBounds(t *testing.T) {
	t.Parallel()
	if _, err := NewProcessCodec(0); err == nil {
		t.Fatal("zero byte limit accepted")
	}
	if _, err := newProcessCodec([32]byte{}, 1); err == nil {
		t.Fatal("zero process identity accepted")
	}
	codec := fixedProcessCodec(t, 1)
	parsed, err := parseProcessIdentity(codec.Identity())
	if err != nil || parsed != codec.identity {
		t.Fatalf("parseProcessIdentity() = %x, %v", parsed, err)
	}
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.Encode(artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	small, err := newProcessCodec(codec.identity, uint64(len(encoded)-1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := small.Encode(artifacts[0]); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("small encode limit error = %v", err)
	}
	if _, err := small.Decode(encoded); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("small decode limit error = %v", err)
	}
}

func TestProcessCodecBoundsCanonicalReencoding(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifacts[0]
	artifact.Functions[0].Body.Span.Source = "a<b.fy"
	codec := fixedProcessCodec(t, 1)
	canonical, err := codec.Encode(artifact)
	if err != nil {
		t.Fatal(err)
	}
	// Valid JSON may be shorter than its mandatory HTML-escaped canonical form.
	raw := bytes.ReplaceAll(canonical, []byte(`\u003c`), []byte("<"))
	if len(raw) >= len(canonical) {
		t.Fatal("test requires a shorter noncanonical encoding")
	}
	if _, err := codec.Decode(raw); !errors.Is(err, ErrProcessArtifactEncoding) {
		t.Fatalf("noncanonical raw JSON error = %v", err)
	}
	bounded, err := newProcessCodec(codec.identity, uint64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bounded.Decode(raw); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("canonical encoding exceeding the byte limit: %v", err)
	}
}

func fixedProcessCodec(t *testing.T, marker byte) *ProcessCodec {
	t.Helper()
	var identity [32]byte
	identity[len(identity)-1] = marker
	codec, err := newProcessCodec(identity, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	return codec
}
