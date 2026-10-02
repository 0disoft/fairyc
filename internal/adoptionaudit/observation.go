// Package adoptionaudit checks retained historical evidence without executing it.
package adoptionaudit

import (
	"encoding/json"
	"fmt"
)

type observation struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func decodeObservation(raw json.RawMessage) (observation, error) {
	var fields struct {
		ExitCode *int    `json:"exitCode"`
		Stdout   *string `json:"stdout"`
		Stderr   *string `json:"stderr"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return observation{}, err
	}
	if fields.ExitCode == nil || fields.Stdout == nil || fields.Stderr == nil {
		return observation{}, fmt.Errorf("malformed output observation")
	}
	return observation{*fields.ExitCode, *fields.Stdout, *fields.Stderr}, nil
}

func compare(a, b json.RawMessage) error {
	left, err := decodeObservation(a)
	if err != nil {
		return err
	}
	right, err := decodeObservation(b)
	if err != nil {
		return err
	}
	if left != right {
		return fmt.Errorf("ordinal output mismatch")
	}
	return nil
}
