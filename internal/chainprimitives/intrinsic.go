package chainprimitives

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
)

type IntrinsicID string

const (
	IntrinsicSHA256           IntrinsicID = "crypto.sha256"
	IntrinsicKeccak256        IntrinsicID = "crypto.keccak256"
	IntrinsicBlake2b256       IntrinsicID = "crypto.blake2b256"
	IntrinsicPoseidon2        IntrinsicID = "crypto.poseidon2"
	IntrinsicSecp256k1Verify  IntrinsicID = "crypto.secp256k1.verify"
	IntrinsicSecp256k1Recover IntrinsicID = "crypto.secp256k1.recover"
	IntrinsicEd25519Verify    IntrinsicID = "crypto.ed25519.verify"
	IntrinsicBLS12381Verify   IntrinsicID = "crypto.bls12_381.verify"
)

var knownIntrinsics = map[IntrinsicID]struct{}{
	IntrinsicSHA256: {}, IntrinsicKeccak256: {}, IntrinsicBlake2b256: {}, IntrinsicPoseidon2: {},
	IntrinsicSecp256k1Verify: {}, IntrinsicSecp256k1Recover: {}, IntrinsicEd25519Verify: {},
	IntrinsicBLS12381Verify: {},
}

func KnownIntrinsic(id IntrinsicID) bool {
	_, ok := knownIntrinsics[id]
	return ok
}

// CostRule gives one deterministic target-owned cost formula:
// Base + PerChunk * ceil(inputBytes / ChunkBytes).
type CostRule struct {
	ID            IntrinsicID
	Base          uint64
	PerChunk      uint64
	ChunkBytes    uint32
	MaxInputBytes uint64
}

func (rule CostRule) validate() error {
	if !KnownIntrinsic(rule.ID) || (rule.Base == 0 && rule.PerChunk == 0) || rule.ChunkBytes == 0 || rule.MaxInputBytes == 0 {
		return ErrInvalidCostSchedule
	}
	return nil
}

func (rule CostRule) Cost(inputBytes uint64) (uint64, error) {
	if err := rule.validate(); err != nil {
		return 0, err
	}
	if inputBytes > rule.MaxInputBytes {
		return 0, ErrInvalidCostSchedule
	}
	chunks := inputBytes / uint64(rule.ChunkBytes)
	if inputBytes%uint64(rule.ChunkBytes) != 0 {
		chunks++
	}
	variable, err := MulU64Checked(rule.PerChunk, chunks)
	if err != nil {
		return 0, ErrInvalidCostSchedule
	}
	total, err := AddU64Checked(rule.Base, variable)
	if err != nil {
		return 0, ErrInvalidCostSchedule
	}
	return total, nil
}

// CostSchedule is immutable after construction. Rules are sorted by intrinsic
// ID so its digest and serialized identity never depend on Go map iteration.
type CostSchedule struct {
	target   string
	revision uint32
	rules    []CostRule
}

func NewCostSchedule(target string, revision uint32, rules []CostRule) (CostSchedule, error) {
	if target == "" || revision == 0 || len(rules) == 0 {
		return CostSchedule{}, ErrInvalidCostSchedule
	}
	cloned := append([]CostRule(nil), rules...)
	sort.Slice(cloned, func(left, right int) bool { return cloned[left].ID < cloned[right].ID })
	for index, rule := range cloned {
		if err := rule.validate(); err != nil {
			return CostSchedule{}, err
		}
		if index > 0 && cloned[index-1].ID == rule.ID {
			return CostSchedule{}, ErrInvalidCostSchedule
		}
	}
	return CostSchedule{target: target, revision: revision, rules: cloned}, nil
}

func (schedule CostSchedule) Target() string    { return schedule.target }
func (schedule CostSchedule) Revision() uint32  { return schedule.revision }
func (schedule CostSchedule) Rules() []CostRule { return append([]CostRule(nil), schedule.rules...) }

func (schedule CostSchedule) Rule(id IntrinsicID) (CostRule, bool) {
	index := sort.Search(len(schedule.rules), func(index int) bool { return schedule.rules[index].ID >= id })
	if index >= len(schedule.rules) || schedule.rules[index].ID != id {
		return CostRule{}, false
	}
	return schedule.rules[index], true
}

func (schedule CostSchedule) Cost(id IntrinsicID, inputBytes uint64) (uint64, error) {
	rule, ok := schedule.Rule(id)
	if !ok {
		return 0, ErrUnknownIntrinsic
	}
	return rule.Cost(inputBytes)
}

// Digest returns a canonical SHA-256 identity suitable for target specs, cache
// keys, and deployment receipts. Cost changes therefore cannot drift silently.
func (schedule CostSchedule) Digest() Hash256 {
	hash := sha256.New()
	writeCanonicalField(hash.Write, []byte(schedule.target))
	var scalar [8]byte
	binary.BigEndian.PutUint32(scalar[:4], schedule.revision)
	_, _ = hash.Write(scalar[:4])
	binary.BigEndian.PutUint32(scalar[:4], uint32(len(schedule.rules)))
	_, _ = hash.Write(scalar[:4])
	for _, rule := range schedule.rules {
		writeCanonicalField(hash.Write, []byte(rule.ID))
		binary.BigEndian.PutUint64(scalar[:], rule.Base)
		_, _ = hash.Write(scalar[:])
		binary.BigEndian.PutUint64(scalar[:], rule.PerChunk)
		_, _ = hash.Write(scalar[:])
		binary.BigEndian.PutUint32(scalar[:4], rule.ChunkBytes)
		_, _ = hash.Write(scalar[:4])
		binary.BigEndian.PutUint64(scalar[:], rule.MaxInputBytes)
		_, _ = hash.Write(scalar[:])
	}
	output, err := NewHash256(hash.Sum(nil))
	if err != nil {
		panic("chainprimitives: SHA-256 produced an invalid digest length")
	}
	return output
}

func writeCanonicalField(write func([]byte) (int, error), field []byte) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(field)))
	_, _ = write(length[:])
	_, _ = write(field)
}

// ExecuteReferenceHash is the compiler-side semantic oracle for algorithms
// available in Go's standard library. Unsupported algorithms fail closed rather
// than substituting another digest.
func ExecuteReferenceHash(id IntrinsicID, input []byte) (Hash256, error) {
	if !KnownIntrinsic(id) {
		return Hash256{}, ErrUnknownIntrinsic
	}
	switch id {
	case IntrinsicSHA256:
		sum := sha256.Sum256(input)
		return Hash256{bytes: sum}, nil
	default:
		return Hash256{}, ErrUnsupportedReference
	}
}
