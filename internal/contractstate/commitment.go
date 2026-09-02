package contractstate

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"sort"
)

const CommitmentAlgorithm = "fairyc.contract-state.merkle/v1"

var (
	ErrCommitmentLimit   = errors.New("contract state commitment exceeds configured limits")
	ErrCommitmentProof   = errors.New("invalid contract state commitment proof")
	ErrCommitmentMissing = errors.New("contract state key is absent from the commitment")
	ErrCommitmentPresent = errors.New("contract state key is present in the commitment")
)

type CommitmentHash [sha256.Size]byte

type CommitmentLimits struct {
	MaxLeaves     uint32
	MaxDepth      uint8
	MaxValueBytes uint32
}

func (limits CommitmentLimits) validate() error {
	if limits.MaxLeaves == 0 {
		return fmt.Errorf("contract state commitment MaxLeaves must be positive")
	}
	if limits.MaxDepth == 0 || limits.MaxDepth > 31 {
		return fmt.Errorf("contract state commitment MaxDepth must be within 1..31")
	}
	if limits.MaxValueBytes == 0 {
		return fmt.Errorf("contract state commitment MaxValueBytes must be positive")
	}
	if limits.MaxLeaves > uint32(1)<<limits.MaxDepth {
		return fmt.Errorf("contract state commitment MaxLeaves exceeds MaxDepth capacity")
	}
	return nil
}

type Commitment struct {
	Algorithm    string
	LayoutHash   LayoutHash
	StoreVersion uint64
	LeafCount    uint32
	Root         CommitmentHash
}

type InclusionProof struct {
	Algorithm    string
	LayoutHash   LayoutHash
	StoreVersion uint64
	LeafCount    uint32
	Index        uint32
	Key          Key
	Value        []byte
	Siblings     []CommitmentHash
}

type AbsenceProof struct {
	Algorithm    string
	LayoutHash   LayoutHash
	StoreVersion uint64
	LeafCount    uint32
	Query        Key
	Before       *InclusionProof
	After        *InclusionProof
}

type commitmentSnapshot struct {
	layoutHash LayoutHash
	version    uint64
	entries    []Entry
}

func BuildCommitment(store *MemoryStore, limits CommitmentLimits) (Commitment, error) {
	snapshot, err := snapshotForCommitment(store, limits)
	if err != nil {
		return Commitment{}, err
	}
	return commitmentFromSnapshot(snapshot, buildCommitmentLevels(snapshot)), nil
}

func BuildInclusionProof(store *MemoryStore, key Key, limits CommitmentLimits) (Commitment, InclusionProof, error) {
	snapshot, err := snapshotForCommitment(store, limits)
	if err != nil {
		return Commitment{}, InclusionProof{}, err
	}
	index, found := findCommitmentEntry(snapshot.entries, key)
	if !found {
		return Commitment{}, InclusionProof{}, ErrCommitmentMissing
	}
	levels := buildCommitmentLevels(snapshot)
	return commitmentFromSnapshot(snapshot, levels), inclusionProofFromSnapshot(snapshot, levels, uint32(index)), nil
}

func BuildAbsenceProof(store *MemoryStore, query Key, limits CommitmentLimits) (Commitment, AbsenceProof, error) {
	snapshot, err := snapshotForCommitment(store, limits)
	if err != nil {
		return Commitment{}, AbsenceProof{}, err
	}
	position, found := findCommitmentEntry(snapshot.entries, query)
	if found {
		return Commitment{}, AbsenceProof{}, ErrCommitmentPresent
	}
	levels := buildCommitmentLevels(snapshot)
	proof := AbsenceProof{
		Algorithm:    CommitmentAlgorithm,
		LayoutHash:   snapshot.layoutHash,
		StoreVersion: snapshot.version,
		LeafCount:    uint32(len(snapshot.entries)),
		Query:        query,
	}
	if position > 0 {
		before := inclusionProofFromSnapshot(snapshot, levels, uint32(position-1))
		proof.Before = &before
	}
	if position < len(snapshot.entries) {
		after := inclusionProofFromSnapshot(snapshot, levels, uint32(position))
		proof.After = &after
	}
	return commitmentFromSnapshot(snapshot, levels), proof, nil
}

