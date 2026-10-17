package cbackendnative

import (
	"github.com/0disoft/fairyc/internal/cdriver"
)

func runByteStringLiteralSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"byte-string-literal-source-call",
		byteStringLiteralSourceCallSmokeSource,
		byteStringLiteralSourceCallCompanion,
		42,
		byteStringLiteralSourceCallExpectedCFragments,
	)
}

func runUTF8StringLiteralSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"utf8-string-literal-source-call",
		utf8StringLiteralSourceCallSmokeSource,
		utf8StringLiteralSourceCallCompanion,
		42,
		utf8StringLiteralSourceCallExpectedCFragments,
	)
}

func runCStringLiteralSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"c-string-literal-source-call",
		cStringLiteralSourceCallSmokeSource,
		cStringLiteralSourceCallCompanion,
		42,
		cStringLiteralSourceCallExpectedCFragments,
	)
}
