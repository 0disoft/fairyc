package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

var scannerCFunctionHeader = regexp.MustCompile(`(?m)^static [^\n]* fy_fn_([0-9]+)\([^\n]*\) \{\n`)

type scannerCSourceParts struct {
	NonFunctionBytes int
	FunctionBytes    []int
}

func splitScannerCSource(source string, functionCount int) (scannerCSourceParts, error) {
	headers := scannerCFunctionHeader.FindAllStringSubmatchIndex(source, -1)
	if len(headers) != functionCount {
		return scannerCSourceParts{}, fmt.Errorf("C definitions = %d, want %d", len(headers), functionCount)
	}
	if len(headers) == 0 {
		return scannerCSourceParts{}, fmt.Errorf("C definitions missing")
	}
	mainStart := strings.LastIndex(source, "int main(void) {\n")
	if mainStart < 0 || (len(headers) != 0 && mainStart <= headers[len(headers)-1][0]) {
		return scannerCSourceParts{}, fmt.Errorf("C main boundary missing or out of order")
	}
	parts := scannerCSourceParts{NonFunctionBytes: headers[0][0] + len(source) - mainStart, FunctionBytes: make([]int, functionCount)}
	for index, header := range headers {
		number, err := strconv.Atoi(source[header[2]:header[3]])
		if err != nil || number != index {
			return scannerCSourceParts{}, fmt.Errorf("C definition %d has unexpected index %q", index, source[header[2]:header[3]])
		}
		end := mainStart
		if index+1 < len(headers) {
			end = headers[index+1][0]
		}
		parts.FunctionBytes[index] = end - header[0]
	}
	return parts, nil
}

func TestPrivateVectorSDKCandidateScannerGeneratedCAttribution(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_SOURCE_DELTA") != "1" {
		t.Skip("opt-in generated-C source attribution")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	var units [2]cbackend.Unit
	var specializations [2]int
	for index, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, count := scannerBudgetFairyProgram(t, variant)
		specializations[index] = count
		unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		units[index] = unit
	}
	if len(units[0].Functions) != len(units[1].Functions) ||
		!slices.Equal(units[0].ForeignSymbols, units[1].ForeignSymbols) ||
		!slices.Equal(units[0].Features, units[1].Features) ||
		specializations[0] != specializations[1] {
		t.Fatalf("candidate changed counts or external surfaces: functions=%d/%d foreign_equal=%t features_equal=%t specializations=%v",
			len(units[0].Functions), len(units[1].Functions), slices.Equal(units[0].ForeignSymbols, units[1].ForeignSymbols),
			slices.Equal(units[0].Features, units[1].Features), specializations)
	}
	common := 0
	for common < len(units[0].Functions) && units[0].Functions[common] == units[1].Functions[common] {
		common++
	}
	if common <= 29 || units[0].Functions[29] != "$p1::scanner::scan" {
		t.Fatalf("scanner source-function prefix changed: common=%d", common)
	}
	var parts [2]scannerCSourceParts
	for index, unit := range units {
		var err error
		parts[index], err = splitScannerCSource(unit.Source, len(unit.Functions))
		if err != nil {
			t.Fatal(err)
		}
	}
	type row struct {
		Index          int    `json:"index"`
		Name           string `json:"name"`
		IndexedBytes   int    `json:"indexed_bytes"`
		CandidateBytes int    `json:"candidate_bytes"`
		DeltaBytes     int    `json:"delta_bytes"`
	}
	changed := make([]row, 0)
	for index := 0; index < common; index++ {
		name := units[0].Functions[index]
		before, after := parts[0].FunctionBytes[index], parts[1].FunctionBytes[index]
		if before != after {
			changed = append(changed, row{index, name, before, after, after - before})
		}
	}
	unmatchedIndexed, unmatchedCandidate := 0, 0
	for index := common; index < len(units[0].Functions); index++ {
		unmatchedIndexed += parts[0].FunctionBytes[index]
		unmatchedCandidate += parts[1].FunctionBytes[index]
	}
	accounted := parts[1].NonFunctionBytes - parts[0].NonFunctionBytes + unmatchedCandidate - unmatchedIndexed
	for _, function := range changed {
		accounted += function.DeltaBytes
	}
	if accounted != len(units[1].Source)-len(units[0].Source) {
		t.Fatalf("source delta accounting = %d, want %d", accounted, len(units[1].Source)-len(units[0].Source))
	}
	report := map[string]any{
		"schema":                         "fairyc.vector-lexer-generated-c-attribution/v1",
		"target":                         units[0].Target,
		"indexed_scanner_sha256":         fmt.Sprintf("%x", sha256.Sum256([]byte(variants[0].scanner))),
		"candidate_scanner_sha256":       fmt.Sprintf("%x", sha256.Sum256([]byte(variants[4].scanner))),
		"main_sha256":                    fmt.Sprintf("%x", sha256.Sum256([]byte(realScannerExecutionMain))),
		"indexed_c_sha256":               fmt.Sprintf("%x", sha256.Sum256([]byte(units[0].Source))),
		"candidate_c_sha256":             fmt.Sprintf("%x", sha256.Sum256([]byte(units[1].Source))),
		"indexed_c_bytes":                len(units[0].Source),
		"candidate_c_bytes":              len(units[1].Source),
		"function_count":                 len(units[0].Functions),
		"common_function_prefix":         common,
		"unmatched_function_count":       len(units[0].Functions) - common,
		"unmatched_function_delta_bytes": unmatchedCandidate - unmatchedIndexed,
		"specialization_count":           specializations[0],
		"nonfunction_delta_bytes":        parts[1].NonFunctionBytes - parts[0].NonFunctionBytes,
		"changed_functions":              changed,
		"scope":                          "Generated C source spans include each function header through the next header, or main for the last function. Only the identical function-name prefix is attributed by source name; generic identities differ between forms and the remaining suffix is grouped. The nonfunction span includes the preamble, prototypes and main. These are C text bytes, not linked machine-code size or a build-time verdict.",
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_SOURCE_DELTA " + string(encoded))
}
