package cli

// These bounded witnesses fill token-alternative gaps identified by the grammar
// ledger. They run through both independent writers in the existing native lane.
// They do not claim exhaustive operator combinations or semantic validity.
func bodyTreeGrammarAuditInputs() []string {
	var inputs []string
	body := func(text string) { inputs = append(inputs, "module main;fn void f(){"+text+"}") }
	for _, op := range []string{"||", "&&", "==", "!=", "<", "<=", ">", ">=", "|", "^", "&", "<<", ">>", "+", "-", "*", "/", "%"} {
		body("return a " + op + " b;")
	}
	for _, op := range []string{"-", "+", "!", "~", "ref", "ref mut", "*"} {
		body("return " + op + " value;")
	}
	for _, op := range []string{"=", "+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|="} {
		body("target " + op + " value;")
	}
	for _, literal := range []string{"true", "false", "null", "1_000", "0xA_F", "0b10_01", "1.25e-2", "2e3", `"a\n\u{AC00}"`, `b"a\x00"`, `c"a\x41"`} {
		body("return " + literal + ";")
	}
	body("return a || b && c == d | e ^ f & g << h + i * j;")
	body("return ((a+b) as i32)*-c;")
	body("call(1,2,);pkg /* gap */ :: make<u8,i32,>(1,2,);let Pair<u8,i32,> p=value;return .None;")
	// Header syntax is accepted but intentionally not structurally exported.
	inputs = append(inputs, "module main;@effects(console) pub unsafe fn T Owner::f<T>(ref mut T arg,){return arg;}")
	return inputs
}
