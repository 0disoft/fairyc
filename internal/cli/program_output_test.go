package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestForwardProgramOutputPreservesStreamIdentityAndFailureCodes(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := forwardProgramOutput("out\n", "err\n", &stdout, &stderr); code != ExitSuccess ||
		stdout.String() != "out\n" || stderr.String() != "err\n" {
		t.Fatalf("forwardProgramOutput() = code %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}

	stderr.Reset()
	if code := forwardProgramOutput("out", "err", failingWriter{}, &stderr); code != ExitInternalFailure ||
		!strings.Contains(stderr.String(), "write output failed") {
		t.Fatalf("stdout failure = code %d, stderr %q", code, stderr.String())
	}
	if code := forwardProgramOutput("out", "err", &stdout, failingWriter{}); code != ExitInternalFailure {
		t.Fatalf("stderr failure = code %d", code)
	}
}
