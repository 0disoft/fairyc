package contractstatenative

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

func TestNativeStateGraphDigest(t *testing.T) {
	header, err := cstdlib.ContractStateGraphDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateGraphRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	var source strings.Builder
	source.WriteString(header)
	source.WriteString(`
#include <stdio.h>
#include <string.h>
#include <fcntl.h>
#include <io.h>
int main(void) {
    uint8_t input[256], output[32], saved[32]; size_t i;
    if (_setmode(_fileno(stdout),_O_BINARY) == -1) { return 1; }
    memset(output,205,sizeof(output)); memcpy(saved,output,sizeof(output));
    if (fycs_graph_digest_v1(NULL,1u,NULL,0u,output) != FYCS_OV_INVALID ||
        fycs_graph_digest_v1(NULL,0u,NULL,1u,output) != FYCS_OV_INVALID ||
        fycs_graph_digest_v1(input,1u,input,0u,NULL) != FYCS_OV_INVALID ||
        fycs_graph_digest_v1(input,SIZE_MAX,input,0u,output) != FYCS_OV_INVALID ||
        fycs_graph_digest_v1(input,(size_t)(UINT64_MAX/8u),input,1u,output) != FYCS_OV_INVALID ||
        memcmp(output,saved,sizeof(output)) != 0) { return 2; }
`)
	var expected []byte
	count := 0
	for _, size := range []int{0, 1, 31, 32, 55, 56, 63, 64, 65, 119, 120, 127, 128, 129, 255} {
		input := make([]byte, size)
		for i := range input {
			input[i] = byte(i)
		}
		digest := sha256.Sum256(input)
		splits := []int{0, size / 2, size}
		slices.Sort(splits)
		for _, split := range slices.Compact(splits) {
			for _, output := range []string{"output", "input", "input+1u"} {
				fmt.Fprintf(&source, "    for(i=0u;i<sizeof(input);++i){input[i]=(uint8_t)i;}\n    if(fycs_graph_digest_v1(input,%du,input+%du,%du,%s)!=FYCS_OV_OK || fwrite(%s,1u,32u,stdout)!=32u){return 3;}\n", split, split, size-split, output, output)
				expected = append(expected, digest[:]...)
				count++
			}
		}
	}
	source.WriteString("    return fflush(stdout)==0 ? 0 : 4;\n}\n")
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, source.String(), runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			out, err := cdriver.ExecuteWindowsMSVCTarget(image)
			if err != nil || out.ExitCode != 0 || out.Stderr != "" || !bytes.Equal([]byte(out.Stdout), expected) {
				t.Fatal("native split/aliased request SHA-256 differs from Go", count, out.ExitCode, out.Stderr, err)
			}
		})
	}
}
