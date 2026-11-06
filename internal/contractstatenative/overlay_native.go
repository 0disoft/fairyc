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

// OverlayExecutable owns a built, bound native framing image. It grants no
// persistence authority; callers must separately publish an admitted result.
type OverlayExecutable struct {
	prepared OverlayTransport
	image    cdriver.WindowsHostArtifact
	valid    bool
}

func (prepared OverlayTransport) Source() (string, error) {
	if !prepared.entry.valid || prepared.workspace == 0 {
		return "", fmt.Errorf("overlay framing requires a prepared owner")
	}
	var source strings.Builder
	source.WriteString(prepared.entry.Source())
	source.WriteString("\n#include <string.h>\n")
	identity := make([]byte, 0, 96)
	// Fixed-width identity components are emitted without host struct layout.
	identity = append(identity, prepared.entry.layout[:]...)
	identity = append(identity, prepared.entry.body[:]...)
	identity = append(identity, prepared.entry.programHash[:]...)
	fmt.Fprintf(&source, "\nstatic const uint8_t fyov_identity[96] = {%s};\n", cBytes(identity))
	fmt.Fprintf(&source, "#define FYOV_WRITE_COUNT %du\n#define FYOV_WORKSPACE %du\n#define FYOV_READ_COUNT %du\n#define FYOV_EXPECTED_EXIT %du\n#define FYOV_LITERAL_SIZE %du\n", max(1, len(prepared.writes)), prepared.workspace, len(prepared.reads), prepared.exit, prepared.maxLiteral)
	keys := make([]contractstate.Key, 0, len(prepared.known))
	for key := range prepared.known {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b contractstate.Key) int { return bytes.Compare(a[:], b[:]) })
	if len(keys) == 0 {
		source.WriteString("static uint8_t fyov_codec(const uint8_t key[32]) { (void)key; return 0u; }\n")
	} else {
		source.WriteString("typedef struct fyov_known { uint8_t key[32]; uint8_t codec; } fyov_known;\nstatic const fyov_known fyov_keys[] = {\n")
		for _, key := range keys {
			codec, err := overlayCodec(prepared.known[key])
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&source, "    {{%s}, %du},\n", cBytes(key[:]), codec)
		}
		fmt.Fprintf(&source, "};\nstatic uint8_t fyov_codec(const uint8_t key[32]) { size_t low = 0u, high = %du; while (low < high) { size_t middle = low + (high-low)/2u; int order = memcmp(fyov_keys[middle].key, key, 32u); if (order < 0) { low = middle+1u; } else { high = middle; } } if (low < %du && memcmp(fyov_keys[low].key, key, 32u) == 0) { return fyov_keys[low].codec; } return 0u; }\n", len(keys), len(keys))
	}
	if len(prepared.reads) != 0 {
		source.WriteString("static const uint32_t fyov_operations[] = {")
		for _, read := range prepared.reads {
			fmt.Fprintf(&source, "%du,", read.operation)
		}
		source.WriteString("};\nstatic const uint8_t fyov_codecs[] = {")
		for _, read := range prepared.reads {
			fmt.Fprintf(&source, "%du,", read.codec)
		}
		source.WriteString("};\n")
	}
	source.WriteString(overlayNativeFrameC)
	return source.String(), nil
}

func BuildOverlay(prepared OverlayTransport, profile cdriver.Profile) (OverlayExecutable, error) {
	source, err := prepared.Source()
	if err != nil {
		return OverlayExecutable{}, err
	}
	runtimeSource, err := cstdlib.ContractStateOverlayRuntimeSource()
	if err != nil {
		return OverlayExecutable{}, err
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return OverlayExecutable{}, err
	}
	image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, source, runtimeSource)
	if err != nil {
		return OverlayExecutable{}, err
	}
	image.Bytes = slices.Clone(image.Bytes)
	return OverlayExecutable{prepared: prepared, image: image, valid: true}, nil
}

func RunOverlay(ctx context.Context, executable OverlayExecutable, input OverlayInput) (OverlayResult, error) {
	if !executable.valid || !input.valid {
		return OverlayResult{}, fmt.Errorf("native overlay requires built and request owners")
	}
	result, err := cdriver.ExecuteWindowsMSVCTargetWithContextAndOutputLimit(ctx, executable.image, bytes.NewReader(input.Bytes()), time.Minute, overlayResponseBytes)
	if err != nil {
		return OverlayResult{}, fmt.Errorf("execute native overlay: %w", err)
	}
	if result.ExitCode != 0 || result.Stderr != "" || result.TrapControl != "" {
		return OverlayResult{}, fmt.Errorf("native overlay process failed framing (exit %d)", result.ExitCode)
	}
	return executable.prepared.Decode(input, []byte(result.Stdout))
}
