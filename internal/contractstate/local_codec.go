package contractstate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
)

type localState struct {
	layout  Layout
	version uint64
	entries []Entry
}

type localPayload struct {
	Schema      string     `json:"schema"`
	ContractUID string     `json:"contractUid"`
	LayoutHash  LayoutHash `json:"layoutHash"`
	Fields      []Field    `json:"fields"`
	Version     uint64     `json:"version"`
	Entries     []Entry    `json:"entries"`
}

const localSchema = "fairyc.local-state/v1"

func encodeLocalState(current localState, limits LocalPersistenceLimits) ([]byte, error) {
	if err := validateLayout(current.layout); err != nil {
		return nil, err
	}
	if len(current.layout.fields) > 65536 || len(current.entries) > limits.MaxEntries {
		return nil, fmt.Errorf("local state count limit exceeded")
	}
	var total int64
	for _, field := range current.layout.fields {
		total += int64(len(field.Name) + len(field.KeyTypeID) + len(field.ValueTypeID))
	}
	for index, entry := range current.entries {
		if len(entry.Value) > limits.MaxValueBytes {
			return nil, ErrValueTooLarge
		}
		total += int64(len(entry.Value)) + 32
		if total > limits.MaxStateBytes {
			return nil, fmt.Errorf("local state bytes exceed file size limit")
		}
		if index > 0 && slices.Compare(current.entries[index-1].Key[:], entry.Key[:]) >= 0 {
			return nil, fmt.Errorf("local state keys are not canonical")
		}
	}
	payload := localPayload{Schema: localSchema, ContractUID: current.layout.ContractUID(), LayoutHash: current.layout.Hash(), Fields: current.layout.Fields(), Version: current.version, Entries: cloneLocalEntries(current.entries)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if int64(len(encoded)+sha256.Size) > limits.MaxStateBytes {
		return nil, fmt.Errorf("local state exceeds file size limit")
	}
	digest := sha256.Sum256(encoded)
	return append(encoded, digest[:]...), nil
}

func decodeLocalState(encoded []byte, limits LocalPersistenceLimits) (localState, error) {
	if len(encoded) < sha256.Size || int64(len(encoded)) > limits.MaxStateBytes {
		return localState{}, fmt.Errorf("invalid local state file size")
	}
	body, checksum := encoded[:len(encoded)-sha256.Size], encoded[len(encoded)-sha256.Size:]
	digest := sha256.Sum256(body)
	if !bytes.Equal(digest[:], checksum) {
		return localState{}, fmt.Errorf("local state checksum mismatch")
	}
	var payload localPayload
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return localState{}, err
	}
	if payload.Schema != localSchema || len(payload.Fields) > 65536 {
		return localState{}, fmt.Errorf("invalid local state schema")
	}
	builder, err := NewLayoutBuilder(payload.ContractUID)
	if err != nil {
		return localState{}, err
	}
	for _, field := range payload.Fields {
		if _, err := builder.addField(field.Name, field.Kind, field.KeyTypeID, field.ValueTypeID); err != nil {
			return localState{}, err
		}
	}
	layout, err := builder.Build()
	if err != nil {
		return localState{}, err
	}
	if layout.Hash() != payload.LayoutHash || !slices.Equal(layout.fields, payload.Fields) {
		return localState{}, fmt.Errorf("local state layout identity mismatch")
	}
	current := localState{layout: layout, version: payload.Version, entries: payload.Entries}
	canonical, err := encodeLocalState(current, limits)
	if err != nil {
		return localState{}, err
	}
	if !bytes.Equal(canonical, encoded) {
		return localState{}, fmt.Errorf("local state encoding is not canonical")
	}
	return current, nil
}
