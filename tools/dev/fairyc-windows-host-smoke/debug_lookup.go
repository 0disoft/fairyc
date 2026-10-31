package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
)

type retainedDebugLookup struct {
	Schema         string `json:"schema"`
	ImageSHA256    string `json:"imageSHA256"`
	PDBSHA256      string `json:"pdbSHA256"`
	SourceRevision string `json:"sourceRevision"`
	Profile        string `json:"profile"`
	RVA            uint32 `json:"rva"`
	Symbol         string `json:"symbol"`
	SourceFile     string `json:"sourceFile"`
	SourceSHA256   string `json:"sourceSHA256"`
	Line           uint32 `json:"line"`
	LineText       string `json:"lineText"`
}

func lookupRetainedWindowsDebug(store, key string, rva uint32) (retainedDebugLookup, error) {
	var result retainedDebugLookup
	manifest, files, err := loadWindowsDebugBundle(store, key)
	if err != nil {
		return result, err
	}
	receipt, err := symbolicateWindowsProgram(files["program.exe"], files["program.pdb"], rva)
	if err != nil {
		return result, err
	}
	if receipt.Displacement != 0 {
		return result, fmt.Errorf("lookup requires an exact function start")
	}
	base := filepath.Base(strings.ReplaceAll(receipt.SourceFile, "\\", "/"))
	if base != "unit.c" && base != "runtime.c" {
		return result, fmt.Errorf("DbgHelp source is not a retained generated-C input")
	}
	lines := strings.Split(string(files[base]), "\n")
	if receipt.Line == 0 || uint64(receipt.Line) > uint64(len(lines)) {
		return result, fmt.Errorf("DbgHelp line is outside the retained source")
	}
	return retainedDebugLookup{
		Schema: "fairyc.windows-debug-lookup/v1", ImageSHA256: key, PDBSHA256: manifest.Files["program.pdb"],
		SourceRevision: manifest.SourceRevision, Profile: manifest.Profile, RVA: rva, Symbol: receipt.Name,
		SourceFile: base, SourceSHA256: manifest.Files[base], Line: receipt.Line, LineText: lines[receipt.Line-1],
	}, nil
}

// Private developer mode; no product CLI flag or network symbol server is added.
func runRetainedDebugLookup(arguments []string, output io.Writer) error {
	flags := flag.NewFlagSet("--debug-lookup", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	store := flags.String("store", "", "local debug-bundle root")
	key := flags.String("image-sha", "", "exact PE SHA-256")
	rva := flags.Uint64("rva", 0, "exact function start RVA, decimal or 0x hexadecimal")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || *store == "" || !validDebugDigest(*key, 32) || *rva == 0 || *rva > math.MaxUint32 {
		return fmt.Errorf("debug lookup requires --store, full --image-sha and nonzero 32-bit --rva")
	}
	result, err := lookupRetainedWindowsDebug(*store, *key, uint32(*rva))
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(result)
}
