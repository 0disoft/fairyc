package semantic

import (
	"strings"
	"testing"
)

func TestRegionOriginsRetainReturnedLoopBindings(t *testing.T) {
	const allocation = "let Result<ref i32,alloc::AllocError> value=scratch.place(1);return value;"
	for name, body := range map[string]string{
		"direct":  allocation,
		"classic": "for(var i32 i=0;i<1;i=i+1){" + allocation + "}",
		"view":    "for(let u8 b in bytes){" + allocation + "}",
		"nested":  "for(var i32 i=0;i<1;i=i+1){for(let u8 b in bytes){" + allocation + "}}",
	} {
		t.Run(name, func(t *testing.T) {
			text := "module main;import std::alloc as alloc;fn Result<ref i32,alloc::AllocError> f(Region scratch,ref u8[] bytes){" + body + "return .err(alloc::AllocError{.status=1});}"
			result := Analyze(regionOriginTextFiles(t, text))
			if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || len(result.RegionOrigins) != 1 {
				t.Fatalf("loop Region proof: phase=%s diagnostics=%v unsupported=%v origins=%v", result.CompletedThrough, result.Diagnostics, result.Unsupported, result.RegionOrigins)
			}
			fact := result.RegionOrigins[0]
			last := strings.Index(text, "return value") + len("return ")
			if !fact.Escapes || fact.Origin != "RegionParam(0)" || fact.LastUse == nil || fact.LastUse.StartByte != last || fact.LastUse.EndByte != last+len("value") {
				t.Fatalf("returned Region binding lost origin: %+v", fact)
			}
		})
	}
}

func TestRegionOriginsSkipBindingIndexWithoutAllocations(t *testing.T) {
	a := newAnalyzer(borrowLiveFiles(t, "module main;fn ref i32 f(ref i32 p){let ref i32 alias=p;return alias;}"), nil)
	a.resolve()
	a.typeCheck()
	a.flowCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 || a.bindingsByExpression != nil || len(a.regionOrigins) != 0 {
		t.Fatalf("non-Region flow built an origin index or lost proof: diagnostics=%v unsupported=%v index=%v", a.diagnostics, a.unsupported, a.bindingsByExpression)
	}
}
