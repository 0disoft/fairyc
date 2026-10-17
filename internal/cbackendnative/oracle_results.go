package cbackendnative

import (
	"encoding/binary"
	"fmt"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cbackendfixture"
	"slices"
	"strings"
)

func expectedDropSiteIDs(fixture cbackendfixture.EmittedCase) ([]uint32, error) {
	result := make([]uint32, 0, len(fixture.DropTrace))
	for _, event := range fixture.DropTrace {
		found := false
		for _, site := range fixture.TraceUnit.DropSites {
			if site.ResourceType != event.ResourceType.String() || site.Package != event.Binding.Package || site.Source != event.Binding.Source ||
				site.StartByte != event.Binding.StartByte || site.EndByte != event.Binding.EndByte {
				continue
			}
			result = append(result, site.ID)
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("drop trace for %s has no C site for %s at %s:%d-%d", fixture.ID, event.ResourceType, event.Binding.Source, event.Binding.StartByte, event.Binding.EndByte)
		}
	}
	return result, nil
}

func decodeDropTrace(contents []byte) ([]uint32, error) {
	if len(contents)%4 != 0 {
		return nil, fmt.Errorf("record stream has %d bytes, want a multiple of 4", len(contents))
	}
	result := make([]uint32, 0, len(contents)/4)
	for offset := 0; offset < len(contents); offset += 4 {
		result = append(result, binary.LittleEndian.Uint32(contents[offset:offset+4]))
	}
	return result, nil
}

func expectedExitCode(fixture cbackendfixture.EmittedCase) (int, error) {
	switch fixture.Outcome {
	case "return":
		if fixture.Value < 0 || fixture.Value >= trapExitBase {
			return 0, fmt.Errorf("native process-status oracle cannot represent return %d in %s", fixture.Value, fixture.ID)
		}
		return int(fixture.Value), nil
	case "trap":
		code, ok := cbackend.TrapCode(fixture.TrapCategory)
		if !ok {
			return 0, fmt.Errorf("unknown trap category %q in %s", fixture.TrapCategory, fixture.ID)
		}
		return trapExitBase + int(code), nil
	default:
		return 0, fmt.Errorf("unsupported native outcome %q in %s", fixture.Outcome, fixture.ID)
	}
}

func summarizeFailures(failures []cbackendfixture.Failure) string {
	if len(failures) == 0 {
		return "emitted case count mismatch"
	}
	values := make([]string, 0, len(failures))
	for _, failure := range failures {
		values = append(values, failure.CaseID+": "+failure.Detail)
	}
	slices.Sort(values)
	return strings.Join(values, "; ")
}
