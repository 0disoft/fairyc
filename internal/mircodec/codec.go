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
	MIRVersion = "fairyc-mir/2"
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
	return encode(program, maxBytes, Schema, mir.Verify)
}

func encode(program mir.Program, maxBytes uint64, schema string, verify func(mir.Program) []mir.VerificationError) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("MIR artifact byte limit must be positive")
	}
	if failures := verify(program); len(failures) != 0 {
		return nil, fmt.Errorf("verify MIR before encoding: %w", failures[0])
	}
	encoded, err := json.Marshal(envelope{Schema: schema, MIRVersion: MIRVersion, Program: program})
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
	return decode(encoded, maxBytes, Schema, mir.Verify)
}

func decode(encoded []byte, maxBytes uint64, schema string, verify func(mir.Program) []mir.VerificationError) (mir.Program, error) {
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
	if artifact.Schema != schema {
		return mir.Program{}, fmt.Errorf("MIR artifact schema %q is incompatible with %q", artifact.Schema, schema)
	}
	if artifact.MIRVersion != MIRVersion {
		return mir.Program{}, fmt.Errorf("MIR artifact version %q is incompatible with %q", artifact.MIRVersion, MIRVersion)
	}
	if failures := verify(artifact.Program); len(failures) != 0 {
		return mir.Program{}, fmt.Errorf("verify decoded MIR artifact: %w", failures[0])
	}
	canonical, err := encode(artifact.Program, maxBytes, schema, verify)
	if err != nil {
		return mir.Program{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return mir.Program{}, ErrNonCanonical
	}
	return artifact.Program, nil
}
