package interpreter_test

import (
	"github.com/0disoft/fairyc/internal/interpreter"
	"testing"
)

func TestExecuteEnumCallArgumentsAreFrozen(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        int32
	}{
		{"payloadless", `module main;enum E{A;B;}fn i32 f(E a,E b){return switch(a){case .A:{10}case .B:{20}}+switch(b){case .A:{1}case .B:{2}};}fn i32 main(){var E e=.A;return f(e,{e=.B;e});}`, 12},
		{"inert-payload", `module main;enum E{A{i32 n;};B;}fn i32 f(E a,E b){return switch(a){case .A{.n=n}:{n}case .B:{20}}+switch(b){case .A{.n=n}:{n}case .B:{2}};}fn i32 main(){var E e=.A{.n=10};return f(e,{e=.B;e});}`, 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := lowerSource(t, tc.input)
			got, err := interpreter.Execute(p, "main::main")
			if err != nil || got.Kind != interpreter.OutcomeReturn || got.Value != tc.want {
				t.Fatalf("enum argument snapshot: %+v %v; want %d", got, err, tc.want)
			}
		})
	}
}
