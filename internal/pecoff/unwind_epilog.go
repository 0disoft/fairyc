package pecoff

import "fmt"

// Version-2 epilog descriptors precede ordinary prolog unwind operations. Their
// offsets are measured backwards from the runtime function's end, not its start.
func version2EpilogSlots(codes []byte, prolog, functionSize uint32) (int, error) {
	if len(codes) < 2 || codes[1]&15 != 6 {
		return 0, nil
	}
	size, info := uint32(codes[0]), codes[1]>>4
	if size == 0 || info > 1 {
		return 0, fmt.Errorf("invalid version-2 epilog size/flags")
	}
	var starts []uint32
	add := func(offset uint32) error {
		if offset < size || offset > functionSize || functionSize-offset < prolog {
			return fmt.Errorf("version-2 epilog range exceeds function body")
		}
		start := functionSize - offset
		for _, previous := range starts {
			if start < previous+size && previous < start+size {
				return fmt.Errorf("overlapping version-2 epilog ranges")
			}
		}
		starts = append(starts, start)
		return nil
	}
	if info == 1 {
		if err := add(size); err != nil {
			return 0, err
		}
	}
	slot := 1
	for slot*2 < len(codes) && codes[slot*2+1]&15 == 6 {
		offset := uint32(codes[slot*2]) + uint32(codes[slot*2+1]>>4)*256
		slot++
		if offset == 0 { // Alignment padding before the ordinary unwind operations.
			break
		}
		if err := add(offset); err != nil {
			return 0, err
		}
	}
	if len(starts) == 0 {
		return 0, fmt.Errorf("truncated version-2 epilog descriptor")
	}
	return slot, nil
}
