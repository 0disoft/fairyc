//go:build ignore

// Build exact Go-generated C with the existing closed private host ABI.
// This is a verification-only path, not a public compiler link contract.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

func run(source, expected, output string, reserve uint64) error {
	if source == "" || output == "" || len(expected) != 64 {
		return fmt.Errorf("source, exact SHA-256 and output are required")
	}
	wanted, err := hex.DecodeString(expected)
	if err != nil || len(wanted) != sha256.Size {
		return fmt.Errorf("invalid source SHA-256")
	}
	path, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() || stat.Size() < 1 || stat.Size() > 96*1024*1024 {
		return fmt.Errorf("Go C source must be a regular file in 1..96MiB")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 96*1024*1024+1))
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if len(data) == 0 || len(data) > 96*1024*1024 {
		return fmt.Errorf("Go source read exceeds bound")
	}
	actual := sha256.Sum256(data)
	if hex.EncodeToString(actual[:]) != expected {
		return fmt.Errorf("Go C source SHA-256 differs")
	}
	text := string(data)
	if !strings.HasPrefix(text, "/* fairyc constrained C transport: x86_64-windows-msvc, ISO C11 */") {
		return fmt.Errorf("expected Go Windows C11 emission")
	}
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`\b(fy_std_[A-Za-z0-9_]+)\s*\(`).FindAllStringSubmatch(text, -1) {
		seen[m[1]] = true
	}
	symbols := make([]string, 0, len(seen))
	for s := range seen {
		symbols = append(symbols, s)
	}
	slices.Sort(symbols)
	library, err := cstdlib.WindowsMSVCCandidateSourceForSymbols(symbols, strings.Contains(text, "fy_private_vector_"))
	if err != nil {
		return err
	}
	scalar, err := cruntime.ScalarSource(100)
	if err != nil {
		return err
	}
	host, err := candidatehost.WindowsSource(text)
	if err != nil {
		return err
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return err
	}
	// Include the exact already-bounded, checksum-verified monolithic Go unit.
	// It is not converted into candidate partitions or stripped/re-emitted.
	include := filepath.ToSlash(path)
	if strings.ContainsAny(include, "\"\r\n") {
		return fmt.Errorf("unsupported C include path")
	}
	header := "#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n#endif\n"
	literals := "/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n"
	unit := "#include \"" + include + "\"\n"
	image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, cdriver.LowProfile, header, literals, []string{unit}, scalar+"\n"+library+"\n"+host, 10*time.Minute, reserve)
	if err != nil {
		return err
	}
	after, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if sha256.Sum256(after) != actual {
		return fmt.Errorf("Go source changed during native build")
	}
	if err := candidatebuild.Publish(output, image.Bytes); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "sourceSHA256": expected, "sourceBytes": len(data), "imageBytes": len(image.Bytes), "stackReserveBytes": reserve, "privateHost": host != "", "toolchain": toolchain.Version, "publicLinkContractChanged": false})
}
func main() {
	source := flag.String("source", "", "exact Go-generated C source")
	digest := flag.String("source-sha256", "", "source SHA-256")
	output := flag.String("output", "", "private output image")
	reserve := flag.Uint64("stack-reserve-bytes", 0, "explicit private stack reserve")
	flag.Parse()
	if err := run(*source, *digest, *output, *reserve); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
