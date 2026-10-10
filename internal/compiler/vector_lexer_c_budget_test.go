package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type scannerBudgetImage struct {
	name            string
	artifact        cdriver.WindowsHostArtifact
	sourceBytes     int
	specializations int
	functions       []string
}

// One execution per lane proves the independent images can be compared.
// Timing from this smoke is diagnostic only; it cannot decide the 8% budget.
func TestPrivateVectorSDKCandidateIndependentCScannerBudgetSmoke(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_SMOKE") != "1" {
		t.Skip("opt-in three-way scanner budget smoke")
	}
	storageMode := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_BOTH_STORAGE")
	if storageMode != "" && storageMode != "1" {
		t.Fatalf("unknown C scanner storage mode %q", storageMode)
	}
	bothStorage := storageMode == "1"
	cpuMode := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_CPU_TIME")
	if cpuMode != "" && cpuMode != "1" {
		t.Fatalf("unknown scanner CPU timing mode %q", cpuMode)
	}
	cpuTime := cpuMode == "1"
	unwindSizes := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_UNWIND_SIZES") == "1"
	unwindOwners := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_UNWIND_OWNERS") == "1"
	captureMap := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_LINK_MAP") == "1" || unwindSizes || unwindOwners
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const sample = "let n=12345; /*open"
	const rounds = 1000
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if perRound <= len(sample) || len(lexed.Diagnostics) == 0 {
		t.Fatal("scanner sample lost cooked or diagnostic coverage")
	}
	cSource := scannerCBenchmarkSourceWithStorageAndTiming(t, false, cpuTime)
	var cArtifact cdriver.WindowsHostArtifact
	if captureMap {
		cArtifact, err = cdriver.BuildWindowsMSVCProbeWithMap(toolchain, cdriver.OptimizedProfile, cSource)
	} else {
		cArtifact, err = cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, cSource)
	}
	if err != nil {
		t.Fatal(err)
	}
	images := []scannerBudgetImage{{name: "independent C", artifact: cArtifact, sourceBytes: len(cSource)}}
	if bothStorage {
		geometricSource := scannerCBenchmarkSourceWithStorageAndTiming(t, true, cpuTime)
		var geometricArtifact cdriver.WindowsHostArtifact
		var buildErr error
		if captureMap {
			geometricArtifact, buildErr = cdriver.BuildWindowsMSVCProbeWithMap(toolchain, cdriver.OptimizedProfile, geometricSource)
		} else {
			geometricArtifact, buildErr = cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, geometricSource)
		}
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		images = append(images, scannerBudgetImage{
			name: "geometric C", artifact: geometricArtifact, sourceBytes: len(geometricSource),
		})
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	if cpuTime {
		support = append([]byte("#define FC_SCANNER_CPU_TIME\n"), support...)
	}
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, specializations := scannerBudgetFairyProgram(t, variant)
		artifact, unit := buildRealScannerVectorNativeWithStorageAndMap(t, program, entry, cdriver.OptimizedProfile,
			"\n"+string(support)+"\nvoid fc_budget_flush(void) { fflush(stdout); }\n", "", captureMap)
		images = append(images, scannerBudgetImage{
			name: variant.name, artifact: artifact, sourceBytes: len(unit.Source),
			specializations: specializations, functions: unit.Functions,
		})
	}
	if captureMap {
		mapRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			if len(image.artifact.LinkMap) == 0 {
				t.Fatalf("%s missing MSVC link map", image.name)
			}
			symbols := scannerBudgetMapSymbols(image.artifact.LinkMap, image.functions)
			mapRows = append(mapRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"map_bytes":  len(image.artifact.LinkMap),
				"map_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.LinkMap)),
				"symbols":    symbols,
			})
		}
		mapReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-link-map/v1", "images": mapRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_LINK_MAP " + string(mapReport))
	}
	if unwindSizes {
		imageRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			pe, inspectErr := pecoff.InspectExecutable(image.artifact.Bytes)
			if inspectErr != nil {
				t.Fatalf("%s PE inspection: %v", image.name, inspectErr)
			}
			unwind, inspectErr := pecoff.InspectAMD64Unwind(image.artifact.Bytes)
			if inspectErr != nil {
				t.Fatalf("%s unwind inspection: %v", image.name, inspectErr)
			}
			byBegin := make(map[uint32]pecoff.RuntimeFunction, len(unwind.Functions))
			var coveredBytes uint64
			for _, function := range unwind.Functions {
				byBegin[function.BeginRVA] = function
				coveredBytes += uint64(function.EndRVA - function.BeginRVA)
			}
			mapped := make([]map[string]any, 0)
			for _, symbol := range scannerBudgetMapSymbols(image.artifact.LinkMap, image.functions) {
				if symbol.PreferredAddress < pe.ImageBase || symbol.PreferredAddress-pe.ImageBase > uint64(^uint32(0)) {
					continue
				}
				function, found := byBegin[uint32(symbol.PreferredAddress-pe.ImageBase)]
				if !found {
					continue
				}
				mapped = append(mapped, map[string]any{
					"name": symbol.Name, "source_name": symbol.SourceName, "owner": symbol.Owner,
					"begin_rva": function.BeginRVA, "end_rva": function.EndRVA,
					"linked_code_bytes": function.EndRVA - function.BeginRVA,
				})
			}
			var textVirtual, textRaw uint32
			for _, section := range pe.Sections {
				if section.Name == ".text" {
					textVirtual, textRaw = section.VirtualSize, section.RawSize
				}
			}
			if textVirtual == 0 || coveredBytes > uint64(textVirtual) {
				t.Fatalf("%s invalid .pdata/.text accounting: %d/%d", image.name, coveredBytes, textVirtual)
			}
			imageRows = append(imageRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"text_virtual_bytes": textVirtual, "text_raw_bytes": textRaw,
				"runtime_function_count":    len(unwind.Functions),
				"unwind_covered_code_bytes": coveredBytes,
				"mapped_functions":          mapped,
			})
		}
		unwindReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-unwind-sizes/v1", "images": imageRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_UNWIND_SIZES " + string(unwindReport))
	}
	if unwindOwners {
		imageRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			pe, inspectErr := pecoff.InspectExecutable(image.artifact.Bytes)
			if inspectErr != nil {
				t.Fatalf("%s PE inspection: %v", image.name, inspectErr)
			}
			unwind, inspectErr := pecoff.InspectAMD64Unwind(image.artifact.Bytes)
			if inspectErr != nil {
				t.Fatalf("%s unwind inspection: %v", image.name, inspectErr)
			}
			starts := make(map[uint32]map[string]struct{}, len(unwind.Functions))
			for _, function := range unwind.Functions {
				starts[function.BeginRVA] = make(map[string]struct{})
			}
			publicRows := scannerBudgetAllMapSymbols(image.artifact.LinkMap, image.functions)
			for _, symbol := range publicRows {
				if symbol.PreferredAddress < pe.ImageBase || symbol.PreferredAddress-pe.ImageBase > uint64(^uint32(0)) {
					continue
				}
				if owners, found := starts[uint32(symbol.PreferredAddress-pe.ImageBase)]; found {
					owners[strings.ToLower(symbol.Owner)] = struct{}{}
				}
			}
			ownerBytes := make(map[string]uint64)
			ownerFunctions := make(map[string]int)
			var knownBytes, unmatchedBytes, ambiguousBytes uint64
			for _, function := range unwind.Functions {
				bytes := uint64(function.EndRVA - function.BeginRVA)
				owners := starts[function.BeginRVA]
				switch len(owners) {
				case 0:
					unmatchedBytes += bytes
				case 1:
					for owner := range owners {
						knownBytes += bytes
						ownerBytes[owner] += bytes
						ownerFunctions[owner]++
					}
				default:
					ambiguousBytes += bytes
				}
			}
			owners := make([]map[string]any, 0, len(ownerBytes))
			for owner, bytes := range ownerBytes {
				owners = append(owners, map[string]any{
					"owner": owner, "linked_code_bytes": bytes, "runtime_functions": ownerFunctions[owner],
				})
			}
			sort.Slice(owners, func(i, j int) bool {
				left, right := owners[i]["linked_code_bytes"].(uint64), owners[j]["linked_code_bytes"].(uint64)
				if left != right {
					return left > right
				}
				return owners[i]["owner"].(string) < owners[j]["owner"].(string)
			})
			imageRows = append(imageRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"runtime_function_count": len(unwind.Functions), "public_map_rows": len(publicRows),
				"unwind_covered_code_bytes": knownBytes + unmatchedBytes + ambiguousBytes,
				"known_code_bytes":          knownBytes, "unmatched_code_bytes": unmatchedBytes,
				"ambiguous_code_bytes": ambiguousBytes, "owners": owners,
			})
		}
		ownerReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-unwind-owners/v1", "images": imageRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_UNWIND_OWNERS " + string(ownerReport))
	}
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_SECTIONS") == "1" {
		sectionRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			pe, inspectErr := pecoff.InspectExecutable(image.artifact.Bytes)
			if inspectErr != nil {
				t.Fatalf("%s PE inspection: %v", image.name, inspectErr)
			}
			sections := make([]map[string]any, 0, len(pe.Sections))
			accounted := uint64(pe.SizeOfHeaders)
			for _, section := range pe.Sections {
				sections = append(sections, map[string]any{
					"name": section.Name, "raw_bytes": section.RawSize,
					"virtual_bytes": section.VirtualSize, "raw_offset": section.RawOffset,
				})
				accounted += uint64(section.RawSize)
			}
			if accounted > uint64(len(image.artifact.Bytes)) {
				t.Fatalf("%s PE section bytes exceed image: %d > %d", image.name, accounted, len(image.artifact.Bytes))
			}
			sectionRows = append(sectionRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"exe_sha256":   fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
				"header_bytes": pe.SizeOfHeaders, "file_alignment": pe.FileAlignment,
				"sections": sections, "unattributed_bytes": uint64(len(image.artifact.Bytes)) - accounted,
			})
		}
		sectionReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-sections/v1", "images": sectionRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_SECTIONS " + string(sectionReport))
	}
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_OBJECT_SECTIONS") == "1" {
		objectRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			objects := make([]map[string]any, 0, len(image.artifact.Objects))
			for _, object := range image.artifact.Objects {
				coff, inspectErr := pecoff.InspectObject(object.Bytes)
				if inspectErr != nil {
					t.Fatalf("%s/%s COFF inspection: %v", image.name, object.Name, inspectErr)
				}
				var codeBytes, otherBytes uint64
				for _, section := range coff.Sections {
					if section.Characteristics&0x20 != 0 { // IMAGE_SCN_CNT_CODE
						codeBytes += uint64(section.RawSize)
					} else {
						otherBytes += uint64(section.RawSize)
					}
				}
				objects = append(objects, map[string]any{
					"name": object.Name, "object_bytes": len(object.Bytes),
					"code_raw_bytes": codeBytes, "other_raw_bytes": otherBytes,
					"sections": len(coff.Sections), "relocations": coff.Relocations,
				})
			}
			objectRows = append(objectRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
				"objects":    objects,
			})
		}
		objectReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-object-sections/v1", "images": objectRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_OBJECT_SECTIONS " + string(objectReport))
	}
	fullSourceMap := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_FULL_SOURCE_MAP") == "1"
	sourceMap := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_SOURCE_MAP") == "1" || fullSourceMap
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_FUNCTIONS") == "1" || sourceMap {
		imageRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			objectRows := make([]map[string]any, 0, len(image.artifact.Objects))
			for _, object := range image.artifact.Objects {
				coff, inspectErr := pecoff.InspectObject(object.Bytes)
				if inspectErr != nil {
					t.Fatalf("%s/%s COFF inspection: %v", image.name, object.Name, inspectErr)
				}
				spans, codeBytes, attributedBytes := scannerCOFFFunctionSpans(coff)
				functionCount := len(spans)
				if sourceMap && object.Name == "unit.obj" {
					for i := range spans {
						indexText, ok := strings.CutPrefix(spans[i].Name, "fy_fn_")
						if !ok {
							continue
						}
						index, parseErr := strconv.Atoi(indexText)
						if parseErr != nil || index < 0 || index >= len(image.functions) {
							t.Fatalf("%s/%s function symbol %q has no emitted source name", image.name, object.Name, spans[i].Name)
						}
						spans[i].SourceName = image.functions[index]
					}
				}
				top := 20
				if fullSourceMap {
					top = len(spans)
				}
				if len(spans) > top {
					spans = spans[:top]
				}
				objectRows = append(objectRows, map[string]any{
					"name": object.Name, "code_raw_bytes": codeBytes,
					"attributed_raw_bytes":   attributedBytes,
					"unattributed_raw_bytes": codeBytes - attributedBytes,
					"function_count":         functionCount,
					"top_functions":          spans,
				})
			}
			imageRows = append(imageRows, map[string]any{
				"lane": image.name, "exe_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
				"objects": objectRows,
			})
		}
		schema := "fairyc.vector-lexer-c-budget-functions/v1"
		marker := "VECTOR_LEXER_C_BUDGET_FUNCTIONS "
		if sourceMap {
			schema = "fairyc.vector-lexer-c-budget-function-source-map/v1"
			marker = "VECTOR_LEXER_C_BUDGET_SOURCE_MAP "
		}
		if fullSourceMap {
			schema = "fairyc.vector-lexer-c-budget-full-function-source-map/v1"
			marker = "VECTOR_LEXER_C_BUDGET_FULL_SOURCE_MAP "
		}
		functionReport, marshalErr := json.Marshal(map[string]any{"schema": schema, "images": imageRows})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log(marker + string(functionReport))
	}
	reporter := ""
	rows := make([]map[string]any, 0, len(images))
	timingFields := 4
	if cpuTime {
		timingFields++
	}
	for _, image := range images {
		outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image.artifact, strings.NewReader("12345 1000\n"))
		if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
			t.Fatalf("%s execution: %+v %v", image.name, outcome, err)
		}
		fields := strings.Fields(outcome.Stdout)
		if len(fields) < timingFields+1 {
			t.Fatalf("%s missing timing fields: %q", image.name, outcome.Stdout)
		}
		prefix := strings.Join(fields[:len(fields)-timingFields], " ")
		if reporter == "" {
			reporter = prefix
		} else if prefix != reporter {
			t.Fatalf("%s reporter differs from independent C", image.name)
		}
		last := fields[len(fields)-timingFields:]
		completed, completedErr := strconv.ParseUint(last[0], 10, 32)
		ticks, ticksErr := strconv.ParseUint(last[1], 10, 64)
		frequency, frequencyErr := strconv.ParseUint(last[2], 10, 64)
		total, totalErr := strconv.ParseUint(last[timingFields-1], 10, 64)
		if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
			completed != rounds || ticks == 0 || frequency == 0 || total != uint64(perRound*rounds) {
			t.Fatalf("%s timing fields=%v errors=%v,%v,%v,%v", image.name, last,
				completedErr, ticksErr, frequencyErr, totalErr)
		}
		row := map[string]any{
			"lane": image.name, "exe_bytes": len(image.artifact.Bytes), "source_bytes": image.sourceBytes,
			"specializations": image.specializations, "qpc_ticks": ticks, "qpc_frequency": frequency,
			"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
		}
		if cpuTime {
			cpuTicks, cpuErr := strconv.ParseUint(last[3], 10, 64)
			if cpuErr != nil {
				t.Fatalf("%s CPU ticks %q: %v", image.name, last[3], cpuErr)
			}
			row["cpu_100ns"] = cpuTicks
		}
		rows = append(rows, row)
	}
	smokeSchema := "fairyc.vector-lexer-c-budget-smoke/v1"
	scope := "Three standalone images, one diagnostic execution per lane; no 8% runtime verdict or timing-stability claim."
	if bothStorage {
		smokeSchema = "fairyc.vector-lexer-c-budget-smoke/v2"
		scope = "Four standalone images including both C storage modes; one diagnostic execution per lane, no 8% verdict."
	}
	if cpuTime {
		smokeSchema = "fairyc.vector-lexer-c-budget-smoke/v3"
		scope += " GetThreadTimes records current-thread CPU time in 100ns units; the instrumented image sizes are not the uninstrumented size budget."
	}
	report, err := json.Marshal(map[string]any{
		"schema": smokeSchema, "sample_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"msvc": toolchain.Version, "profile": cdriver.OptimizedProfile.Name,
		"rounds": rounds, "per_round_result": perRound, "lanes": rows,
		"scope": scope,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_C_BUDGET_SMOKE " + string(report))
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_MEASURE") == "1" {
		measureScannerBudgetRotations(t, images, sample, uint64(perRound), cpuTime)
	}
}

