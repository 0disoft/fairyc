"""Observe exact struct Vec storage through authenticated SDK ASTs."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "vector-structs"
observer.CASE_OBSERVER = Path(__file__)
TOKEN = """
struct Token{u32 kind;u32 trivia;usize start;usize end;usize cooked_index;}
fn Token token(u32 kind,usize start){return Token{.kind,.trivia=2,.start,.end=start+3,.cooked_index=5};}
"""
observer.CASES = {
    "token-copy-lifecycle": observer.FOREIGN + TOKEN + """
fn i32 main(){
    {
        var v::Vec<Token> x=v::empty<Token>();let Token original=token(11,17);
        assert(v::push<Token>(ref mut x,original).is_ok && original.kind==11 && original.start==17);
        assert(v::push<Token>(ref mut x,token(23,41)).is_ok);
        let usize capacity=v::capacity<Token>(ref x);assert(v::reserve<Token>(ref mut x,capacity+1).is_ok);
        let ref Token a=v::at<Token>(ref x,0);assert(a.kind==11 && a.trivia==2 && a.start==17 && a.end==20 && a.cooked_index==5);
        let ref mut Token b=v::at_mut<Token>(ref mut x,1);b.kind=37;b.end=47;assert(b.kind==37 && b.start==41 && b.end==47);
        assert(switch(v::pop<Token>(ref mut x)){case .some(t):{t.kind==37 && t.start==41 && t.end==47}case .none:{false}});
        assert(v::truncate<Token>(ref mut x,0)==1 && v::clear<Token>(ref mut x)==0);
        assert(switch(v::pop<Token>(ref mut x)){case .none:{true}case .some(_):{false}});
        assert(v::push<Token>(ref mut x,original).is_ok);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "struct-oom-preserves-copy": observer.FOREIGN + TOKEN + """
fn i32 main(){
    {
        var v::Vec<Token> x=v::empty<Token>();let Token original=token(11,17);unsafe{fixture_fail_next();}
        let Result<usize,Token> failed=v::push<Token>(ref mut x,original);
        assert(switch(failed){case .err(t):{t.kind==11 && t.start==17 && t.end==20}case .ok(_):{false}});
        assert(original.kind==11 && original.start==17 && v::len<Token>(ref x)==0 && v::capacity<Token>(ref x)==0);
        assert(v::push<Token>(ref mut x,original).is_ok);let usize capacity=v::capacity<Token>(ref x);
        unsafe{fixture_fail_next();}assert(!v::reserve<Token>(ref mut x,capacity+1).is_ok);
        assert(v::len<Token>(ref x)==1 && v::capacity<Token>(ref x)==capacity && v::at<Token>(ref x,0).end==20);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "padded-nested-struct": observer.FOREIGN + """
struct Padded{u8 byte;i64 wide;bool flag;}
struct Packet{Padded value;Option<i32> code;}
fn i32 main(){
    {
        var v::Vec<Packet> x=v::empty<Packet>();let Packet p=Packet{.value=Padded{.byte=255,.wide=-9223372036854775808,.flag=true},.code=.some(37)};
        assert(v::push<Packet>(ref mut x,p).is_ok);
        assert(switch(v::pop<Packet>(ref mut x)){case .some(t):{t.value.byte==255 && t.value.wide==-9223372036854775808 && t.value.flag && switch(t.code){case .some(n):{n==37}case .none:{false}}}case .none:{false}});
        assert(v::push<Packet>(ref mut x,p).is_ok);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "bounds-empty-struct": TOKEN + "fn i32 main(){var v::Vec<Token> x=v::empty<Token>();discard v::at<Token>(ref x,0);return 7;}",
}
observer.TRAPS = {"bounds-empty-struct": 5}
observer.REJECT = {
    "wrong-nominal-identity": (TOKEN + "struct Other{u32 kind;u32 trivia;usize start;usize end;usize cooked_index;}fn i32 main(){var v::Vec<Token> x=v::empty<Token>();discard v::push<Token>(ref mut x,Other{.kind=1,.trivia=2,.start=3,.end=4,.cooked_index=5});return 7;}", "phase 5 code 1"),
    "retained-struct-borrow": (TOKEN + "fn i32 main(){var v::Vec<Token> x=v::empty<Token>();assert(v::push<Token>(ref mut x,token(11,17)).is_ok);let ref Token a=v::at<Token>(ref x,0);discard v::clear<Token>(ref mut x);discard a.kind;return 7;}", "code 2101"),
    "struct-owned-field": ("resource Item{i32 n;cleanup{}}struct Bad{Item value;}fn i32 main(){var v::Vec<Bad> x=v::empty<Bad>();return 7;}", "phase 5 code 43"),
    "stolen-struct-storage": (TOKEN + "fn i32 main(){discard unsafe{storage_size<Token>()};return 7;}", "phase 5 code 5102"),
}

if __name__ == "__main__":
    observer.main()
