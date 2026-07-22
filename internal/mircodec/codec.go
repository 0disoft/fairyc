// Package mircodec owns the exact-version, compiler-private encoding of verified MIR programs.
package mircodec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/mir"
)

const (
	Schema     = "fairyc.mir-artifact/v0"
	MIRVersion = "fairyc-mir/0"
)

var (
	ErrArtifactTooLarge = errors.New("MIR artifact exceeds the caller byte limit")
	ErrNonCanonical     = errors.New("MIR artifact is not canonically encoded")
)

type envelope struct {
	Schema     string      `json:"schema"`
	MIRVersion string      `json:"mirVersion"`
	Program    mir.Program `json:"program"`
}

// Encode verifies program and returns its deterministic exact-version JSON bytes. The caller must
// supply a positive byte limit; Stage 0 deliberately has no guessed universal artifact budget.
func Encode(program mir.Program, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("MIR artifact byte limit must be positive")
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return nil, fmt.Errorf("verify MIR before encoding: %w", failures[0])
	}
	encoded, err := json.Marshal(envelope{Schema: Schema, MIRVersion: MIRVersion, Program: program})
	if err != nil {
		return nil, fmt.Errorf("encode MIR artifact: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrArtifactTooLarge
	}
	return encoded, nil
}

// Decode accepts only one canonical exact-version envelope, enforces the caller's byte limit, and
// reruns the MIR verifier before returning executable compiler state.
func Decode(encoded []byte, maxBytes uint64) (mir.Program, error) {
	if maxBytes == 0 {
		return mir.Program{}, fmt.Errorf("MIR artifact byte limit must be positive")
	}
	if len(encoded) == 0 {
		return mir.Program{}, fmt.Errorf("MIR artifact is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return mir.Program{}, ErrArtifactTooLarge
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var artifact envelope
	if err := decoder.Decode(&artifact); err != nil {
		return mir.Program{}, fmt.Errorf("decode MIR artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return mir.Program{}, fmt.Errorf("decode MIR artifact: trailing JSON value")
		}
		return mir.Program{}, fmt.Errorf("decode MIR artifact trailing data: %w", err)
	}
	if artifact.Schema != Schema {
		return mir.Program{}, fmt.Errorf("MIR artifact schema %q is incompatible with %q", artifact.Schema, Schema)
	}
	if artifact.MIRVersion != MIRVersion {
		return mir.Program{}, fmt.Errorf("MIR artifact version %q is incompatible with %q", artifact.MIRVersion, MIRVersion)
	}
	if failures := mir.Verify(artifact.Program); len(failures) != 0 {
		return mir.Program{}, fmt.Errorf("verify decoded MIR artifact: %w", failures[0])
	}
	canonical, err := Encode(artifact.Program, maxBytes)
	if err != nil {
		return mir.Program{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return mir.Program{}, ErrNonCanonical
	}
	return artifact.Program, nil
}
