package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func bytesLiteralCodecSource() string {
	var literal strings.Builder
	values := make([]string, 256)
	for i := range values {
		fmt.Fprintf(&literal, "\\x%02x", i)
		values[i] = fmt.Sprint(i)
	}
	return fmt.Sprintf(`module main;
import std::bytes as bytes;
fn i32 main() {
    let u8[1] expected = [128];
    assert(bytes::equal(b"\x80", ref expected));
    assert(bytes::compare(b"\x80", ref expected) == 0);
    let ref u8[] all = b"%s";
    let u8[256] oracle = [%s];
    assert(bytes::equal(all, ref oracle));
    assert(bytes::compare(all, ref oracle) == 0);
    var usize i = 0;
    while (i < all.len) { assert(all[i] == oracle[i]); i += 1; }
    return 0;
}`, literal.String(), strings.Join(values, ", "))
}

func TestBytesLiteralCodecExecution(t *testing.T) {
	runByteViewExecution(t, bytesLiteralCodecSource(), "literal-codec", map[string]int{"byteValues": 256, "minimalReproByte": 128}, "Go byte values")
}

func TestBytesLiteralCodecCache(t *testing.T) {
	text := bytesLiteralCodecSource()
	t.Run("memory", func(t *testing.T) { testByteViewSDKCache(t, text) })
	t.Run("independent disk sessions", func(t *testing.T) {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		compiler, err := os.ReadFile(executable)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(compiler)
		configuration := workspacecompiler.Configuration{
			Target: target.X8664WindowsMSVC, MaxArtifactBytes: 8 << 20,
			Persistent: &workspacecompiler.PersistentConfiguration{
				Root: t.TempDir(), CompilerBuild: "test-executable-sha256:" + hex.EncodeToString(digest[:]),
				Limits: packagecache.DiskLimits{MaxEntries: 8, MaxBytes: 32 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute},
			},
		}
		workspace := checkedIntegerConversionWorkspace(t, text)
		for attempt := 0; attempt < 2; attempt++ {
			session, err := workspacecompiler.NewSession(configuration)
			if err != nil {
				t.Fatal(err)
			}
			result, checkErr := session.Check(workspace)
			closeErr := session.Close()
			if checkErr != nil || closeErr != nil || !result.Compiler.Verified {
				t.Fatalf("cache check: %v close: %v %+v", checkErr, closeErr, result.Cache)
			}
			if attempt == 0 && result.Cache.Misses != 2 || attempt == 1 && (result.Cache.Outcome != workspacecompiler.OutcomeAllHit || result.Cache.Hits != 2) {
				t.Fatalf("unexpected cache reuse: %+v", result.Cache)
			}
			outcome, err := interpreter.Execute(result.Compiler.MIR, tokenScannerEntry(t, result.Compiler))
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
				t.Fatalf("cached literal execution: %v %+v", err, outcome)
			}
		}
	})
}
