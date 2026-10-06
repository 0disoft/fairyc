package main

// Test-only Go semantic/MIR/interpreter oracle. Candidate MIR never supplies
// reference expectations and no Go code runs inside the Fairy file CLI.
import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

type scalarMIRSeed struct {
	name, input string
	admitted    bool
}

func writeMIR(destination string) error {
	return writeMIRSeeds(destination, scalarMIRInventory(), "fairyc.mir-file-oracle/v1")
}

func writeMIRSeeds(destination string, seeds []scalarMIRSeed, schema string) error {
	type fixture struct {
		Name          string              `json:"name"`
		InputHex      string              `json:"inputHex"`
		Admitted      bool                `json:"admitted"`
		Program       mir.Program         `json:"program"`
		Outcome       interpreter.Outcome `json:"outcome"`
		OutcomeOrigin string              `json:"outcomeOrigin"`
		TraceCalls    bool                `json:"traceCalls,omitempty"`
	}
	var fixtures []fixture
	for _, seed := range seeds {
		input := "module main; " + seed.input
		file := source.NewFile("main.fy", []byte(input))
		tree := syntax.Parse(file, source.DefaultLimits())
		if !tree.ErrorFree() {
			return fmt.Errorf("MIR oracle syntax %s: %+v", seed.name, tree.Diagnostics)
		}
		module, err := ast.Lower(tree)
		if err != nil {
			return err
		}
		checked := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: module}})
		if len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
			return fmt.Errorf("MIR oracle semantics %s: %+v %+v", seed.name, checked.Diagnostics, checked.Unsupported)
		}
		program, err := mir.Lower(checked.HIR)
		if err != nil {
			return fmt.Errorf("MIR oracle lower %s: %w", seed.name, err)
		}
		if failures := mir.Verify(program); len(failures) != 0 {
			return fmt.Errorf("MIR oracle verifier %s: %v", seed.name, failures)
		}
		trace := strings.HasPrefix(seed.name, "defer-trace-")
		options := interpreter.Options{}
		if trace {
			options.TestIntrinsics = map[string]interpreter.TestIntrinsicKind{"main::defer_trace": interpreter.TestIntrinsicWriteStdoutByte}
			options.OutputLimit = 4096
		}
		outcome, err := interpreter.ExecuteWithOptions(program, "main::main", options)
		if err != nil {
			return fmt.Errorf("MIR oracle execution %s: %w", seed.name, err)
		}
		fixtures = append(fixtures, fixture{seed.name, hex.EncodeToString([]byte(input)), seed.admitted, program, outcome, "go-mir-interpreter", trace})
	}
	data, err := json.MarshalIndent(struct {
		Schema   string    `json:"schema"`
		Fixtures []fixture `json:"fixtures"`
	}{schema, fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0o644)
}

func scalarMIRInventory() []scalarMIRSeed {
	seeds := []scalarMIRSeed{
		{"constant", "fn i32 main(){return 17;}", true},
		{"i64-and-forward-call", "fn i32 main(){var i64 x=4000000000; x+=add(1,2); assert(x==4000000003); return 0;} fn i64 add(i64 a,i64 b){return a+b;}", true},
		{"signed-minima", "fn i32 main(){let i32 x=-2147483648; let i64 y=-0x8000000000000000; assert(x==-2147483648); assert(y==-9223372036854775808); return 0;}", true},
		{"unsigned-limits", "fn i32 main(){let usize x=18446744073709551615; let u32 y=4294967295; let u8 z=255; assert(x==0xffffffffffffffff); assert(y==0xffffffff); assert(z==0xff); return 0;}", true},
		{"all-operators", "fn i32 main(){var i32 x=17; x+=2; x-=1; x*=2; x/=3; x%=7; x&=3; x|=4; x^=1; x<<=1; x>>=1; assert(x==4); assert(+x==4); assert(~x==-5); assert(-x==-4); assert(!(x==3)); assert(x!=3); assert(x<5); assert(x<=4); assert(x>3); assert(x>=4); return x;}", true},
		{"ordered-nested-calls", "fn i32 main(){return sum(id(1),sum(id(2),id(3)));} fn i32 id(i32 x){return x;} fn i32 sum(i32 a,i32 b){return a+b;}", true},
		{"void-call", "fn i32 main(){observe(2); discard observe(1); return 0;} fn void observe(i32 x){discard x;}", true},
		{"lexical-blocks", "fn i32 main(){var i32 x=2; let i32 y={let i32 x=3; x+1}; x=y; return x;}", true},
		{"compound-read-frozen", "fn i32 main(){var i32 x=5; x+={x=2; 3}; return x;}", true},
		{"overflow-i32", "fn i32 main(){var i32 x=2147483647; return x+1;}", true},
		{"overflow-i64", "fn i32 main(){let i64 x=9223372036854775807; discard x+1; return 0;}", true},
		{"negate-minimum", "fn i32 main(){let i64 x=-9223372036854775808; discard -x; return 0;}", true},
		{"division-zero", "fn i32 main(){let i64 x=5; discard x/0; return 0;}", true},
		{"signed-division-overflow", "fn i32 main(){let i64 x=-9223372036854775808; discard x/-1; return 0;}", true},
		{"signed-remainder-overflow", "fn i32 main(){let i32 x=-2147483648; discard x%-1; return 0;}", true},
		{"shift-negative", "fn i32 main(){let i32 x=2; return x>>-1;}", true},
		{"shift-width", "fn i32 main(){let u8 x=1; discard x<<8; return 0;}", true},
		{"shift-overflow", "fn i32 main(){let u32 x=4294967295; discard x<<1; return 0;}", true},
		{"assertion", "fn i32 main(){assert(false); return 0;}", true},
		{"if-simple", "fn i32 main(){return if(true){1}else{2};}", true},
		{"short-circuit-simple", "fn i32 main(){assert(false || true); return 0;}", true},
		{"while-basic", "fn i32 main(){var i32 x=0; while(x<2){x+=1;} return x;}", true},
		{"for-basic", "fn i32 main(){var i32 x=0; for(var i32 i=0;i<2;i+=1){x+=1;} return x;}", true},
		{"defer-basic", "fn i32 main(){defer observe(1); return 0;} fn void observe(i32 x){discard x;}", true},
	}
	seeds = append(seeds, scalarMIRControlSeeds()...)
	seeds = append(seeds, scalarMIRLoopSeeds()...)
	seeds = append(seeds, scalarMIRDeferSeeds()...)
	return seeds
}
