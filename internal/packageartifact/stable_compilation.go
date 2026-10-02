package packageartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/contractstatemir"
)

const (
	StableCompilationSchema  = "fairyc.stable-package-compilation/v1"
	StableCompilationVersion = "fairyc-stable-package-compilation/3"
)

var (
	ErrStableCompilationTooLarge = errors.New("stable package compilation artifact exceeds the caller byte limit")
	ErrStableCompilationEncoding = errors.New("stable package compilation artifact is not canonically encoded")
)

// StableCompilationArtifact binds one package's stable MIR and exact
// contract-state body payload without merging their independent schemas.
type StableCompilationArtifact struct {
	MIR       StableArtifact                           `json:"mir"`
	StateBody contractstatemir.StableStateBodyArtifact `json:"stateBody"`
}

type stableCompilationEnvelope struct {
	Schema      string                    `json:"schema"`
	Version     string                    `json:"version"`
	Compilation StableCompilationArtifact `json:"compilation"`
}

// DescribeStableCompilationArtifact projects process-local MIR and binds it to
// one independently verified stable contract-state body.
func DescribeStableCompilationArtifact(artifact Artifact, stateBody contractstatemir.StableStateBodyArtifact, maxBytes uint64) (StableCompilationArtifact, error) {
	mirArtifact, err := DescribeStableArtifact(artifact, maxBytes)
	if err != nil {
		return StableCompilationArtifact{}, err
	}
	result := StableCompilationArtifact{MIR: mirArtifact, StateBody: stateBody}
	if err := ValidateStableCompilationArtifact(result); err != nil {
		return StableCompilationArtifact{}, err
	}
	return result, nil
}

// ValidateStableCompilationArtifact independently validates both payloads and
// requires one exact stable package-coordinate owner.
func ValidateStableCompilationArtifact(artifact StableCompilationArtifact) error {
	if err := ValidateStableArtifact(artifact.MIR); err != nil {
		return fmt.Errorf("stable compilation MIR: %w", err)
	}
	if err := contractstatemir.ValidateStableStateBody(artifact.StateBody); err != nil {
		return fmt.Errorf("stable compilation contract-state body: %w", err)
	}
	if artifact.MIR.Coordinate != artifact.StateBody.Package {
		return fmt.Errorf("stable compilation MIR and contract-state body have different package coordinates")
	}
	return nil
}

// EncodeStableCompilationArtifact returns canonical exact-version JSON under
// a mandatory caller limit.
func EncodeStableCompilationArtifact(artifact StableCompilationArtifact, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("stable package compilation artifact byte limit must be positive")
	}
	if err := ValidateStableCompilationArtifact(artifact); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(stableCompilationEnvelope{
		Schema: StableCompilationSchema, Version: StableCompilationVersion, Compilation: artifact,
	})
	if err != nil {
		return nil, fmt.Errorf("encode stable package compilation artifact: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrStableCompilationTooLarge
	}
	return encoded, nil
}

// DecodeStableCompilationArtifact enforces caller bounds, exact versions,
// strict fields, canonical JSON, and both nested semantic validators.
func DecodeStableCompilationArtifact(encoded []byte, maxBytes uint64) (StableCompilationArtifact, error) {
	if maxBytes == 0 {
		return StableCompilationArtifact{}, fmt.Errorf("stable package compilation artifact byte limit must be positive")
	}
	if len(encoded) == 0 {
		return StableCompilationArtifact{}, fmt.Errorf("stable package compilation artifact is empty")
	}
	if uint64(len(encoded)) > maxBytes {
		return StableCompilationArtifact{}, ErrStableCompilationTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var envelope stableCompilationEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return StableCompilationArtifact{}, fmt.Errorf("decode stable package compilation artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return StableCompilationArtifact{}, fmt.Errorf("decode stable package compilation artifact: trailing JSON value")
		}
		return StableCompilationArtifact{}, fmt.Errorf("decode stable package compilation artifact trailing data: %w", err)
	}
	if envelope.Schema != StableCompilationSchema || envelope.Version != StableCompilationVersion {
		return StableCompilationArtifact{}, fmt.Errorf("stable package compilation artifact schema or version is incompatible")
	}
	if err := ValidateStableCompilationArtifact(envelope.Compilation); err != nil {
		return StableCompilationArtifact{}, fmt.Errorf("verify decoded stable package compilation artifact: %w", err)
	}
	canonical, err := EncodeStableCompilationArtifact(envelope.Compilation, maxBytes)
	if err != nil {
		return StableCompilationArtifact{}, err
	}
	if !bytes.Equal(encoded, canonical) {
		return StableCompilationArtifact{}, ErrStableCompilationEncoding
	}
	return envelope.Compilation, nil
}
