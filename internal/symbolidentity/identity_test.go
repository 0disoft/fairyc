package symbolidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestEncodeMatchesCanonicalV1Vector(t *testing.T) {
	t.Parallel()
	if Algorithm != "fairyc.symbol-id/v1-sha256-128+256" || Encoding != "fairyc.symbol-identity/v1" ||
		SignatureEncoding != "fairyc.symbol-signature/v1" || VisiblePrefix != "fysym1-sha256-128-" ||
		DigestPrefix != "fysym1-sha256-256-" {
		t.Fatalf("identity contract constants drifted")
	}
	descriptor := Descriptor{
		PackageUID: "example.math", Module: "math", Kind: "function",
		DeclarationPath: "add", GenericArity: 0, Signature: []byte("signature"),
	}
	encoded, err := Encode(descriptor)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	want := []byte("fairyc.symbol-identity\x00\x01" +
		"u\x00\x00\x00\x00\x00\x00\x00\x0cexample.math" +
		"m\x00\x00\x00\x00\x00\x00\x00\x04math" +
		"k\x00\x00\x00\x00\x00\x00\x00\x08function" +
		"p\x00\x00\x00\x00\x00\x00\x00\x03add" +
		"g\x00\x00\x00\x00\x00\x00\x00\x00" +
		"s\x00\x00\x00\x00\x00\x00\x00\x09signature")
	if string(encoded) != string(want) {
		t.Fatalf("Encode() = %s, want %s", hex.EncodeToString(encoded), hex.EncodeToString(want))
	}

	id, err := Compute(descriptor)
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	wantDigest := FullDigest(sha256.Sum256(want))
	var wantVisible VisibleID
	copy(wantVisible[:], wantDigest[:len(wantVisible)])
	if id.Visible != wantVisible || id.Full != wantDigest {
		t.Fatalf("Compute() = %+v, want visible %s full %s", id, wantVisible, wantDigest)
	}
	if id.Visible.String() != "fysym1-sha256-128-"+hex.EncodeToString(wantDigest[:len(wantVisible)]) ||
		id.Full.String() != "fysym1-sha256-256-"+hex.EncodeToString(wantDigest[:]) {
		t.Fatalf("canonical text = %s / %s", id.Visible, id.Full)
	}
}

func TestIdentityEncodingPreservesEveryTupleBoundary(t *testing.T) {
	t.Parallel()
	base := Descriptor{PackageUID: "pkg", Module: "main", Kind: "function", DeclarationPath: "run", Signature: []byte("sig")}
	variants := []Descriptor{
		{PackageUID: "pkg2", Module: base.Module, Kind: base.Kind, DeclarationPath: base.DeclarationPath, Signature: base.Signature},
		{PackageUID: base.PackageUID, Module: "other", Kind: base.Kind, DeclarationPath: base.DeclarationPath, Signature: base.Signature},
		{PackageUID: base.PackageUID, Module: base.Module, Kind: "struct", DeclarationPath: base.DeclarationPath, Signature: base.Signature},
		{PackageUID: base.PackageUID, Module: base.Module, Kind: base.Kind, DeclarationPath: "other", Signature: base.Signature},
		{PackageUID: base.PackageUID, Module: base.Module, Kind: base.Kind, DeclarationPath: base.DeclarationPath, GenericArity: 1, Signature: base.Signature},
		{PackageUID: base.PackageUID, Module: base.Module, Kind: base.Kind, DeclarationPath: base.DeclarationPath, Signature: []byte("sig2")},
	}
	baseID, err := Compute(base)
	if err != nil {
		t.Fatal(err)
	}
	for index, variant := range variants {
		id, computeErr := Compute(variant)
		if computeErr != nil {
			t.Fatalf("variant %d Compute() error = %v", index, computeErr)
		}
		if id == baseID {
			t.Fatalf("variant %d did not change identity", index)
		}
	}

	left, err := Compute(Descriptor{PackageUID: "ab", Module: "c", Kind: "function", DeclarationPath: "d", Signature: []byte("e")})
	if err != nil {
		t.Fatal(err)
	}
	right, err := Compute(Descriptor{PackageUID: "a", Module: "bc", Kind: "function", DeclarationPath: "d", Signature: []byte("e")})
	if err != nil {
		t.Fatal(err)
	}
	if left == right {
		t.Fatalf("component-boundary descriptors collided: %+v", left)
	}
}

