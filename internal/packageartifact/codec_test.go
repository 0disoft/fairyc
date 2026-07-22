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
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
	if err != nil {
		t.Fatal(err)
	}
	codec := fixedProcessCodec(t, 1)
	encoded, err := codec.Encode(artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(encoded, []byte(`{"schema":"fairyc.package-mir-process/v1"`)) {
		t.Fatalf("encoded prefix = %q", encoded[:min(len(encoded), 80)])
	}
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, artifacts[0]) {
		t.Fatal("decoded process-local artifact differs from source")
	}
}

func TestProcessCodecRejectsForeignNoncanonicalAndCorruptBytes(t *testing.T) {
	t.Parallel()
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
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
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
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
