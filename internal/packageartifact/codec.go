package packageartifact

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	ProcessCodecSchema    = "fairyc.package-mir-process/v0"
	ProcessCodecVersion   = "fairyc-package-mir-process/0"
	processIdentityPrefix = "fyproc0-"
)

var (
	ErrProcessArtifactTooLarge = errors.New("process-local package artifact exceeds the caller byte limit")
	ErrProcessArtifactForeign  = errors.New("process-local package artifact belongs to another compiler process")
	ErrProcessArtifactEncoding = errors.New("process-local package artifact is not canonically encoded")
)

// ProcessCodec encodes package artifacts only for one compiler process. Its random identity is
// embedded in every payload, so copied bytes fail closed in another process or codec instance.
type ProcessCodec struct {
	identity [32]byte
	maxBytes uint64
}

type processEnvelope struct {
	Schema   string   `json:"schema"`
	Version  string   `json:"version"`
	Process  string   `json:"process"`
	Artifact Artifact `json:"artifact"`
}

// NewProcessCodec creates one bounded codec with a cryptographically random non-secret process
// identity. The identity provides isolation, not authentication.
func NewProcessCodec(maxBytes uint64) (*ProcessCodec, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("process-local package artifact byte limit must be positive")
	}
	var identity [32]byte
	if _, err := io.ReadFull(rand.Reader, identity[:]); err != nil {
		return nil, fmt.Errorf("create process-local package artifact identity: %w", err)
	}
	return newProcessCodec(identity, maxBytes)
}

func newProcessCodec(identity [32]byte, maxBytes uint64) (*ProcessCodec, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("process-local package artifact byte limit must be positive")
	}
	if identity == ([32]byte{}) {
		return nil, fmt.Errorf("process-local package artifact identity must be nonzero")
	}
	return &ProcessCodec{identity: identity, maxBytes: maxBytes}, nil
}

// Identity returns the process-private compatibility member used by in-memory cache keys.
func (codec *ProcessCodec) Identity() string {
	if codec == nil {
		return ""
	}
	return processIdentityPrefix + hex.EncodeToString(codec.identity[:])
}

// Encode verifies and canonically encodes one artifact under this codec's mandatory byte limit.
func (codec *ProcessCodec) Encode(artifact Artifact) ([]byte, error) {
	if codec == nil || codec.maxBytes == 0 || codec.identity == ([32]byte{}) {
		return nil, fmt.Errorf("process-local package artifact codec is not initialized")
	}
	if err := Verify(artifact); err != nil {
		return nil, fmt.Errorf("verify process-local package artifact before encoding: %w", err)
	}
	encoded, err := json.Marshal(processEnvelope{
		Schema: ProcessCodecSchema, Version: ProcessCodecVersion, Process: codec.Identity(), Artifact: artifact,
	})
	if err != nil {
		return nil, fmt.Errorf("encode process-local package artifact: %w", err)
	}
	if uint64(len(encoded)) > codec.maxBytes {
		return nil, ErrProcessArtifactTooLarge
	}
	return encoded, nil
}

// Decode accepts only canonical bytes created by this exact codec instance and reruns artifact
// verification before returning independently owned compiler state.
func (codec *ProcessCodec) Decode(encoded []byte) (Artifact, error) {
	if codec == nil || codec.maxBytes == 0 || codec.identity == ([32]byte{}) {
		return Artifact{}, fmt.Errorf("process-local package artifact codec is not initialized")
	}
	if len(encoded) == 0 {
		return Artifact{}, fmt.Errorf("process-local package artifact is empty")
	}
	if uint64(len(encoded)) > codec.maxBytes {
		return Artifact{}, ErrProcessArtifactTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope processEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return Artifact{}, fmt.Errorf("decode process-local package artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Artifact{}, fmt.Errorf("decode process-local package artifact: trailing JSON value")
		}
		return Artifact{}, fmt.Errorf("decode process-local package artifact trailing data: %w", err)
	}
	if envelope.Schema != ProcessCodecSchema {
		return Artifact{}, fmt.Errorf("process-local package artifact schema %q is incompatible with %q", envelope.Schema, ProcessCodecSchema)
	}
	if envelope.Version != ProcessCodecVersion {
		return Artifact{}, fmt.Errorf("process-local package artifact version %q is incompatible with %q", envelope.Version, ProcessCodecVersion)
	}
	if envelope.Process != codec.Identity() {
		return Artifact{}, ErrProcessArtifactForeign
	}
	if err := Verify(envelope.Artifact); err != nil {
		return Artifact{}, fmt.Errorf("verify decoded process-local package artifact: %w", err)
	}
	canonical, err := codec.Encode(envelope.Artifact)
	if err != nil {
		return Artifact{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return Artifact{}, ErrProcessArtifactEncoding
	}
	return envelope.Artifact, nil
}

func parseProcessIdentity(value string) ([32]byte, error) {
	var identity [32]byte
	if !strings.HasPrefix(value, processIdentityPrefix) {
		return identity, fmt.Errorf("process identity has invalid prefix")
	}
	text := strings.TrimPrefix(value, processIdentityPrefix)
	if len(text) != 64 || text != strings.ToLower(text) {
		return identity, fmt.Errorf("process identity must contain 64 lowercase hexadecimal digits")
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return identity, fmt.Errorf("decode process identity: %w", err)
	}
	copy(identity[:], decoded)
	return identity, nil
}
