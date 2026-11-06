package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type atlasReturnErrorOracleSite struct{ function, expression string }
type atlasReturnErrorOracleFixture struct {
	name, source string
	sites        []atlasReturnErrorOracleSite
}

// The Go observation is the single written question token, not the complete
// operand, the implicit error return, or an explicit .err construction.
func atlasReturnErrorSiteOracleCases() []atlasReturnErrorOracleFixture {
	type site = atlasReturnErrorOracleSite
	const prefix = "module main;fn Result<i32,i32> read(){return .ok(7);}fn i32 main(){return 0;}"
	return []atlasReturnErrorOracleFixture{
		{"none", prefix, nil},
		{"explicit-error", prefix + "fn Result<i32,i32> exercise(){return .err(3);}", nil},
		{"bound-result", prefix + "fn Result<i32,i32> exercise(){let value=read();let n=value?;return .ok(n);}", []site{{"exercise", "value?"}}},
		{"call-result", prefix + "fn Result<i32,i32> exercise(){let n=read()?;return .ok(n);}", []site{{"exercise", "read()?"}}},
		{"repeated", prefix + "fn Result<i32,i32> exercise(){let first=read()?;let second=read()?;return .ok(first+second);}", []site{{"exercise", "first=read()?"}, {"exercise", "second=read()?"}}},
		{"nested", prefix + "fn Result<i32,i32> wrap(i32 n){return .ok(n);}fn Result<i32,i32> exercise(){return .ok(wrap(read()?)?);}", []site{{"exercise", "read()?"}, {"exercise", "wrap(read()?)?"}}},
		{"branches", prefix + "fn Result<i32,i32> exercise(bool flag){if(flag){let left=read()?;}else{let right=read()?;}return .ok(0);}", []site{{"exercise", "left=read()?"}, {"exercise", "right=read()?"}}},
		{"loop", prefix + "fn Result<i32,i32> exercise(){var i32 n=0;while(n<2){n+=read()?;}return .ok(n);}", []site{{"exercise", "read()?"}}},
		{"before-later-trap", prefix + "fn i32 take(i32 a,i32 b){return a+b;}fn Result<i32,i32> exercise(){return .ok(take(read()?,{assert(false);0}));}", []site{{"exercise", "read()?"}}},
		{"after-earlier-trap", prefix + "fn i32 take(i32 a,i32 b){return a+b;}fn Result<i32,i32> exercise(){return .ok(take({assert(false);0},read()?));}", nil},
		{"after-direct-trap", prefix + "fn Result<i32,i32> exercise(){assert(false);let n=read()?;return .ok(n);}", nil},
		{"trapping-operand", prefix + "fn Result<i32,i32> exercise(){let n={assert(false);read()}?;return .ok(n);}", nil},
		{"returning-operand", prefix + "fn Result<i32,i32> exercise(){let n={return .err(3);read()}?;return .ok(n);}", nil},
	}
}

func TestAtlasReturnErrorSitesOracle(t *testing.T) {
	for _, f := range atlasReturnErrorSiteOracleCases() {
		t.Run(f.name, func(t *testing.T) {
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(writeWorkspaceSessionFixture(t, f.source), false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("return-error oracle source: %d %s", code, &diagnostic)
			}
			seenMain, count := false, 0
			for _, partition := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				for _, summary := range decoded.Functions() {
					if summary.Function.Module != "main" {
						continue
					}
					seenMain = seenMain || summary.Function.Name == "main"
					var want []semanticsnapshot.Span
					for _, s := range f.sites {
						if s.function != summary.Function.Name {
							continue
						}
						if strings.Count(f.source, s.expression) != 1 || !strings.HasSuffix(s.expression, "?") {
							t.Fatalf("ambiguous original question selector: %+v", s)
						}
						start := strings.Index(f.source, s.expression) + len(s.expression) - 1
						want = append(want, semanticsnapshot.Span{PackageInstance: summary.Function.PackageInstance, Source: "main.fy", StartByte: start, EndByte: start + 1})
					}
					if !slices.Equal(summary.ReturnErrorSites, want) {
						t.Fatalf("%s return-error sites: got %+v want written question tokens %+v", summary.Function.Name, summary.ReturnErrorSites, want)
					}
					count += len(summary.ReturnErrorSites)
				}
			}
			if !seenMain || count != len(f.sites) {
				t.Fatalf("incomplete return-error oracle: main=%v sites=%d want %d", seenMain, count, len(f.sites))
			}
		})
	}
}