func VerifyInclusion(expected Commitment, proof InclusionProof, limits CommitmentLimits) error {
	if err := validateCommitment(expected, limits); err != nil {
		return err
	}
	if proof.Algorithm != CommitmentAlgorithm || proof.Algorithm != expected.Algorithm {
		return fmt.Errorf("%w: unsupported inclusion proof algorithm %q", ErrCommitmentProof, proof.Algorithm)
	}
	if proof.LayoutHash != expected.LayoutHash || proof.StoreVersion != expected.StoreVersion || proof.LeafCount != expected.LeafCount {
		return fmt.Errorf("%w: inclusion proof header does not match commitment", ErrCommitmentProof)
	}
	if proof.LeafCount == 0 || proof.Index >= proof.LeafCount {
		return fmt.Errorf("%w: invalid inclusion index %d for leaf count %d", ErrCommitmentProof, proof.Index, proof.LeafCount)
	}
	if uint32(len(proof.Value)) > limits.MaxValueBytes {
		return fmt.Errorf("%w: proof value exceeds MaxValueBytes", ErrCommitmentLimit)
	}
	depth := commitmentDepth(proof.LeafCount)
	if len(proof.Siblings) != int(depth) {
		return fmt.Errorf("%w: proof depth %d does not match expected %d", ErrCommitmentProof, len(proof.Siblings), depth)
	}

	hash := hashCommitmentLeaf(proof.LayoutHash, proof.StoreVersion, proof.Key, proof.Value)
	position := proof.Index
	for _, sibling := range proof.Siblings {
		if position&1 == 0 {
			hash = hashCommitmentNode(proof.LayoutHash, proof.StoreVersion, hash, sibling)
		} else {
			hash = hashCommitmentNode(proof.LayoutHash, proof.StoreVersion, sibling, hash)
		}
		position >>= 1
	}
	root := hashCommitmentRoot(proof.LayoutHash, proof.StoreVersion, proof.LeafCount, hash)
	if root != expected.Root {
		return fmt.Errorf("%w: inclusion root mismatch", ErrCommitmentProof)
	}
	return nil
}

func VerifyAbsence(expected Commitment, proof AbsenceProof, limits CommitmentLimits) error {
	if err := validateCommitment(expected, limits); err != nil {
		return err
	}
	if proof.Algorithm != CommitmentAlgorithm || proof.Algorithm != expected.Algorithm {
		return fmt.Errorf("%w: unsupported absence proof algorithm %q", ErrCommitmentProof, proof.Algorithm)
	}
	if proof.LayoutHash != expected.LayoutHash || proof.StoreVersion != expected.StoreVersion || proof.LeafCount != expected.LeafCount {
		return fmt.Errorf("%w: absence proof header does not match commitment", ErrCommitmentProof)
	}
	if proof.LeafCount == 0 {
		if proof.Before != nil || proof.After != nil {
			return fmt.Errorf("%w: empty-tree proof contains neighbors", ErrCommitmentProof)
		}
		empty := hashCommitmentEmpty(expected.LayoutHash, expected.StoreVersion)
		root := hashCommitmentRoot(expected.LayoutHash, expected.StoreVersion, 0, empty)
		if root != expected.Root {
			return fmt.Errorf("%w: empty-tree root mismatch", ErrCommitmentProof)
		}
		return nil
	}
	if proof.Before == nil && proof.After == nil {
		return fmt.Errorf("%w: non-empty absence proof has no boundary", ErrCommitmentProof)
	}

	if proof.Before != nil {
		if err := VerifyInclusion(expected, *proof.Before, limits); err != nil {
			return fmt.Errorf("%w: predecessor: %v", ErrCommitmentProof, err)
		}
		if slices.Compare(proof.Before.Key[:], proof.Query[:]) >= 0 {
			return fmt.Errorf("%w: predecessor is not below query", ErrCommitmentProof)
		}
	}
	if proof.After != nil {
		if err := VerifyInclusion(expected, *proof.After, limits); err != nil {
			return fmt.Errorf("%w: successor: %v", ErrCommitmentProof, err)
		}
		if slices.Compare(proof.After.Key[:], proof.Query[:]) <= 0 {
			return fmt.Errorf("%w: successor is not above query", ErrCommitmentProof)
		}
	}

	switch {
	case proof.Before == nil:
		if proof.After.Index != 0 {
			return fmt.Errorf("%w: first successor has index %d", ErrCommitmentProof, proof.After.Index)
		}
	case proof.After == nil:
		if proof.Before.Index+1 != proof.LeafCount {
			return fmt.Errorf("%w: final predecessor index %d does not end tree", ErrCommitmentProof, proof.Before.Index)
		}
	default:
		if proof.Before.Index+1 != proof.After.Index {
			return fmt.Errorf("%w: predecessor and successor are not adjacent", ErrCommitmentProof)
		}
		if slices.Compare(proof.Before.Key[:], proof.After.Key[:]) >= 0 {
			return fmt.Errorf("%w: neighbor keys are not strictly ordered", ErrCommitmentProof)
		}
	}
	return nil
}

