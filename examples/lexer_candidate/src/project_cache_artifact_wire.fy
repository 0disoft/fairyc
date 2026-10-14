module project_cache_artifact_wire;
import project_cache_keys as keys;
import project_cache_key as ordinary;
import project_generic_cache_key as generic;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;
import std::bytes as bytes;
import std::vec as v;

// Canonical key ledger and one whole-project IR member. This private envelope
// does not change Go's per-package formats or authorize a reuse by itself.
pub struct Parts{wire::Span keys;wire::Span program;}
pub fn usize key_limit(){return 8388608;}
fn wire::Span field(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){let n=wire::number_from(raw,c,work);let start=c.offset;if(c.code==0){if(n>key_limit()){c.code=1002;}else if(n==0 || start>raw.len || n>raw.len-start){c.code=1001;}else{c.offset+=n;}}return wire::Span{.start,.end=c.offset};}
fn usize count(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){let n=wire::number_from(raw,c,work);if(c.code==0){if(n>512){c.code=1002;}else if(n==0 || c.offset>raw.len || n>(raw.len-c.offset)/32){c.code=1001;}}return n;}
fn u32 ledger_shape(ref u8[] raw,ref mut usize work){var c=switch(wire::cursor_marker(raw,b"FYCKEYS0",work)){case .ok(p):{p}case .err(code):{return code;}};let n=count(raw,ref mut c,work);var usize i=0;while(i<n && c.code==0){var usize j=0;while(j<4 && c.code==0){discard field(raw,ref mut c,work);j+=1;}i+=1;}return wire::end(raw,ref mut c,work);}
pub fn Result<Parts,u32> preflight(ref u8[] raw,ref mut usize work){var c=switch(wire::cursor_marker(raw,b"FYCACHE0",work)){case .ok(p):{p}case .err(code):{return .err(code);}};let keys=wire::member_from(raw,ref mut c,key_limit(),b"FYCKEYS0",work);let program=wire::member_from(raw,ref mut c,wire::byte_limit(),b"FYPRJ000",work);let code=wire::end(raw,ref mut c,work);if(code!=0){return .err(code);}let checked=ledger_shape(raw[keys.start..keys.end],work);if(checked!=0){return .err(checked);}return .ok(Parts{.keys,.program});}
fn Result<v::Vec<usize>,u32> ordered(ref keys::Keys input,ref mut usize work){let n=keys::count(input);if(n==0 || n>512){return .err(1001);}var v::Vec<usize> order=v::empty<usize>();var usize i=0;while(i<n){if(!wire::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut order,i).is_ok){return .err(1003);}i+=1;}i=1;
    while(i<n){let current=*v::at<usize>(ref order,i);let a=keys::instance(keys::at(input,current));var usize at=i;while(at>0){let previous=*v::at<usize>(ref order,at-1);let b=keys::instance(keys::at(input,previous));if(!wire::charge(work,a.len+b.len+1)){return .err(1002);}if(!ordinary_before(a,b)){break;}{let slot=v::at_mut<usize>(ref mut order,at);*slot=previous;}at-=1;}{let slot=v::at_mut<usize>(ref mut order,at);*slot=current;}i+=1;}
    i=1;while(i<n){let a=keys::instance(keys::at(input,*v::at<usize>(ref order,i-1)));let b=keys::instance(keys::at(input,*v::at<usize>(ref order,i)));if(!wire::charge(work,a.len+b.len+1)){return .err(1002);}if(!ordinary_before(a,b)){return .err(1001);}i+=1;}return .ok(order);
}
fn bool ordinary_before(ref u8[] a,ref u8[] b){var usize i=0;while(i<a.len && i<b.len){if(a[i]!=b[i]){return a[i]<b[i];}i+=1;}return a.len<b.len;}
pub fn Result<payload::Payload,u32> encode_keys(ref keys::Keys input,ref mut usize work){let order=switch(ordered(input,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var wire::Status s=wire::Status{.code=0};var out=wire::writer_marker(b"FYCKEYS0",ref mut s,work);wire::number(ref mut out,keys::count(input),ref mut s,work);var usize i=0;
    while(i<keys::count(input) && s.code==0){let p=keys::at(input,*v::at<usize>(ref order,i));wire::member(ref mut out,keys::instance(p),ref mut s,work);wire::member(ref mut out,keys::coordinate(p),ref mut s,work);wire::member(ref mut out,ordinary::canonical(keys::ordinary_key(p)),ref mut s,work);wire::member(ref mut out,generic::canonical(keys::generic_key(p)),ref mut s,work);if(bytes::len(ref out)>key_limit()){return .err(1002);}i+=1;}return wire::finish(out,s);
}
fn u32 same(ref u8[] raw,wire::Span span,ref u8[] expected,ref mut usize work){if(span.start>span.end || span.end>raw.len){return 1001;}if(!wire::charge(work,expected.len+1)){return 1002;}return if(bytes::equal(raw[span.start..span.end],expected)){0}else{1001};}
pub fn u32 matches(ref u8[] raw,ref keys::Keys expected,ref mut usize work){let shape=ledger_shape(raw,work);if(shape!=0){return shape;}let order=switch(ordered(expected,work)){case .ok(p):{p}case .err(code):{return code;}};var c=switch(wire::cursor_marker(raw,b"FYCKEYS0",work)){case .ok(p):{p}case .err(code):{return code;}};let n=count(raw,ref mut c,work);if(c.code!=0){return c.code;}if(n!=keys::count(expected)){return 1001;}var usize i=0;
    while(i<n){let p=keys::at(expected,*v::at<usize>(ref order,i));var usize j=0;while(j<4){let part=field(raw,ref mut c,work);if(c.code!=0){return c.code;}let value=if(j==0){keys::instance(p)}else if(j==1){keys::coordinate(p)}else if(j==2){ordinary::canonical(keys::ordinary_key(p))}else{generic::canonical(keys::generic_key(p))};let code=same(raw,part,value,work);if(code!=0){return code;}j+=1;}i+=1;}return wire::end(raw,ref mut c,work);
}
pub fn Result<payload::Payload,u32> encode(ref keys::Keys keys,ref u8[] program,ref mut usize work){let ledger=switch(encode_keys(keys,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var wire::Status s=wire::Status{.code=0};var out=wire::writer_marker(b"FYCACHE0",ref mut s,work);wire::member(ref mut out,payload::content(ref ledger),ref mut s,work);wire::member(ref mut out,program,ref mut s,work);return wire::finish(out,s);}
