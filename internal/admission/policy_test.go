package admission

import "testing"

func TestAdmissionBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy[uint32]
		valid  bool
	}{
		{"ordinary", Policy[uint32]{}, true},
		{"query", Policy[uint32]{Capabilities: Capabilities{Storage: StorageQueries}}, true},
		{"transfer", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers}}, true},
		{"default-package-owner", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers}, StorageOwners: []uint32{0}}, true},
		{"empty-scope", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers}, StorageOwners: []uint32{}}, false},
		{"duplicate-scope", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers}, StorageOwners: []uint32{1, 1}}, false},
		{"owner-without-storage", Policy[uint32]{StorageOwners: []uint32{1}}, false},
		{"sdk-without-authority", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers, Arguments: NominalArguments, Templates: SDKOwnerTemplates}}, false},
		{"sdk-without-nominal", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers, Templates: SDKOwnerTemplates}, StorageOwners: []uint32{1}}, false},
		{"sdk", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers, Arguments: NominalArguments, Templates: SDKOwnerTemplates}, StorageOwners: []uint32{1}}, true},
		{"candidate-without-sdk", Policy[uint32]{Capabilities: Capabilities{Vectors: CandidateVectors}}, false},
		{"candidate", Policy[uint32]{Capabilities: Capabilities{Storage: StorageTransfers, Arguments: NominalArguments, Templates: SDKOwnerTemplates, Vectors: CandidateVectors}, StorageOwners: []uint32{1}}, true},
		{"unknown-storage", Policy[uint32]{Capabilities: Capabilities{Storage: 255}}, false},
		{"unknown-argument", Policy[uint32]{Capabilities: Capabilities{Arguments: 255}}, false},
		{"unknown-template", Policy[uint32]{Capabilities: Capabilities{Templates: 255}}, false},
		{"unknown-vector", Policy[uint32]{Capabilities: Capabilities{Vectors: 255}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.policy.Validate(); (err == nil) != tt.valid {
				t.Fatalf("valid=%v: %v", tt.valid, err)
			}
		})
	}
}
