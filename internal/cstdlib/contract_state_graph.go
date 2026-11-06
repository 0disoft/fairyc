package cstdlib

import (
	_ "embed"
	"strings"
)

//go:embed stage0_contract_state_graph.h
var contractStateGraphHeader string

//go:embed stage0_contract_state_graph.c
var contractStateGraphImplementation string

//go:embed stage0_contract_state_graph_key.c
var contractStateGraphKey string

// ContractStateGraphDeclarations selects the private graph value/borrow kernel.
// This alone grants no source, native graph execution or publication authority.
func ContractStateGraphDeclarations() (string, error) {
	header, err := canonicalSource("contract-state graph declarations", contractStateGraphHeader)
	if err != nil {
		return "", err
	}
	overlay, err := ContractStateOverlayDeclarations()
	if err != nil {
		return "", err
	}
	return strings.Replace(header, `#include "stage0_contract_state_overlay.h"`, overlay, 1), nil
}

func ContractStateGraphRuntimeSource() (string, error) {
	header, err := ContractStateGraphDeclarations()
	if err != nil {
		return "", err
	}
	implementation, err := canonicalSource("contract-state graph runtime", contractStateGraphImplementation)
	if err != nil {
		return "", err
	}
	overlay, err := ContractStateOverlayRuntimeSource()
	if err != nil {
		return "", err
	}
	key, err := canonicalSource("contract-state graph map key", contractStateGraphKey)
	if err != nil {
		return "", err
	}
	return overlay + "\n" + strings.Replace(implementation, `#include "stage0_contract_state_graph.h"`, header, 1) + "\n" + strings.Replace(key, `#include "stage0_contract_state_graph.h"`, header, 1), nil
}