func TestParseRequiresCanonicalFullWidthText(t *testing.T) {
	t.Parallel()
	id, err := Compute(Descriptor{PackageUID: "pkg", Module: "main", Kind: "function", DeclarationPath: "run", Signature: []byte("sig")})
	if err != nil {
		t.Fatal(err)
	}
	visible, err := ParseVisibleID(id.Visible.String())
	if err != nil || visible != id.Visible {
		t.Fatalf("ParseVisibleID() = %s, %v", visible, err)
	}
	full, err := ParseFullDigest(id.Full.String())
	if err != nil || full != id.Full {
		t.Fatalf("ParseFullDigest() = %s, %v", full, err)
	}
	for _, value := range []string{
		strings.TrimPrefix(id.Visible.String(), VisiblePrefix),
		VisiblePrefix + strings.ToUpper(strings.TrimPrefix(id.Visible.String(), VisiblePrefix)),
		id.Visible.String() + "00",
		VisiblePrefix + strings.Repeat("z", len(id.Visible)*2),
	} {
		if _, parseErr := ParseVisibleID(value); parseErr == nil {
			t.Fatalf("ParseVisibleID(%q) succeeded", value)
		}
	}
	for _, value := range []string{
		strings.TrimPrefix(id.Full.String(), DigestPrefix),
		DigestPrefix + strings.ToUpper(strings.TrimPrefix(id.Full.String(), DigestPrefix)),
		id.Full.String() + "00",
		DigestPrefix + strings.Repeat("z", len(id.Full)*2),
	} {
		if _, parseErr := ParseFullDigest(value); parseErr == nil {
			t.Fatalf("ParseFullDigest(%q) succeeded", value)
		}
	}
}

func TestDescriptorValidationRejectsUnstableOrAmbiguousCoordinates(t *testing.T) {
	t.Parallel()
	valid := Descriptor{PackageUID: "pkg", Module: "main", Kind: "function", DeclarationPath: "run", Signature: []byte("sig")}
	tests := []struct {
		name       string
		mutate     func(*Descriptor)
		wantReason string
	}{
		{name: "empty uid", mutate: func(value *Descriptor) { value.PackageUID = "" }, wantReason: ReasonEmptyComponent},
		{name: "invalid uid utf8", mutate: func(value *Descriptor) { value.PackageUID = string([]byte{0xff}) }, wantReason: ReasonInvalidUTF8},
		{name: "uid control", mutate: func(value *Descriptor) { value.PackageUID = "pkg\nother" }, wantReason: ReasonControlCharacter},
		{name: "uid unicode control", mutate: func(value *Descriptor) { value.PackageUID = "pkg\u0085other" }, wantReason: ReasonControlCharacter},
		{name: "invalid module", mutate: func(value *Descriptor) { value.Module = "main/other" }, wantReason: ReasonInvalidPath},
		{name: "empty module segment", mutate: func(value *Descriptor) { value.Module = "main::" }, wantReason: ReasonInvalidPath},
		{name: "invalid declaration path", mutate: func(value *Descriptor) { value.DeclarationPath = "Type::bad-name" }, wantReason: ReasonInvalidPath},
		{name: "invalid kind", mutate: func(value *Descriptor) { value.Kind = "variable" }, wantReason: ReasonInvalidKind},
		{name: "empty signature", mutate: func(value *Descriptor) { value.Signature = nil }, wantReason: ReasonEmptySignature},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := valid
			test.mutate(&candidate)
			_, err := Compute(candidate)
			var validationError *ValidationError
			if !errors.As(err, &validationError) || validationError.Reason != test.wantReason {
				t.Fatalf("Compute() error = %v, want reason %q", err, test.wantReason)
			}
		})
	}
}

func TestValidateCollisionSetRejectsMismatchAndVisibleCollision(t *testing.T) {
	t.Parallel()
	var leftFull FullDigest
	var rightFull FullDigest
	for index := range leftFull {
		leftFull[index] = byte(index)
		rightFull[index] = byte(index)
	}
	rightFull[len(VisibleID{})] ^= 0xff
	var visible VisibleID
	copy(visible[:], leftFull[:len(visible)])
	left := ID{Visible: visible, Full: leftFull}
	right := ID{Visible: visible, Full: rightFull}
	if err := ValidateCollisionSet([]ID{left, left}); err != nil {
		t.Fatalf("duplicate exact identity rejected: %v", err)
	}
	var collision *CollisionError
	if err := ValidateCollisionSet([]ID{left, right}); !errors.As(err, &collision) {
		t.Fatalf("collision error = %v", err)
	}
	malformed := left
	malformed.Visible[0] ^= 0xff
	if err := ValidateCollisionSet([]ID{malformed}); err == nil || errors.As(err, &collision) {
		t.Fatalf("malformed identity error = %v", err)
	}
}
