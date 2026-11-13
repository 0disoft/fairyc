package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// A source limit must not publish a new cache generation or overwrite the last
// good native product. Exercise the authored CLI, not just its status helper.
func compositeResultCLICapacityRefusal(t *testing.T, command *copiedClosureCommand, root string) {
	t.Helper()
	good := copiedClosureCacheSnapshot(t, root)
	product := filepath.Join(root, "build", "result-cli.exe")
	previous, err := os.ReadFile(product)
	if err != nil {
		t.Fatal(err)
	}
	helper := compositeResultCLIOverLimitHelper()
	writeTestSource(t, root, "libs/answer/src/answer.fy", helper)
	groupedStaticCLIStaleLock(t, command)
	for _, name := range []string{"check", "build", "run"} {
		out, diagnostic := command.run(ExitSourceDiagnostics, name, "--cache-root", "cache", "--cache-status")
		if out != "" || !strings.HasPrefix(diagnostic, "cache: cold uncached\n") || !strings.Contains(diagnostic, ": E4503:") || strings.Contains(diagnostic, "invariant") {
			t.Fatalf("generic limit %s: stdout=%q stderr=%q", name, out, diagnostic)
		}
		span := regexp.MustCompile(`\[bytes ([0-9]+)\.\.([0-9]+)\]`).FindStringSubmatch(diagnostic)
		var start, end int
		if len(span) != 3 {
			t.Fatal("generic limit lost original source range", diagnostic)
		}
		if _, err := fmt.Sscan(span[1]+" "+span[2], &start, &end); err != nil {
			t.Fatal(err)
		}
		var source string
		if strings.Contains(diagnostic, "libs/answer/answer.fy:") {
			source = helper
		} else if strings.Contains(diagnostic, "main.fy:") {
			source = compositeResultCLIMain()
		}
		if start < 0 || end <= start || end > len(source) || !strings.Contains(source[start:end], "<") {
			t.Fatal("generic limit is not anchored to an original generic request", diagnostic)
		}
		after, err := os.ReadFile(product)
		if err != nil || !bytes.Equal(previous, after) || !reflect.DeepEqual(good, copiedClosureCacheSnapshot(t, root)) {
			t.Fatal("generic limit changed good executable or cache", err)
		}
		t.Logf("capacity refusal %s preserved cache/product: %s", name, diagnostic)
	}
	writeTestSource(t, root, "libs/answer/src/answer.fy", compositeResultCLIProvider)
	groupedStaticCLIExact(t, command, 0, "cache: hit", "check")
}
