package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

func emitResultTagBranches(output *strings.Builder, value string, success, failure mir.BlockID) {
	// MSVC can propagate a callee's constant Result tag, infer that its caller
	// always traps, and reject the caller's valid MIR continuations with C4702.
	// Snapshot once through the same private volatile-read barrier used for
	// inline arguments. No tag mutation, warning suppression or ABI change.
	output.WriteString("#if defined(_MSC_VER)\n    {\n")
	fmt.Fprintf(output, "        volatile uint32_t fy_result_tag_observed = (%s).tag;\n", value)
	output.WriteString("        uint32_t fy_result_tag = fy_result_tag_observed;\n")
	fmt.Fprintf(output, "        if (fy_result_tag == FY_RESULT_OK) { goto fy_b_%d; }\n", success)
	fmt.Fprintf(output, "        if (fy_result_tag == FY_RESULT_ERR) { goto fy_b_%d; }\n", failure)
	output.WriteString("    }\n#else\n")
	fmt.Fprintf(output, "    if ((%s).tag == FY_RESULT_OK) { goto fy_b_%d; }\n", value, success)
	fmt.Fprintf(output, "    if ((%s).tag == FY_RESULT_ERR) { goto fy_b_%d; }\n", value, failure)
	output.WriteString("#endif\n")
}
