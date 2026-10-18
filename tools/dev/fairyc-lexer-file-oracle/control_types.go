package main

// These fixtures exercise authored Fairy control checking; this Go utility
// supplies independent semantic outcomes and diagnostic spans as test data.
func addControlTypes(add func(string, string, string, string) error) error {
	for _, s := range []struct{ name, input string }{
		{"if-value-i64", "module main; fn i64 f(bool b) { return if(b) { 4000000000 } else { 5000000000 }; }"},
		{"if-inferred", "module main; fn i32 f(bool b) { let x=if(b) { 1 } else { 2 }; return x; }"},
		{"else-if-value", "module main; fn i64 f(i64 x) { return if(x>0) { x } else if(x<0) { -x } else { 0 }; }"},
		{"if-all-return", "module main; fn i64 f(bool b) { if(b) { return 4000000000; } else { return 5000000000; } }"},
		{"else-if-all-return", "module main; fn i64 f(i64 x) { if(x>0) { return x; } else if(x<0) { return -x; } else { return 0; } }"},
		{"if-void", "module main; fn void f(bool b) { if(b) { g(); } } fn void g() {}"},
		{"while-continue-break", "module main; fn i64 f(i64 limit) { var i64 n=0; while(n<limit) { n+=1; if(n<2) { continue; } if(n>4) { break; } } return n; }"},
		{"while-call-condition", "module main; fn void f() { while(g()) { break; } } fn bool g() { return false; }"},
		{"for-i64", "module main; fn i64 f(i64 limit) { var i64 n=0; for(var i64 i=0; i<limit; i+=1) { n+=i; } return n; }"},
		{"for-shadow", "module main; fn i64 f(i64 i) { for(var i64 i=i; i<3; i+=1) { let i64 i=i+1; i; } return i; }"},
		{"for-inferred", "module main; fn void f() { for(var i=0; i<3; i+=1) {} }"},
		{"for-assignment", "module main; fn i32 f() { var i32 i=0; for(i=1; i<4; i+=1) {} return i; }"},
		{"for-omitted-clauses", "module main; fn void f() { for(;;) { break; } }"},
		{"for-bool", "module main; fn void f() { for(var bool b=true; b; b=false) { continue; } }"},
		{"nested-loops", "module main; fn i32 f() { var i32 n=0; for(var i32 i=0; i<3; i+=1) { while(n<2) { n+=1; break; } continue; } return n; }"},
		{"assert-discard", "module main; fn void f(i64 x) { assert(x>=0); discard x+1; }"},
		{"false-assert-return-gate", "module main; fn i64 f() { assert(false); }"},
		{"defer-scalars", "module main; fn void f(i64 x) { defer g(x+1,true); } fn void g(i64 x,bool b) {}"},
		{"defer-in-loop", "module main; fn void f() { for(var i32 i=0; i<2; i+=1) { defer g(i); if(i==1) { continue; } } } fn void g(i32 x) {}"},
		{"post-after-body-shadow", "module main; fn void f() { for(var i64 i=0; i<3; i+=1) { let bool i=true; i; } }"},
	} {
		if err := add("control-"+s.name, s.input, "pass", ""); err != nil {
			return err
		}
	}
	for _, s := range []struct{ name, input, message string }{
		{"if-condition", "module main; fn void f() { if(1) {} }", "expected bool operand"},
		{"while-condition", "module main; fn void f() { while(1) {} }", "expected bool operand"},
		{"for-condition", "module main; fn void f() { for(;1;) {} }", "expected bool operand"},
		{"assert-condition", "module main; fn void f() { assert(1); }", "expected bool operand"},
		{"if-branch-types", "module main; fn i64 f(bool b) { return if(b) { 1 } else { true }; }", "type mismatch"},
		{"else-if-branch-types", "module main; fn i64 f(bool b) { return if(b) { 1 } else if(b) { 2 } else { false }; }", "type mismatch"},
		{"if-branch-return-types", "module main; fn i64 f(bool b) { if(b) { return true; } else { return 1; } }", "type mismatch"},
		{"if-one-return", "module main; fn i64 f(bool b) { if(b) { return 1; } }", "missing return value"},
		{"if-partial-else", "module main; fn i64 f(bool b) { if(b) { return 1; } else {} }", "missing return value"},
		{"else-if-missing-else", "module main; fn i64 f(bool b) { if(b) { return 1; } else if(b) { return 2; } }", "missing return value"},
		{"while-not-return-proof", "module main; fn i64 f() { while(true) { return 1; } }", "missing return value"},
		{"for-not-return-proof", "module main; fn i64 f() { for(;;) { return 1; } }", "missing return value"},
		{"break-outside", "module main; fn void f() { break; }", "loop control outside loop"},
		{"continue-outside", "module main; fn void f() { { continue; } }", "loop control outside loop"},
		{"break-after-loop", "module main; fn void f() { while(false) {} break; }", "loop control outside loop"},
		{"continue-after-for", "module main; fn void f() { for(;;) { break; } continue; }", "loop control outside loop"},
		{"for-initial-call", "module main; fn void f() { for(var i64 i=g(); i<3; i+=1) {} } fn i64 g() { return 0; }", "unsupported for header"},
		{"for-condition-call", "module main; fn void f() { for(;g();) {} } fn bool g() { return false; }", "unsupported for header"},
		{"for-post-call", "module main; fn void f() { for(var i64 i=0; i<3; i=g()) {} } fn i64 g() { return 0; }", "unsupported for header"},
		{"for-initial-block", "module main; fn void f() { for(var i32 i={ 0 }; i<3; i+=1) {} }", "unsupported for header"},
		{"for-immutable-post", "module main; fn void f() { for(let i32 i=0; i<3; i+=1) {} }", "assignment target is immutable"},
		{"for-post-type", "module main; fn void f() { for(var i32 i=0; i<3; i=true) {} }", "type mismatch"},
		{"defer-not-call", "module main; fn void f() { defer 1; }", "unsupported defer call"},
		{"defer-value-call", "module main; fn void f() { defer g(); } fn i32 g() { return 0; }", "unsupported defer call"},
		{"defer-argument-type", "module main; fn void f() { defer g(true); } fn void g(i64 x) {}", "type mismatch"},
		{"late-branch-error", "module main; fn void f(bool b) { if(b) {} else { let i32 x=true; } }", "type mismatch"},
		{"late-after-return", "module main; fn i32 f() { return 0; while(false) { let i32 x=true; } }", "type mismatch"},
	} {
		if err := add("control-"+s.name, s.input, "error", s.message); err != nil {
			return err
		}
	}
	return nil
}
