package pecoff

import "encoding/binary"

// BuildAMD64ChainLinkProbe emits a closed platform fixture, not compiler output.
// A real SUB/NOP/MOV/ADD/RET function returns 42 through two chained fragments.
// All chain fields have nonzero addends to independent code/metadata anchors.
func BuildAMD64ChainLinkProbe() ([]byte, error) {
	const coffSymbolBytes = 18
	const textAt, pdataAt, xdataAt = 140, 166, 202
	const pdataRelocs, xdataRelocs, symbolsAt = 250, 340, 400
	object := make([]byte, symbolsAt+3*coffSymbolBytes+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 3)
	binary.LittleEndian.PutUint32(object[8:], symbolsAt)
	binary.LittleEndian.PutUint32(object[12:], 3)
	for i, s := range []struct {
		name             string
		size, raw, reloc uint32
		count            uint16
		flags            uint32
	}{
		{".text", 26, textAt, 0, 0, sectionContainsCode | sectionReadable | sectionExecutable | 0x00500000},
		{".pdata", 36, pdataAt, pdataRelocs, 9, sectionInitializedData | sectionReadable | 0x00300000},
		{".xdata", 48, xdataAt, xdataRelocs, 6, sectionInitializedData | sectionReadable | 0x00300000},
	} {
		o := coffFileHeaderBytes + i*coffSectionHeaderBytes
		copy(object[o:], s.name)
		binary.LittleEndian.PutUint32(object[o+16:], s.size)
		binary.LittleEndian.PutUint32(object[o+20:], s.raw)
		binary.LittleEndian.PutUint32(object[o+24:], s.reloc)
		binary.LittleEndian.PutUint16(object[o+32:], s.count)
		binary.LittleEndian.PutUint32(object[o+36:], s.flags)
	}
	copy(object[textAt:], []byte{0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc,
		0x48, 0x83, 0xec, 0x28, 0x90, 0x90, 0x90, 0x90,
		0xb8, 42, 0, 0, 0, 0x48, 0x83, 0xc4, 0x28, 0xc3})
	copy(object[xdataAt+8:], []byte{1, 4, 1, 0, 4, 0x42, 0, 0})
	object[xdataAt+16], object[xdataAt+32] = 0x21, 0x21
	emitReloc := func(at int, site, symbol, addend uint32, raw int) {
		binary.LittleEndian.PutUint32(object[at:], site)
		binary.LittleEndian.PutUint32(object[at+4:], symbol)
		binary.LittleEndian.PutUint16(object[at+8:], ObjectRelocationAMD64Addr32NB)
		binary.LittleEndian.PutUint32(object[raw+int(site):], addend)
	}
	for i, entry := range [][3]uint32{{8, 12, 8}, {12, 16, 16}, {16, 26, 32}} {
		for field, addend := range entry {
			symbol := uint32(0)
			if field == 2 {
				symbol = 2
			}
			emitReloc(pdataRelocs+(i*3+field)*coffRelocationBytes, uint32(i*12+field*4), symbol, addend, pdataAt)
		}
	}
	for i, entry := range [][3]uint32{{8, 12, 8}, {12, 16, 16}} {
		for field, addend := range entry {
			symbol := uint32(0)
			if field == 2 {
				symbol = 2
			}
			emitReloc(xdataRelocs+(i*3+field)*coffRelocationBytes, uint32(20+i*16+field*4), symbol, addend, xdataAt)
		}
	}
	for i, symbol := range []struct {
		name    string
		value   uint32
		section uint16
		kind    uint16
	}{
		{"fy_code", 0, 1, 0}, {"fy_chain", 8, 1, coffSymbolTypeFunction}, {"fy_meta", 0, 3, 0},
	} {
		o := symbolsAt + i*coffSymbolBytes
		copy(object[o:], symbol.name)
		binary.LittleEndian.PutUint32(object[o+8:], symbol.value)
		binary.LittleEndian.PutUint16(object[o+12:], symbol.section)
		binary.LittleEndian.PutUint16(object[o+14:], symbol.kind)
		object[o+16] = coffStorageClassExternal
	}
	binary.LittleEndian.PutUint32(object[symbolsAt+3*coffSymbolBytes:], 4)
	if _, err := InspectObject(object); err != nil {
		return nil, err
	}
	return object, nil
}
