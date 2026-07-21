package semantic_test

import "testing"

func TestAnalyzePreservesPublicVisibilityInTypedHIR(t *testing.T) {
	t.Parallel()
	text := `module api;

pub struct Packet { i32 value; }
struct HiddenPacket { i32 value; }

pub enum Status { ready; }
enum HiddenStatus { ready; }

pub resource Handle {
    i32 value;
    cleanup { let i32 observed = this.value; }
}
resource HiddenHandle {
    i32 value;
    cleanup { }
}

pub unsafe extern C fn i32 raw_public(i32 value);
unsafe extern C fn i32 raw_private(i32 value);

pub fn i32 exported(i32 value) { return value; }
fn i32 hidden(i32 value) { return value; }
`
	result := analyzeText(t, map[string]string{"api.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	module := result.HIR.Modules[0]
	if len(module.Structs) != 2 || !module.Structs[0].Public || module.Structs[1].Public {
		t.Fatalf("struct visibility = %+v", module.Structs)
	}
	if len(module.Enums) != 2 || !module.Enums[0].Public || module.Enums[1].Public {
		t.Fatalf("enum visibility = %+v", module.Enums)
	}
	if len(module.Resources) != 2 || !module.Resources[0].Public || module.Resources[1].Public {
		t.Fatalf("resource visibility = %+v", module.Resources)
	}
	if len(module.ForeignFunctions) != 2 || !module.ForeignFunctions[0].Public || module.ForeignFunctions[1].Public {
		t.Fatalf("foreign function visibility = %+v", module.ForeignFunctions)
	}
	if len(module.Functions) != 3 || !module.Functions[0].Public || module.Functions[1].Public || module.Functions[2].Public {
		t.Fatalf("function visibility = %+v", module.Functions)
	}
	if module.Functions[2].DropGlueFor == "" {
		t.Fatalf("resource cleanup function = %+v", module.Functions[2])
	}
}
