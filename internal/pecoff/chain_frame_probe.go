package pecoff

import "encoding/binary"

// BuildAMD64FrameChainProbe is a closed RBP/XMM6 platform fixture. The primary
// pushes RBP, allocates 64 bytes, establishes RBP=RSP+32 and saves XMM6; two
// secondary fragments share that fixed frame and use real ADDR32NB chain tails.
func BuildAMD64FrameChainProbe() ([]byte, error) {
	original, err := BuildAMD64ChainLinkProbe()
	if err != nil {
		return nil, err
	}
	const textAt = 140
	const pdataAt = 180
	const xdataAt = 216
	const pdataRelocs = 272
	const xdataRelocs = 362
	const symbolsAt = 422
	b := make([]byte, symbolsAt+58)
	copy(b, original[:140])
	copy(b[symbolsAt:], original[400:])
	binary.LittleEndian.PutUint32(b[8:], symbolsAt)
	for i, s := range []struct{ size, raw, relocs uint32 }{{40, textAt, 0}, {36, pdataAt, pdataRelocs}, {56, xdataAt, xdataRelocs}} {
		at := 20 + i*40
		binary.LittleEndian.PutUint32(b[at+16:], s.size)
		binary.LittleEndian.PutUint32(b[at+20:], s.raw)
		binary.LittleEndian.PutUint32(b[at+24:], s.relocs)
	}
	for i := 0; i < 8; i++ {
		b[textAt+i] = 0xcc
	}
	copy(b[textAt+8:], []byte{0x55, 0x48, 0x83, 0xec, 0x40, 0x48, 0x8d, 0x6c, 0x24, 0x20, 0x0f, 0x29, 0x74, 0x24, 0x20, 0x90, 0xb8, 42, 0, 0, 0, 0x0f, 0x28, 0x74, 0x24, 0x20, 0x48, 0x83, 0xc4, 0x40, 0x5d, 0xc3})
	copy(b[xdataAt+8:], []byte{1, 15, 5, 0x25, 15, 0x68, 2, 0, 10, 3, 5, 0x72, 1, 0x50, 0, 0})
	copy(b[xdataAt+24:], []byte{0x21, 0, 0, 0x25})
	copy(b[xdataAt+40:], []byte{0x21, 0, 0, 0x25})
	emit := func(at int, site, symbol, addend uint32, raw int) {
		binary.LittleEndian.PutUint32(b[at:], site)
		binary.LittleEndian.PutUint32(b[at+4:], symbol)
		binary.LittleEndian.PutUint16(b[at+8:], ObjectRelocationAMD64Addr32NB)
		binary.LittleEndian.PutUint32(b[raw+int(site):], addend)
	}
	for i, entry := range [][3]uint32{{8, 23, 8}, {23, 24, 24}, {24, 40, 40}} {
		for j, value := range entry {
			symbol := uint32(0)
			if j == 2 {
				symbol = 2
			}
			emit(pdataRelocs+(i*3+j)*10, uint32(i*12+j*4), symbol, value, pdataAt)
		}
	}
	for i, entry := range [][3]uint32{{8, 23, 8}, {23, 24, 24}} {
		for j, value := range entry {
			symbol := uint32(0)
			if j == 2 {
				symbol = 2
			}
			emit(xdataRelocs+(i*3+j)*10, uint32(28+i*16+j*4), symbol, value, xdataAt)
		}
	}
	if _, err := InspectObject(b); err != nil {
		return nil, err
	}
	return b, nil
}
