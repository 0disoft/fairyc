package main

func scalarMIRLoopSeeds() []scalarMIRSeed {
	return []scalarMIRSeed{
		{"while-zero", "fn i32 main(){while(false){discard fail();}return 4;}fn i32 fail(){assert(false);return 99;}", true},
		{"while-i64", "fn i32 main(){var i64 x=4000000000;while(x<4000000003){x+=1;}assert(x==4000000003);return 0;}", true},
		{"while-continue", "fn i32 main(){var i32 i=0;var i32 total=0;while(i<5){i+=1;if(i==3){continue;}total+=i;}return total;}", true},
		{"while-break", "fn i32 main(){var i32 i=0;while(true){i+=1;if(i==3){break;}}return i;}", true},
		{"while-nested-control", "fn i32 main(){var i32 i=0;var i32 total=0;while(i<3){i+=1;var i32 j=0;while(j<4){j+=1;if(j==2){continue;}if(j==4){break;}total+=j;}}return total;}", true},
		{"while-condition-call", "fn i32 main(){var i32 i=0;while(below(i)){i+=1;}return i;}fn bool below(i32 n){return n<3;}", true},
		{"while-condition-effects", "fn i32 main(){var i32 i=0;while({i+=1;i<4}){}return i;}", true},
		{"while-condition-return", "fn i32 main(){while({return 9;true}){discard fail();}return 0;}fn i32 fail(){assert(false);return 99;}", true},
		{"while-condition-break-outer", "fn i32 main(){var i32 i=0;while(i<5){while({i+=1;break;true}){i=99;}}return i;}", true},
		{"while-condition-continue-outer", "fn i32 main(){var i32 i=0;var i32 n=0;while(i<3){while({i+=1;continue;true}){n+=1;}}return i*10+n;}", true},
		{"while-return-body", "fn i32 main(){while(true){return 9;}return 0;}", true},
		{"while-break-skips-call", "fn i32 main(){while(true){break;discard fail();}return 4;}fn i32 fail(){assert(false);return 99;}", true},
		{"while-overflow", "fn i32 main(){var i32 i=2147483647;while(true){i+=1;}return 0;}", true},
		{"for-continue-post", "fn i32 main(){var i32 total=0;for(var i32 i=0;i<5;i+=1){if(i==2){continue;}total+=i;}return total;}", true},
		{"for-break-skips-post", "fn i32 main(){var i32 i=0;for(;i<5;i+=1){if(i==1){break;}}return i;}", true},
		{"for-omitted-clauses", "fn i32 main(){var i32 i=0;for(;;){i+=1;if(i==4){break;}}return i;}", true},
		{"for-no-post-continue", "fn i32 main(){var i32 i=0;for(;i<4;){i+=1;continue;}return i;}", true},
		{"for-initial-assignment", "fn i32 main(){var i32 i=99;var i32 total=0;for(i=0;i<3;i=i+1){total+=i;}return total;}", true},
		{"for-zero-initial-once", "fn i32 main(){var i32 i=99;for(i=7;false;i+=1){i=100;}return i;}", true},
		{"for-always-break", "fn i32 main(){var i32 i=0;for(;true;i+=1){break;}return i;}", true},
		{"for-always-return", "fn i32 main(){for(var i32 i=0;true;i+=1){return 9;}return 0;}", true},
		{"for-post-overflow", "fn i32 main(){for(var i32 i=2147483647;true;i+=1){continue;}return 0;}", true},
		{"for-shadowed-local", "fn i32 main(){var i32 i=7;var i32 total=0;for(var i32 i=0;i<3;i+=1){total+=i;}return i+total;}", true},
		{"mixed-nested-loops", "fn i32 main(){var i32 total=0;for(var i32 i=0;i<3;i+=1){var i32 j=0;while(j<4){j+=1;if(j==2){continue;}if(j==4){break;}total+=j;}if(i==1){continue;}total+=10;}return total;}", true},
		{"loop-helper-call", "fn i32 main(){var i32 total=0;for(var i32 i=0;i<3;i+=1){total+=twice(i);}return total;}fn i32 twice(i32 x){return x*2;}", true},
	}
}