func snapshotForCommitment(store *MemoryStore, limits CommitmentLimits) (commitmentSnapshot, error) {
	if err := limits.validate(); err != nil {
		return commitmentSnapshot{}, err
	}
	if store == nil {
		return commitmentSnapshot{}, fmt.Errorf("contract state store is nil")
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if len(store.values) > int(limits.MaxLeaves) {
		return commitmentSnapshot{}, fmt.Errorf("%w: %d leaves exceed MaxLeaves %d", ErrCommitmentLimit, len(store.values), limits.MaxLeaves)
	}
	keys := make([]Key, 0, len(store.values))
	for key := range store.values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		return slices.Compare(keys[left][:], keys[right][:]) < 0
	})
	entries := make([]Entry, len(keys))
	for index, key := range keys {
		value := store.values[key]
		if len(value) > int(limits.MaxValueBytes) {
			return commitmentSnapshot{}, fmt.Errorf("%w: value for %s exceeds MaxValueBytes", ErrCommitmentLimit, key)
		}
		entries[index] = Entry{Key: key, Value: slices.Clone(value)}
	}
	if commitmentDepth(uint32(len(entries))) > limits.MaxDepth {
		return commitmentSnapshot{}, fmt.Errorf("%w: tree depth exceeds MaxDepth", ErrCommitmentLimit)
	}
	return commitmentSnapshot{layoutHash: store.layout.hash, version: store.version, entries: entries}, nil
}

func buildCommitmentLevels(snapshot commitmentSnapshot) [][]CommitmentHash {
	if len(snapshot.entries) == 0 {
		return [][]CommitmentHash{{hashCommitmentEmpty(snapshot.layoutHash, snapshot.version)}}
	}
	width := nextPowerOfTwo(uint32(len(snapshot.entries)))
	leaves := make([]CommitmentHash, int(width))
	for index := uint32(0); index < width; index++ {
		if index < uint32(len(snapshot.entries)) {
			entry := snapshot.entries[index]
			leaves[index] = hashCommitmentLeaf(snapshot.layoutHash, snapshot.version, entry.Key, entry.Value)
		} else {
			leaves[index] = hashCommitmentPadding(snapshot.layoutHash, snapshot.version, index)
		}
	}
	levels := [][]CommitmentHash{leaves}
	for len(levels[len(levels)-1]) > 1 {
		current := levels[len(levels)-1]
		next := make([]CommitmentHash, len(current)/2)
		for index := range next {
			next[index] = hashCommitmentNode(snapshot.layoutHash, snapshot.version, current[index*2], current[index*2+1])
		}
		levels = append(levels, next)
	}
	return levels
}

func commitmentFromSnapshot(snapshot commitmentSnapshot, levels [][]CommitmentHash) Commitment {
	treeRoot := levels[len(levels)-1][0]
	return Commitment{
		Algorithm:    CommitmentAlgorithm,
		LayoutHash:   snapshot.layoutHash,
		StoreVersion: snapshot.version,
		LeafCount:    uint32(len(snapshot.entries)),
		Root:         hashCommitmentRoot(snapshot.layoutHash, snapshot.version, uint32(len(snapshot.entries)), treeRoot),
	}
}

