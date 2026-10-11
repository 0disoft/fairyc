module project_json_tree;
import project_json_tokens as tokens;
import project_json_decode as decode;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

struct Frame{usize start;usize keys;u32 state;}
// Owned validated token tree. next skips an entire scalar or container; source
// offsets always address the original spelling, values the decoded pool.
pub resource Tree{tokens::Tokens tokens;v::Vec<usize> after;cleanup{}}
pub fn usize count(ref Tree tree){return tokens::count(ref tree.tokens);}
pub fn tokens::Token token(ref Tree tree,usize at){return tokens::at(ref tree.tokens,at);}
pub fn ref u8[] value(ref Tree tree,usize at){return tokens::value(ref tree.tokens,at);}
pub fn ref u8[] source(ref Tree tree){return tokens::source(ref tree.tokens);}
pub fn usize next(ref Tree tree,usize at){assert(at<count(tree));return *v::at<usize>(ref tree.after,at);}
fn decode::Problem fail(ref tokens::Tokens input,u32 code,usize at){let usize size=tokens::count(input);let usize start=if(at<size){tokens::at(input,at).start}else{tokens::source(input).len};let usize end=if(at<size){tokens::at(input,at).end}else{start};return decode::Problem{.code,.start,.end};}
fn void state(ref mut v::Vec<Frame> stack,u32 value){let usize index=v::len<Frame>(stack)-1;let ref mut Frame frame=v::at_mut<Frame>(stack,index);frame.state=value;}
fn void close(ref mut v::Vec<Frame> stack,ref mut v::Vec<usize> keys,ref mut v::Vec<usize> after,usize at){let frame=*v::at<Frame>(stack,v::len<Frame>(stack)-1);let ref mut usize slot=v::at_mut<usize>(after,frame.start);*slot=at+1;discard v::truncate<usize>(keys,frame.keys);let usize index=v::len<Frame>(stack)-1;discard v::truncate<Frame>(stack,index);}
fn Result<Tree,decode::Problem> validate(tokens::Tokens input,ref mut usize meter){let usize size=tokens::count(ref input);if(size==0){return .err(fail(ref input,1,0));}var v::Vec<usize> after=v::empty<usize>();var v::Vec<Frame> stack=v::empty<Frame>();var v::Vec<usize> keys=v::empty<usize>();var usize at=0;while(at<size){if(!work::charge(meter,1)){return .err(fail(ref input,3,at));}if(!v::push<usize>(ref mut after,at+1).is_ok){return .err(fail(ref input,2,at));}at+=1;}at=0;var bool root=false;
    while(at<size){if(!work::charge(meter,1)){return .err(fail(ref input,3,at));}let t=tokens::at(ref input,at);let usize depth=v::len<Frame>(ref stack);if(depth==0){if(root){return .err(fail(ref input,1,at));}root=true;}else{let frame=*v::at<Frame>(ref stack,depth-1);
            // Object: 1 initial key/end, 2 colon, 3 value, 4 comma/end, 5 key.
            // Array: 6 initial value/end, 7 comma/end, 8 value after comma.
            if(frame.state==1 || frame.state==5){if(t.kind==2 && frame.state==1){close(ref mut stack,ref mut keys,ref mut after,at);at+=1;continue;}if(t.kind!=7){return .err(fail(ref input,1,at));}var usize key=frame.keys;while(key<v::len<usize>(ref keys)){let previous=*v::at<usize>(ref keys,key);let ref u8[] a=tokens::value(ref input,previous);let ref u8[] b=tokens::value(ref input,at);if(!work::charge(meter,a.len+b.len+1)){return .err(fail(ref input,3,at));}if(bytes::equal(a,b)){return .err(fail(ref input,4,at));}key+=1;}if(!v::push<usize>(ref mut keys,at).is_ok){return .err(fail(ref input,2,at));}state(ref mut stack,2);at+=1;continue;}
            if(frame.state==2){if(t.kind!=5){return .err(fail(ref input,1,at));}state(ref mut stack,3);at+=1;continue;}
            if(frame.state==4 || frame.state==7){let u32 closing=if(frame.state==4){2}else{4};if(t.kind==closing){close(ref mut stack,ref mut keys,ref mut after,at);at+=1;continue;}if(t.kind!=6){return .err(fail(ref input,1,at));}state(ref mut stack,if(frame.state==4){5}else{8});at+=1;continue;}
            if(frame.state==6 && t.kind==4){close(ref mut stack,ref mut keys,ref mut after,at);at+=1;continue;}
            state(ref mut stack,if(frame.state==3){4}else{7});
        }
        if(t.kind==1 || t.kind==3){if(depth>=512){return .err(fail(ref input,3,at));}if(!v::push<Frame>(ref mut stack,Frame{.start=at,.keys=v::len<usize>(ref keys),.state=if(t.kind==1){1}else{6}}).is_ok){return .err(fail(ref input,2,at));}}
        else if(t.kind<7 || t.kind>11){return .err(fail(ref input,1,at));}at+=1;
    }if(v::len<Frame>(ref stack)!=0){return .err(fail(ref input,1,size));}return .ok(Tree{.tokens=input,.after});
}
// Codes: 1 malformed/trailing JSON, 2 allocation, 3 input/work/depth bound,
// 4 duplicate decoded key in one object. Nesting uses a bounded explicit stack.
pub fn Result<Tree,decode::Problem> parse(bytes::Buffer source,ref mut usize meter){let input=switch(tokens::scan(source,meter)){case .ok(value):{value}case .err(p):{return .err(p);}};return validate(input,meter);}
