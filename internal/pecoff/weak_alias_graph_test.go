package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestDirectWeakAliasPlan(t *testing.T) {
	chain, err := BuildAMD64WeakChainProbeObject(false)
	if err != nil {
		t.Fatal(err)
	}
	inv, err := InspectWeakAliases(chain)
	if err != nil || len(inv.WeakExternals) != 2 || len(inv.Directives) != 0 {
		t.Fatalf("inventory %+v %v", inv, err)
	}
	p, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: chain}}, "weak.obj", "fy_wa")
	if err != nil || strings.Join(p.Path, " -> ") != "fy_wa -> fy_wb -> fy_fa" || p.Owner != "weak.obj" || p.Selected != "fy_fa" {
		t.Fatalf("plan %+v %v", p, err)
	}
	cycle, err := BuildAMD64WeakChainProbeObject(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: cycle}}, "weak.obj", "fy_wa"); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("accepted cycle: %v", err)
	}
	for _, index := range []uint32{2, 4} {
		strong := append([]byte(nil), chain...)
		table := binary.LittleEndian.Uint32(strong[8:])
		copy(strong[table:], "other_a\x00")
		copy(strong[table+18:], "other_b\x00")
		copy(strong[table+6*18:], "fy_wa\x00\x00\x00")
		if index == 4 {
			copy(strong[table+6*18:], "fy_wb\x00\x00\x00")
		}
		copy(strong[table+7*18:], "other_f\x00")
		// Keep the provider's unused weak records distinct from the test roots.
		copy(strong[table+2*18:], "other_w\x00")
		copy(strong[table+4*18:], "other_x\x00")
		p, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: cycle}, {Name: "strong.obj", Bytes: strong}}, "weak.obj", "fy_wa")
		want := "fy_wa"
		if index == 4 {
			want = "fy_wb"
		}
		if err != nil || p.Selected != want || p.Owner != "strong.obj" {
			t.Fatalf("strong cycle cut: %+v %v", p, err)
		}
	}
}

func TestWeakAliasBoundaries(t *testing.T) {
	for _, text := range []string{"/alternatename:=fallback", "/alternatename:root=", "/alternatename:root=a=b", "/alternatename:root=f\x00"} {
		if _, err := parseWeakDirectives(text); err == nil {
			t.Fatalf("accepted malformed directive %q", text)
		}
	}
	d, err := parseWeakDirectives(` /DEFAULTLIB:LIBCMT "/ALTERNATENAME:root=fallback" /alternatename:middle=end `)
	if err != nil || len(d) != 2 || d[0].Name != "root" || d[0].Fallback != "fallback" {
		t.Fatalf("directive inventory %+v %v", d, err)
	}
	chain, err := BuildAMD64WeakChainProbeObject(false)
	if err != nil {
		t.Fatal(err)
	}
	table := binary.LittleEndian.Uint32(chain[8:])
	library := append([]byte(nil), chain...)
	binary.LittleEndian.PutUint32(library[table+3*18+4:], ObjectWeakSearchLibrary)
	if _, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: library}}, "weak.obj", "fy_wa"); err == nil || !strings.Contains(err.Error(), "archive") {
		t.Fatalf("guessed archive selection: %v", err)
	}
	missing := append([]byte(nil), chain...)
	binary.LittleEndian.PutUint16(missing[table+6*18+12:], 0)
	binary.LittleEndian.PutUint32(missing[table+6*18+8:], 0)
	if _, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: missing}}, "weak.obj", "fy_wa"); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("accepted unresolved terminal: %v", err)
	}
	for _, name := range []string{"../weak.obj", `dir\weak.obj`, `C:weak.obj`} {
		if _, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: name, Bytes: chain}}, name, "fy_wa"); err == nil {
			t.Fatal("accepted nonlocal object identity")
		}
	}
	conflict := append([]byte(nil), chain...)
	copy(conflict[table+4*18:], "fy_wa\x00\x00\x00")
	if _, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: conflict}}, "weak.obj", "fy_wa"); err == nil || !strings.Contains(err.Error(), "contradictory") {
		t.Fatalf("accepted contradictory alternatives: %v", err)
	}
	if _, err := PlanDirectWeakAlias([]WeakAliasObject{{Name: "weak.obj", Bytes: chain}, {Name: "other.obj", Bytes: chain}}, "weak.obj", "fy_wa"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("accepted ambiguous strong definitions: %v", err)
	}
}
