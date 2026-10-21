package cdriver

import (
	"encoding/binary"
	"strings"
	"testing"
)

func locationReceipt(source string, start, end uint64) string {
	data := make([]byte, 20+len(source))
	data[0], data[1] = 3, 1
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(source)))
	binary.LittleEndian.PutUint64(data[4:12], start)
	binary.LittleEndian.PutUint64(data[12:20], end)
	copy(data[20:], source)
	return string(data)
}

func TestTrapLocationReceipt(t *testing.T) {
	for _, source := range []string{"src/한글.fy", strings.Repeat("s", 256)} {
		outcome, err := classifyNativeOutcome("test", 131, "out", "err", locationReceipt(source, 17, 29))
		if err != nil || outcome.TrapCategory != "division-by-zero" || outcome.TrapLocation == nil ||
			*outcome.TrapLocation != (TrapLocation{Source: source, StartByte: 17, EndByte: 29}) ||
			outcome.Stdout != "out" || outcome.Stderr != "err" {
			t.Fatalf("location outcome: %+v, %v", outcome, err)
		}
	}
	valid := locationReceipt("main.fy", 17, 29)
	for _, invalid := range []string{
		valid[:19], valid[:len(valid)-1], valid + "x", valid + valid,
		valid[:1] + "\x02" + valid[2:], locationReceipt("main.fy", 29, 17),
		locationReceipt("", 0, 0), locationReceipt(strings.Repeat("s", 257), 0, 0),
		locationReceipt("bad\x00.fy", 0, 0), locationReceipt("bad\xff.fy", 0, 0),
	} {
		if _, _, err := decodeTrapControl(131, invalid); err == nil {
			t.Fatalf("accepted invalid receipt %q", invalid)
		}
	}
	if _, _, err := decodeTrapControl(130, valid); err == nil {
		t.Fatal("accepted mismatched exit status")
	}
}
