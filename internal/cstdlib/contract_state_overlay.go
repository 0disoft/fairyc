package cstdlib

import (
	_ "embed"
	"strings"
)

//go:embed stage0_contract_state_overlay.h
var contractStateOverlayHeader string

//go:embed stage0_contract_state_overlay.c
var contractStateOverlayImplementation string

// ContractStateOverlayDeclarations returns the private physical-key overlay ABI.
// It admits no source feature, target or persistent publication authority.
func ContractStateOverlayDeclarations() (string, error) {
	return canonicalSource("contract-state overlay declarations", contractStateOverlayHeader)
}

// ContractStateOverlayRuntimeSource returns an allocation-free multi-key kernel.
// It is separate from the original single-cell runtime and its admission policy.
func ContractStateOverlayRuntimeSource() (string, error) {
	header, err := ContractStateOverlayDeclarations()
	if err != nil {
		return "", err
	}
	implementation, err := canonicalSource("contract-state overlay runtime", contractStateOverlayImplementation)
	if err != nil {
		return "", err
	}
	return strings.Replace(implementation, `#include "stage0_contract_state_overlay.h"`, header, 1), nil
}
