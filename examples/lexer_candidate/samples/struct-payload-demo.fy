module main;

struct Token { i64 value; bool ready; }
struct Problem { u32 code; usize start; usize end; }

fn Result<Token,Problem> parse(bool valid) {
    return if(valid) {
        .ok(Token{.value=4000000000,.ready=true})
    } else {
        .err(Problem{.code=8,.start=2,.end=9})
    };
}

fn i32 inspect(Result<Token,Problem> input) {
    return switch(input) {
        case .ok(token): { assert(token.value==4000000000 && token.ready); 7 }
        case .err(problem): { assert(problem.code==8 && problem.start==2 && problem.end==9); 3 }
    };
}

fn i32 main() {
    assert(inspect(parse(true))==7);
    assert(inspect(parse(false))==3);
    return 7;
}
