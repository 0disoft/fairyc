package pecoff

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"fmt"
	"slices"
)

const (
	comdatRelocation16Bytes = 2
	comdatRelocation32Bytes = 4
	comdatRelocation64Bytes = 8
)

type generatedCOMDATInterval struct{ start, end uint32 }

func generatedCOMDATRelocationWidth(kind uint16) (uint32, bool) {
	switch kind {
	case ObjectRelocationAMD64Addr64:
		return comdatRelocation64Bytes, true
	case ObjectRelocationAMD64Section:
		return comdatRelocation16Bytes, true
	case ObjectRelocationAMD64Addr32, ObjectRelocationAMD64Addr32NB, ObjectRelocationAMD64Rel32,
		ObjectRelocationAMD64Rel32_1, ObjectRelocationAMD64Rel32_2, ObjectRelocationAMD64Rel32_3,
		ObjectRelocationAMD64Rel32_4, ObjectRelocationAMD64Rel32_5, ObjectRelocationAMD64SecRel:
		return comdatRelocation32Bytes, true
	default:
		return 0, false
	}
}

func generatedCOMDATRelocationFields(relocations []ObjectRelocation, size int) ([]generatedCOMDATInterval, error) {
	var fields []generatedCOMDATInterval
	for _, relocation := range relocations {
		if relocation.Type == ObjectRelocationAMD64Absolute {
			continue
		}
		width, measured := generatedCOMDATRelocationWidth(relocation.Type)
		if !measured {
			return nil, fmt.Errorf("has unmeasured relocation %#x", relocation.Type)
		}
		start := relocation.VirtualAddress
		if uint64(start)+uint64(width) > uint64(size) {
			return nil, fmt.Errorf("relocation exceeds COMDAT bytes")
		}
		fields = append(fields, generatedCOMDATInterval{start, start + width})
	}
	slices.SortFunc(fields, func(a, b generatedCOMDATInterval) int { return cmp.Compare(a.start, b.start) })
	var previous uint32
	for _, field := range fields {
		if field.start < previous {
			return nil, fmt.Errorf("has overlapping relocation fields")
		}
		previous = field.end
	}
	return fields, nil
}

// Fields have already passed range and overlap validation. Hash zero bytes in
// place of relocations to preserve the original masked digest without copying
// or clearing either input. Only the small, local hash state is mutated.
func compareGeneratedCOMDATContent(expected, actual []byte, fields []generatedCOMDATInterval) (int, int, [sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	var zeros [comdatRelocation64Bytes]byte
	hasher := sha256.New()
	var previous uint32
	relocated := 0
	matched := len(expected) == len(actual)
	for _, field := range fields {
		matched = matched && bytes.Equal(expected[previous:field.start], actual[previous:field.start])
		hasher.Write(expected[previous:field.start])
		hasher.Write(zeros[:field.end-field.start])
		relocated += int(field.end - field.start)
		previous = field.end
	}
	compared := len(expected) - relocated
	if !matched || compared == 0 || !bytes.Equal(expected[previous:], actual[previous:]) {
		return 0, 0, digest, fmt.Errorf("non-relocated COMDAT bytes differ or are empty")
	}
	hasher.Write(expected[previous:])
	hasher.Sum(digest[:0])
	return compared, relocated, digest, nil
}
