package cbackend_test

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestEmissionExplanationPreservesCAndExplainsReachability(t *testing.T) {
	program := lowerSource(t, `module main;
unsafe extern C fn i32 visit(unsafe extern C fn i32(i32,i32,ptr mut void) cb,ptr mut void data);
unsafe fn i32 handler(i32 a,i32 b,ptr mut void data){return a+b;}
fn i32 invoke(){return unsafe{visit(handler,null)};}
resource Token { i32 value; cleanup { let i32 observed = this.value; } }
fn i32 close_token(){let Token token=Token{.value=42};return token.value;}
fn i32 cycle_left(i32 n){if (n == 0) { return 0; } return cycle_right(n-1);}
fn i32 cycle_right(i32 n){if (n == 0) { return 0; } return cycle_left(n-1);}
fn i32 unused(){return 5;}
fn i32 main(){return close_token()+invoke()+cycle_left(2);}
`)
	const entry = "main::main"
	profile := cbackend.Stage0Profile()
	plain, err := cbackend.Emit(program, entry, profile)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		symbol  string
		emitted bool
		path    []cbackend.EmissionStep
	}{
		{"main", true, []cbackend.EmissionStep{}},
		{"close_token", true, []cbackend.EmissionStep{{From: entry, To: "main::close_token", Reason: "call"}}},
		{"main::$drop$Token", true, []cbackend.EmissionStep{{From: entry, To: "main::close_token", Reason: "call"}, {From: "main::close_token", To: "main::$drop$Token", Reason: "drop"}}},
		{"main::Token::cleanup", true, []cbackend.EmissionStep{{From: entry, To: "main::close_token", Reason: "call"}, {From: "main::close_token", To: "main::$drop$Token", Reason: "drop"}}},
		{"handler", true, []cbackend.EmissionStep{{From: entry, To: "main::invoke", Reason: "call"}, {From: "main::invoke", To: "main::handler", Reason: "callback"}}},
		{"cycle_right", true, []cbackend.EmissionStep{{From: entry, To: "main::cycle_left", Reason: "call"}, {From: "main::cycle_left", To: "main::cycle_right", Reason: "call"}}},
		{"unused", false, []cbackend.EmissionStep{}},
	} {
		t.Run(tc.symbol, func(t *testing.T) {
			traced, report, err := cbackend.EmitWithExplanation(program, entry, profile, tc.symbol)
			if err != nil || report.Emitted != tc.emitted || !reflect.DeepEqual(report.Path, tc.path) {
				t.Fatalf("explanation: %+v (%v)", report, err)
			}
			if traced.Source != plain.Source {
				t.Fatal("observing reachability changed the emitted C bytes")
			}
		})
	}
}