type scannerCOFFFunctionSpan struct {
	Name         string `json:"name"`
	SourceName   string `json:"source_name,omitempty"`
	Section      string `json:"section"`
	Offset       uint32 `json:"offset"`
	RawSpanBytes uint32 `json:"raw_span_bytes"`
}

type scannerBudgetMapSymbol struct {
	Name             string `json:"name"`
	SourceName       string `json:"source_name,omitempty"`
	Section          uint64 `json:"section"`
	Offset           uint64 `json:"offset"`
	PreferredAddress uint64 `json:"preferred_address"`
	Owner            string `json:"owner"`
}

// Keep only scanner or generated-function rows; the MSVC map itself may
// contain local temporary paths and unrelated CRT symbols.
func scannerBudgetMapSymbols(linkMap []byte, functions []string) []scannerBudgetMapSymbol {
	return scannerBudgetMapRows(linkMap, functions, false)
}

func scannerBudgetAllMapSymbols(linkMap []byte, functions []string) []scannerBudgetMapSymbol {
	return scannerBudgetMapRows(linkMap, functions, true)
}

func scannerBudgetMapRows(linkMap []byte, functions []string, all bool) []scannerBudgetMapSymbol {
	rows := make([]scannerBudgetMapSymbol, 0)
	for _, line := range strings.Split(strings.ReplaceAll(string(linkMap), "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := fields[1]
		if !all && !strings.HasPrefix(name, "fy_fn_") && !strings.HasPrefix(name, "fc_") {
			continue
		}
		sectionText, offsetText, ok := strings.Cut(fields[0], ":")
		if !ok {
			continue
		}
		section, sectionErr := strconv.ParseUint(sectionText, 16, 16)
		offset, offsetErr := strconv.ParseUint(offsetText, 16, 32)
		address, addressErr := strconv.ParseUint(fields[2], 16, 64)
		if sectionErr != nil || offsetErr != nil || addressErr != nil || address == 0 {
			continue
		}
		row := scannerBudgetMapSymbol{
			Name: name, Section: section, Offset: offset, PreferredAddress: address,
			Owner: filepath.Base(strings.ReplaceAll(fields[len(fields)-1], "\\", "/")),
		}
		if indexText, mapped := strings.CutPrefix(name, "fy_fn_"); mapped {
			if index, parseErr := strconv.Atoi(indexText); parseErr == nil && index >= 0 && index < len(functions) {
				row.SourceName = functions[index]
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PreferredAddress != rows[j].PreferredAddress {
			return rows[i].PreferredAddress < rows[j].PreferredAddress
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// COFF function symbols provide starts, not exact linked function sizes. The
// section span up to the next start includes any alignment padding.
func scannerCOFFFunctionSpans(coff pecoff.ObjectReport) ([]scannerCOFFFunctionSpan, uint64, uint64) {
	spans := make([]scannerCOFFFunctionSpan, 0)
	var codeBytes, attributedBytes uint64
	for sectionIndex, section := range coff.Sections {
		if section.Characteristics&0x20 == 0 { // IMAGE_SCN_CNT_CODE
			continue
		}
		codeBytes += uint64(section.RawSize)
		starts := make(map[uint32][]string)
		for _, symbol := range coff.SymbolRecords {
			if symbol.SectionNumber != int32(sectionIndex+1) || symbol.Type&0x20 == 0 || symbol.Value >= section.RawSize {
				continue
			}
			starts[symbol.Value] = append(starts[symbol.Value], symbol.Name)
		}
		offsets := make([]uint32, 0, len(starts))
		for offset := range starts {
			offsets = append(offsets, offset)
		}
		sort.Slice(offsets, func(i, j int) bool { return offsets[i] < offsets[j] })
		for i, offset := range offsets {
			end := section.RawSize
			if i+1 < len(offsets) {
				end = offsets[i+1]
			}
			names := starts[offset]
			sort.Strings(names)
			span := end - offset
			attributedBytes += uint64(span)
			spans = append(spans, scannerCOFFFunctionSpan{
				Name: strings.Join(names, " | "), Section: section.Name,
				Offset: offset, RawSpanBytes: span,
			})
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].RawSpanBytes != spans[j].RawSpanBytes {
			return spans[i].RawSpanBytes > spans[j].RawSpanBytes
		}
		return spans[i].Name < spans[j].Name
	})
	return spans, codeBytes, attributedBytes
}

// Twelve fixed, rotating image sets follow one warmup. Raw QPC and optional
// thread CPU samples inform later stability assessment, not an automatic verdict.
func measureScannerBudgetRotations(t *testing.T, images []scannerBudgetImage, sample string, perRound uint64, cpuTime bool) {
	t.Helper()
	if len(images) != 3 && len(images) != 4 {
		t.Fatalf("budget lane count=%d", len(images))
	}
	const rounds = 250000
	const measured = 12
	const seed = 12345
	prefixes := make([]string, len(images))
	rows := make([]map[string]any, 0, measured+1)
	timingFields := 4
	if cpuTime {
		timingFields++
	}
	for triplet := 0; triplet <= measured; triplet++ {
		order := make([]string, len(images))
		lanes := make([]map[string]any, 0, len(images))
		for position := range images {
			index := (triplet + position) % len(images)
			image := images[index]
			order[position] = image.name
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image.artifact,
				strings.NewReader(fmt.Sprintf("%d %d\n", seed, rounds)))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("triplet %d lane %s: %+v %v", triplet, image.name, outcome, err)
			}
			fields := strings.Fields(outcome.Stdout)
			if len(fields) < timingFields+1 {
				t.Fatalf("triplet %d lane %s missing timing: %q", triplet, image.name, outcome.Stdout)
			}
			prefix := strings.Join(fields[:len(fields)-timingFields], " ")
			if prefixes[index] == "" {
				prefixes[index] = prefix
			} else if prefixes[index] != prefix {
				t.Fatalf("triplet %d lane %s reporter changed", triplet, image.name)
			}
			last := fields[len(fields)-timingFields:]
			completed, completedErr := strconv.ParseUint(last[0], 10, 32)
			ticks, ticksErr := strconv.ParseUint(last[1], 10, 64)
			frequency, frequencyErr := strconv.ParseUint(last[2], 10, 64)
			total, totalErr := strconv.ParseUint(last[timingFields-1], 10, 64)
			if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
				completed != rounds || ticks == 0 || frequency == 0 || total != perRound*rounds {
				t.Fatalf("triplet %d lane %s timing=%v errors=%v,%v,%v,%v", triplet, image.name,
					last, completedErr, ticksErr, frequencyErr, totalErr)
			}
			lane := map[string]any{
				"lane": image.name, "qpc_ticks": ticks, "qpc_frequency": frequency,
				"total": total, "completed": completed,
			}
			if cpuTime {
				cpuTicks, cpuErr := strconv.ParseUint(last[3], 10, 64)
				if cpuErr != nil {
					t.Fatalf("triplet %d lane %s CPU ticks %q: %v", triplet, image.name, last[3], cpuErr)
				}
				lane["cpu_100ns"] = cpuTicks
			}
			lanes = append(lanes, lane)
		}
		row := map[string]any{"warmup": triplet == 0, "order": order, "lanes": lanes}
		if len(images) == 4 {
			row["rotation"] = triplet
		} else {
			row["triplet"] = triplet
		}
		rows = append(rows, row)
	}
	for index := 1; index < len(prefixes); index++ {
		if prefixes[index] != prefixes[0] {
			t.Fatalf("%s reporter differs from independent C", images[index].name)
		}
	}
	imageRows := make([]map[string]any, 0, len(images))
	for _, image := range images {
		imageRows = append(imageRows, map[string]any{
			"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
			"exe_sha256":   fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
			"source_bytes": image.sourceBytes, "specializations": image.specializations,
		})
	}
	reportValues := map[string]any{
		"sample_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"go":            runtime.Version(), "msvc": images[0].artifact.Toolchain.Version,
		"gomaxprocs": runtime.GOMAXPROCS(0), "seed": seed,
		"profile":          cdriver.OptimizedProfile.Name,
		"rounds":           rounds,
		"per_round_result": perRound, "images": imageRows, "samples": rows,
		"scope": "Standalone optimized images, one fixed warmup and twelve rotating measured sets; each execution verifies reporter, completed rounds and checksum. QPC excludes process startup and final output. No retries, exclusions or adaptive stopping. Token reporting is outside the timed loop; this sample does not prove a full-language 8% budget.",
	}
	if len(images) == 4 {
		reportValues["schema"] = "fairyc.vector-lexer-c-budget-rotations/v2"
		reportValues["warmup_rotations"] = 1
		reportValues["measured_rotations"] = measured
	} else {
		reportValues["schema"] = "fairyc.vector-lexer-c-budget-triplets/v1"
		reportValues["warmup_triplets"] = 1
		reportValues["measured_triplets"] = measured
		reportValues["scope"] = "Three standalone optimized images, one fixed warmup and twelve rotating measured triplets; each execution verifies reporter, completed rounds and checksum. QPC excludes process startup and final output. No retries, exclusions or adaptive stopping. Token reporting is outside the timed loop; this sample does not prove a full-language 8% budget."
	}
	if cpuTime {
		reportValues["schema"] = "fairyc.vector-lexer-c-budget-rotations/v3"
		reportValues["cpu_unit_ns"] = 100
		reportValues["scope"] = reportValues["scope"].(string) + " GetThreadTimes observes current-thread CPU time around each QPC interval; CPU samples exclude descheduling but may be quantized. The instrumented image sizes cannot decide the uninstrumented size budget."
	}
	report, err := json.Marshal(reportValues)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) == 4 {
		t.Log("VECTOR_LEXER_C_BUDGET_ROTATIONS " + string(report))
	} else {
		t.Log("VECTOR_LEXER_C_BUDGET_TRIPLETS " + string(report))
	}
}

func scannerBudgetFairyProgram(t *testing.T, variant realScannerVectorVariant) (mir.Program, string, int) {
	return scannerBudgetFairyProgramWithMain(t, variant, realScannerExecutionMain)
}

func scannerBudgetFairyProgramWithMain(t *testing.T, variant realScannerVectorVariant, mainSource string) (mir.Program, string, int) {
	return scannerBudgetFairyProgramWithMainTiming(t, variant, mainSource, nil)
}

type scannerBudgetCheckPhases struct {
	PrepareNS  int64 `json:"prepare_ns"`
	LoadNS     int64 `json:"load_ns"`
	MetadataNS int64 `json:"metadata_ns"`
	AnalyzeNS  int64 `json:"analyze_ns"`
	LowerNS    int64 `json:"lower_ns"`
}

func scannerBudgetFairyProgramWithMainTiming(t *testing.T, variant realScannerVectorVariant, mainSource string, phases *scannerBudgetCheckPhases) (mir.Program, string, int) {
	return scannerBudgetFairyProgramWithMainTimingPlan(t, variant, mainSource, phases, nil)
}

func scannerBudgetFairyProgramWithMainTimingPlan(t *testing.T, variant realScannerVectorVariant, mainSource string, phases *scannerBudgetCheckPhases, plan *genericplan.Plan) (mir.Program, string, int) {
	t.Helper()
	var mark time.Time
	if phases != nil {
		mark = time.Now()
	}
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range realScannerSupportFiles() {
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeVectorCacheFile(t, root, name, string(data))
	}
	writeVectorCacheFile(t, root, "src/scanner.fy", variant.scanner)
	writeVectorCacheFile(t, root, "src/main.fy", mainSource)
	if phases != nil {
		phases.PrepareNS = time.Since(mark).Nanoseconds()
		mark = time.Now()
	}
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	if phases != nil {
		phases.LoadNS = time.Since(mark).Nanoseconds()
		mark = time.Now()
	}
	input, inputErr := workspace.SemanticInput()
	coordinates, coordinateErr := workspace.PackageCoordinates()
	owners, ownerErr := workspace.VectorCandidateStorageOwners()
	if inputErr != nil || coordinateErr != nil || ownerErr != nil || len(owners) != 1 {
		t.Fatalf("scanner package authority: %v %v %v owners=%v", inputErr, coordinateErr, ownerErr, owners)
	}
	if phases != nil {
		phases.MetadataNS = time.Since(mark).Nanoseconds()
		mark = time.Now()
	}
	var analysis compiler.GenericAnalysis
	if variant.candidate {
		analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	} else {
		analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	}
	if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
		len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
		t.Fatalf("%s scanner Flow: %v diagnostics=%+v unsupported=%+v", variant.name, err,
			analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
	}
	if plan != nil {
		*plan = analysis.Plan
	}
	if phases != nil {
		phases.AnalyzeNS = time.Since(mark).Nanoseconds()
		mark = time.Now()
	}
	checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !checked.Verified {
		t.Fatalf("%s verified MIR: %v diagnostics=%+v", variant.name, err, checked.Semantic.Diagnostics)
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	if phases != nil {
		phases.LowerNS = time.Since(mark).Nanoseconds()
	}
	return checked.MIR, hir.QualifiedModulePath(rootID, "main") + "::main", len(analysis.Plan.Specializations)
}
