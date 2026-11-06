package contractstatenative

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

// GraphExecutable binds a built private framing image to its prepared decoder.
// A run returns evidence only; storage CAS remains a separate authority.
type GraphExecutable struct {
	prepared GraphTransport
	image    cdriver.WindowsHostArtifact
	valid    bool
}

func (prepared GraphTransport) Source() (string, error) {
	if !prepared.entry.valid || prepared.start == 0 || prepared.work == 0 {
		return "", fmt.Errorf("graph framing requires a prepared owner")
	}
	var out strings.Builder
	out.WriteString(prepared.entry.Source())
	out.WriteString("\n#include <string.h>\n")
	identity := append([]byte(nil), prepared.entry.layout[:]...)
	identity = append(identity, prepared.entry.body[:]...)
	identity = append(identity, prepared.program[:]...)
	fmt.Fprintf(&out, "static const uint8_t fygr_identity[96] = {%s};\n#define FYGR_WORK %du\n#define FYGR_LITERAL %du\n#define FYGR_TRAP_COUNT %du\n", cBytes(identity), prepared.work, prepared.literal, len(prepared.entry.traps))
	keys := make([]contractstate.Key, 0, len(prepared.input.known))
	for key := range prepared.input.known {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b contractstate.Key) int { return bytes.Compare(a[:], b[:]) })
	if len(keys) == 0 {
		out.WriteString("static uint8_t fygr_codec(const uint8_t key[32]) { (void)key; return 0u; }\n")
	} else {
		out.WriteString("typedef struct fygr_known { uint8_t key[32]; uint8_t codec; } fygr_known;\nstatic const fygr_known fygr_keys[] = {\n")
		for _, key := range keys {
			codec, _ := overlayCodec(prepared.input.known[key])
			fmt.Fprintf(&out, "{{%s},%du},\n", cBytes(key[:]), codec)
		}
		fmt.Fprintf(&out, "};\nstatic uint8_t fygr_codec(const uint8_t key[32]) { size_t low = 0u, high = %du; while (low < high) { size_t middle = low+(high-low)/2u; int order = memcmp(fygr_keys[middle].key,key,32u); if (order < 0) { low = middle+1u; } else { high = middle; } } if (low < %du && memcmp(fygr_keys[low].key,key,32u) == 0) { return fygr_keys[low].codec; } return 0u; }\n", len(keys), len(keys))
	}
	out.WriteString(graphNativeFrameC)
	return out.String(), nil
}

func BuildGraph(prepared GraphTransport, profile cdriver.Profile) (GraphExecutable, error) {
	source, err := prepared.Source()
	if err != nil {
		return GraphExecutable{}, err
	}
	runtimeSource, err := cstdlib.ContractStateGraphRuntimeSource()
	if err != nil {
		return GraphExecutable{}, err
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return GraphExecutable{}, err
	}
	image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, source, runtimeSource)
	if err != nil {
		return GraphExecutable{}, err
	}
	image.Bytes = slices.Clone(image.Bytes)
	return GraphExecutable{prepared: prepared, image: image, valid: true}, nil
}

func RunGraph(ctx context.Context, executable GraphExecutable, input GraphInput) (GraphResult, error) {
	if ctx == nil || !executable.valid || !input.valid {
		return GraphResult{}, fmt.Errorf("native graph requires built and request owners")
	}
	result, err := executable.run(ctx, input)
	if err != nil {
		return GraphResult{}, fmt.Errorf("execute native graph: %w", err)
	}
	return executable.prepared.decodeProcess(input, result)
}

func (executable GraphExecutable) run(ctx context.Context, input GraphInput) (cdriver.WindowsHostOutcome, error) {
	return cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(ctx, executable.image, bytes.NewReader(input.Bytes()), time.Minute, overlayResponseBytes)
}

func (prepared GraphTransport) decodeProcess(input GraphInput, result cdriver.WindowsHostOutcome) (GraphResult, error) {
	if result.ExitCode != 0 || result.Stderr != "" || result.TrapControl != "" {
		return GraphResult{}, fmt.Errorf("native graph process failed framing (exit %d)", result.ExitCode)
	}
	return prepared.Decode(input, []byte(result.Stdout))
}