func inclusionProofFromSnapshot(snapshot commitmentSnapshot, levels [][]CommitmentHash, index uint32) InclusionProof {
	entry := snapshot.entries[index]
	proof := InclusionProof{
		Algorithm:    CommitmentAlgorithm,
		LayoutHash:   snapshot.layoutHash,
		StoreVersion: snapshot.version,
		LeafCount:    uint32(len(snapshot.entries)),
		Index:        index,
		Key:          entry.Key,
		Value:        slices.Clone(entry.Value),
		Siblings:     make([]CommitmentHash, 0, len(levels)-1),
	}
	position := index
	for levelIndex := 0; levelIndex+1 < len(levels); levelIndex++ {
		proof.Siblings = append(proof.Siblings, levels[levelIndex][position^1])
		position >>= 1
	}
	return proof
}

func validateCommitment(commitment Commitment, limits CommitmentLimits) error {
	if err := limits.validate(); err != nil {
		return err
	}
	if commitment.Algorithm != CommitmentAlgorithm {
		return fmt.Errorf("%w: unsupported commitment algorithm %q", ErrCommitmentProof, commitment.Algorithm)
	}
	if commitment.LeafCount > limits.MaxLeaves || commitmentDepth(commitment.LeafCount) > limits.MaxDepth {
		return fmt.Errorf("%w: commitment metadata exceeds limits", ErrCommitmentLimit)
	}
	return nil
}

func findCommitmentEntry(entries []Entry, key Key) (int, bool) {
	return slices.BinarySearchFunc(entries, key, func(entry Entry, target Key) int {
		return slices.Compare(entry.Key[:], target[:])
	})
}

func commitmentDepth(leafCount uint32) uint8 {
	if leafCount <= 1 {
		return 0
	}
	var depth uint8
	width := uint32(1)
	for width < leafCount {
		width <<= 1
		depth++
	}
	return depth
}

func nextPowerOfTwo(value uint32) uint32 {
	if value <= 1 {
		return 1
	}
	value--
	value |= value >> 1
	value |= value >> 2
	value |= value >> 4
	value |= value >> 8
	value |= value >> 16
	return value + 1
}

func hashCommitmentEmpty(layoutHash LayoutHash, version uint64) CommitmentHash {
	payload := []byte("fairyc-contract-state-merkle-empty/v1")
	payload = append(payload, layoutHash[:]...)
	payload = appendU64(payload, version)
	return sha256.Sum256(payload)
}

func hashCommitmentPadding(layoutHash LayoutHash, version uint64, index uint32) CommitmentHash {
	payload := []byte("fairyc-contract-state-merkle-padding/v1")
	payload = append(payload, layoutHash[:]...)
	payload = appendU64(payload, version)
	payload = appendU32(payload, index)
	return sha256.Sum256(payload)
}

func hashCommitmentLeaf(layoutHash LayoutHash, version uint64, key Key, value []byte) CommitmentHash {
	payload := []byte("fairyc-contract-state-merkle-leaf/v1")
	payload = append(payload, layoutHash[:]...)
	payload = appendU64(payload, version)
	payload = append(payload, key[:]...)
	payload = appendCanonicalBytes(payload, value)
	return sha256.Sum256(payload)
}

func hashCommitmentNode(layoutHash LayoutHash, version uint64, left, right CommitmentHash) CommitmentHash {
	payload := []byte("fairyc-contract-state-merkle-node/v1")
	payload = append(payload, layoutHash[:]...)
	payload = appendU64(payload, version)
	payload = append(payload, left[:]...)
	payload = append(payload, right[:]...)
	return sha256.Sum256(payload)
}

func hashCommitmentRoot(layoutHash LayoutHash, version uint64, leafCount uint32, treeRoot CommitmentHash) CommitmentHash {
	payload := []byte("fairyc-contract-state-merkle-root/v1")
	payload = append(payload, layoutHash[:]...)
	payload = appendU64(payload, version)
	payload = appendU32(payload, leafCount)
	payload = append(payload, treeRoot[:]...)
	return sha256.Sum256(payload)
}
