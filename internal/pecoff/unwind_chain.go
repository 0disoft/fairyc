package pecoff

import (
	"encoding/binary"
	"fmt"
)

// This is an inspector resource bound, not a Microsoft ABI depth limit.
const maximumUnwindChainDepth = 64

// Each link is checked with its own function interval, including targets that
// have no separate .pdata row. Walking is iterative and bounded for hostile PE
// inputs. Reusing an unwind RVA on the same path cannot terminate.
func inspectUnwindChain(image []byte, executable Report, fn RuntimeFunction) (int, error) {
	seen := make(map[uint32]bool)
	var frame byte
	for depth := 0; ; depth++ {
		if seen[fn.UnwindInfoRVA] {
			return 0, fmt.Errorf("chained unwind cycle at %#x", fn.UnwindInfoRVA)
		}
		if depth > maximumUnwindChainDepth {
			return 0, fmt.Errorf("chained unwind exceeds inspector depth bound %d", maximumUnwindChainDepth)
		}
		seen[fn.UnwindInfoRVA] = true
		_, flags, version, err := inspectUnwindInfo(image, executable, fn.UnwindInfoRVA, fn.EndRVA-fn.BeginRVA)
		if err != nil {
			return 0, fmt.Errorf("chained unwind target %#x: %w", fn.UnwindInfoRVA, err)
		}
		offset, _, _ := mapImageRVA(executable, fn.UnwindInfoRVA, unwindHeaderBytes)
		header := image[offset : offset+unwindHeaderBytes]
		if depth == 0 {
			frame = header[3]
		} else if header[3] != frame {
			return 0, fmt.Errorf("chained unwind frame register/offset mismatch at %#x", fn.UnwindInfoRVA)
		}
		if flags&unwindFlagChainInfo == 0 {
			return depth, nil
		}
		count := int(header[2])
		codes := image[offset+unwindHeaderBytes : offset+unwindHeaderBytes+count*2]
		first := 0
		if version == 2 {
			first, _ = version2EpilogSlots(codes, uint32(header[1]), fn.EndRVA-fn.BeginRVA)
		}
		for slot := first; slot < count; {
			op, info := codes[slot*2+1]&15, codes[slot*2+1]>>4
			// Secondary records may save into the primary fixed frame, but
			// cannot add pushes/allocations or establish a different frame.
			if op == 0 || op == 1 || op == 2 || op == 3 || op == 10 {
				return 0, fmt.Errorf("chained unwind changes fixed stack/frame at %#x", fn.UnwindInfoRVA)
			}
			if version == 2 && op == 7 {
				slot++
				continue
			}
			consumed, _ := unwindOperationSlots(op, info)
			slot += consumed // inspectUnwindInfo already checked slot widths.
		}
		tail := offset + unwindHeaderBytes + ((count+1)&^1)*2
		fn = RuntimeFunction{
			BeginRVA:      binary.LittleEndian.Uint32(image[tail:]),
			EndRVA:        binary.LittleEndian.Uint32(image[tail+4:]),
			UnwindInfoRVA: binary.LittleEndian.Uint32(image[tail+8:]),
		}
	}
}
