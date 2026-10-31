package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const mixedStackSchema = "fairyc.windows-mixed-stack/v1"

type mixedStackEvent struct {
	Kind   string  `json:"kind"`
	Values []int64 `json:"values"`
}

func parseMixedStackTrace(output string) ([]mixedStackEvent, error) {
	var events []mixedStackEvent
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "FY_MIXED" || len(events) >= 8 {
			return nil, fmt.Errorf("invalid mixed-stack trace line %q", line)
		}
		event := mixedStackEvent{Kind: fields[1]}
		for _, field := range fields[2:] {
			value, err := strconv.ParseInt(field, 10, 64)
			if err != nil {
				return nil, err
			}
			event.Values = append(event.Values, value)
		}
		events = append(events, event)
	}
	return events, nil
}

func validateMixedStackObservation(row mixedStackObservation) error {
	fail := func() error { return fmt.Errorf("invalid mixed-stack observation %s/%s", row.Case, row.Profile) }
	if row.Profile != "low" && row.Profile != "optimized" || row.Toolchain == "" || row.RuntimeFunctions <= 0 {
		return fail()
	}
	for _, digest := range []string{row.FairySourceSHA256, row.GeneratedCSHA256, row.CompanionSHA256, row.ImageSHA256, row.CallerSHA256, row.CalleeSHA256} {
		data, err := hex.DecodeString(digest)
		if err != nil || len(data) != 32 || digest != strings.ToLower(digest) {
			return fail()
		}
	}
	success := row.Case == "walk-and-c-boundary-catch"
	early := row.Case == "callback-trap-no-unwind"
	if !success && !early && row.Case != "reject-handler-address" && row.Case != "reject-handler-data" && row.Case != "reject-caller-stack" {
		return fail()
	}
	if success {
		if row.Exit != 42 || row.TrapCode != 0 || row.TrapCategory != "" || row.TrapControlHex != "" {
			return fail()
		}
	} else {
		control, err := hex.DecodeString(row.TrapControlHex)
		if err != nil || len(control) == 0 || control[0] != 6 || row.Exit != 134 || row.TrapCode != 6 || row.TrapCategory != "assertion-failed" {
			return fail()
		}
	}
	kinds := []string{"callback", "frame", "frame", "frame", "handler", "result"}
	if success {
		kinds = []string{"callback", "frame", "frame", "frame", "handler", "frame", "result"}
	}
	if early {
		kinds = []string{"callback"}
	}
	if len(row.Trace) != len(kinds) {
		return fail()
	}
	stage := int64(0)
	var previousSP int64
	for i, event := range row.Trace {
		v := event.Values
		if event.Kind != kinds[i] {
			return fail()
		}
		switch event.Kind {
		case "callback":
			input := int64(40)
			if early {
				input = 13
			}
			if len(v) != 3 || v[0] <= 0 || v[1] != input || v[2] != 1 {
				return fail()
			}
		case "frame":
			if len(v) != 3 || v[0] != stage || v[1] <= 0 || v[2] <= previousSP {
				return fail()
			}
			stage++
			previousSP = v[2]
		case "handler":
			if len(v) != 9 {
				return fail()
			}
			for _, value := range v {
				if value <= 0 {
					return fail()
				}
			}
			handlerDelta, dataDelta, stackDelta := int64(0), int64(0), int64(0)
			switch row.Case {
			case "reject-handler-address":
				handlerDelta = 1
			case "reject-handler-data":
				dataDelta = 4
			case "reject-caller-stack":
				stackDelta = 8
			}
			if v[1]-v[0] != handlerDelta || v[3]-v[2] != dataDelta || v[4] != v[5] || v[7]-v[6] != stackDelta {
				return fail()
			}
		case "result":
			result := int64(-99)
			if success {
				result = 41
			}
			if len(v) != 4 || v[0] != result || v[1] != 1 || v[2] != 1 || v[3] != 1 {
				return fail()
			}
		}
	}
	return nil
}

// Shared by the native test and the existing CI smoke: no second native run.
func writeMixedStackEvidence(rows []mixedStackObservation) error {
	if len(rows) != 10 {
		return fmt.Errorf("mixed-stack receipt requires ten executions")
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		if err := validateMixedStackObservation(row); err != nil {
			return err
		}
		key := row.Case + "/" + row.Profile
		if seen[key] {
			return fmt.Errorf("duplicate mixed-stack case %s", key)
		}
		seen[key] = true
	}
	path := os.Getenv("FAIRYC_MIXED_STACK_EVIDENCE")
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(struct {
		Schema         string                  `json:"schema"`
		SourceRevision string                  `json:"sourceRevision"`
		Passed         bool                    `json:"passed"`
		Rows           []mixedStackObservation `json:"rows"`
	}{mixedStackSchema, os.Getenv("FAIRYC_SOURCE_REVISION"), true, rows}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
