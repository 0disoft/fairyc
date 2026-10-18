package main

func scalarMIRDeferSeeds() []scalarMIRSeed {
	// The explicit fixture-only trace hook has the same fn(i32)->i32 shape as
	// the existing Go interpreter intrinsic. No trace hook runs in the candidate.
	const helpers = "fn i32 defer_trace(i32 x){discard x;return 1;}fn void emit(i32 x){discard defer_trace(x);}"
	seeds := []scalarMIRSeed{
		{"defer-trace-capture-nested-lifo", "fn i32 main(){var i32 x=65;defer emit(x);defer emit(66);x=1;if(true){defer emit(67);}return 42;}", true},
		{"defer-trace-void-fallthrough", "fn i32 main(){work();return 42;}fn void work(){defer emit(65);}", true},
		{"defer-trace-skipped-branch", "fn i32 main(){defer emit(65);if(false){defer emit(66);}return 0;}", true},
		{"defer-trace-argument-order", "fn i32 main(){defer pair(defer_trace(65),defer_trace(66));return 0;}fn void pair(i32 a,i32 b){assert(a==1&&b==1);discard defer_trace(67);}", true},
		{"defer-trace-return-argument", "fn i32 main(){defer emit(66);defer emit({return 9;65});return 0;}", true},
		{"defer-trace-trap-registration", "fn i32 main(){defer emit(65);defer emit(fail());return 0;}fn i32 fail(){return 1/0;}", true},
		{"defer-trace-trap-body", "fn i32 main(){defer emit(65);assert(false);return 0;}", true},
		{"defer-trace-trap-cleanup", "fn i32 main(){defer emit(65);defer fail();return 0;}fn void fail(){discard defer_trace(66);assert(false);}", true},
		{"defer-trace-return-value-first", "fn i32 main(){defer emit(65);return defer_trace(66);}", true},
		{"defer-trace-block-value", "fn i32 main(){defer emit(67);let i32 x={defer emit(65);let i32 y={defer emit(66);3};y+1};return x;}", true},
		{"defer-trace-while-control", "fn i32 main(){defer emit(90);var i32 i=0;while(i<4){i+=1;defer emit(64+i);if(i==2){continue;}if(i==3){break;}discard defer_trace(70);}return i;}", true},
		{"defer-trace-for-control", "fn i32 main(){defer emit(90);for(var i32 i=0;i<3;i+=1){defer emit(65+i);if(i==1){continue;}discard defer_trace(70);}return 0;}", true},
		{"defer-trace-nested-break", "fn i32 main(){defer emit(90);for(var i32 i=0;i<2;i+=1){defer emit(65+i);while(true){defer emit(80);break;}discard defer_trace(70);}return 0;}", true},
		{"defer-trace-return-loop", "fn i32 main(){defer emit(90);while(true){defer emit(65);return 7;}return 0;}", true},
		{"defer-trace-condition-break-outer", "fn i32 main(){defer emit(90);while(true){defer emit(65);while({defer emit(66);break;true}){} }return 0;}", true},
		{"defer-trace-condition-continue-outer", "fn i32 main(){defer emit(90);var i32 i=0;while(i<2){defer emit(65+i);while({defer emit(66);i+=1;continue;true}){} }return i;}", true},
		{"defer-trace-short-circuit-skipped", "fn i32 main(){defer emit(90);discard false&&{defer emit(65);true};return 0;}", true},
		{"defer-trace-short-circuit-taken", "fn i32 main(){defer emit(90);discard true&&{defer emit(65);true};return 0;}", true},
		{"defer-trace-both-branches", "fn i32 main(){defer emit(90);if(true){defer emit(65);}else{defer emit(66);}if(false){defer emit(67);}else{defer emit(68);}return 0;}", true},
		{"defer-trace-zero-arguments", "fn i32 main(){defer work();return 0;}fn void work(){discard defer_trace(65);}", true},
		{"defer-trace-shadowed-local", "fn i32 main(){var i32 x=65;defer emit(x);discard {var i32 x=66;defer emit(x);x=67;};x=68;return 0;}", true},
	}
	for i := range seeds {
		seeds[i].input += helpers
	}
	return append(seeds, scalarMIRSeed{"defer-i64-capture", "fn i32 main(){var i64 x=4000000000;defer check(x);x=1;return 0;}fn void check(i64 x){assert(x==4000000000);}", true})
}
