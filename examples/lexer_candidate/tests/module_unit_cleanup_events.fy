module tests::module_unit_cleanup_events;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_moves as moves;
import module_unit_origin_returns as origins;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import module_unit_origin_check as checker;
import std::process as process;
import file_source as files;
import module_unit_origin_inputs as lifetime_inputs;
import module_unit_origin_flow_model as lifetime;
import module_unit_origin_sets as lifetime_sets;
import module_unit_origin_provenance as lifetime_sites;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);primary(sources,metadata)}};}
@effects(console) fn moves::Plan retained(ref u8[] root){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained(root);}};let typed=primary(ref sources,ref metadata);return switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);discard io::write_usize(p.start);assert(false);retained(root)}};}
@effects(console) fn void check_case(usize index,ref u8[] text,ref usize[] wanted){let p=retained(text);discard io::write("case ");discard io::write_usize(index);discard io::write(" ");discard io::write_usize(moves::cleanup_count(ref p));discard io::write("\n");if(wanted.len==1){assert(wanted[0]==0 && moves::cleanup_count(ref p)==0);return;}assert(moves::count(ref p)==0 && wanted.len%3==0 && moves::cleanup_count(ref p)==wanted.len/3);var usize i=0;while(i<moves::cleanup_count(ref p)){let e=moves::cleanup_at(ref p,i);assert(e.kind==5 && e.before==1 && e.after==1 && e.type!=0 && e.frame!=0 && e.start<e.end);var bool found=false;var usize j=0;while(j<wanted.len/3){found=found || e.exit as usize==wanted[j*3] && e.start==wanted[j*3+1] && e.end==wanted[j*3+2];j+=1;}discard io::write("cleanup ");discard io::write_u32(e.exit);discard io::write(" ");discard io::write_usize(e.start);discard io::write(" ");discard io::write_usize(e.end);discard io::write(" ");discard io::write_usize(e.binding);discard io::write(" ");discard io::write_usize(e.type);discard io::write("\n");assert(found);i+=1;}}
@test @effects(console) fn void cleanup_requests_follow_live_owners_and_exit_paths(){
    // local scope
    let usize[3] wanted0=[1,140,161];
    check_case(0,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();}",ref wanted0);
    // moved return
    let usize[1] wanted1=[0];
    check_case(1,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(){let R owner=make();return owner;}",ref wanted1);
    // parameter scope
    let usize[3] wanted2=[1,147,149];
    check_case(2,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(R owner){}",ref wanted2);
    // parameter move
    let usize[1] wanted3=[0];
    check_case(3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(R owner){return owner;}",ref wanted3);
    // nested scope
    let usize[3] wanted4=[1,149,171];
    check_case(4,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){discard {let R owner=make();0};}",ref wanted4);
    // early return
    let usize[3] wanted5=[2,160,167];
    check_case(5,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();return;}",ref wanted5);
    // break local
    let usize[3] wanted6=[3,175,181];
    check_case(6,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let R owner=make();break;}}",ref wanted6);
    // continue local
    let usize[3] wanted7=[4,175,184];
    check_case(7,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let R owner=make();continue;}}",ref wanted7);
    // outer owner survives break
    let usize[3] wanted8=[1,146,183];
    check_case(8,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){let R owner=make();while(c){break;}}",ref wanted8);
    // live replacement
    let usize[6] wanted9=[1,140,174,5,160,165];
    check_case(9,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){var R owner=make();owner=make();}",ref wanted9);
    // dead replacement
    let usize[3] wanted10=[2,188,201];
    check_case(10,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(){var R owner=make();let R saved=owner;owner=make();return saved;}",ref wanted10);
    // propagation error edge
    let usize[6] wanted11=[6,181,192,2,193,207];
    check_case(11,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn Result<i32,i32> f(){let R owner=make();let i32 n=fallible()?;return .ok(n);}",ref wanted11);
    // consumed is_ok
    let usize[3] wanted12=[7,184,194];
    check_case(12,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let Result<R,i32> item=.ok(make());discard item.is_ok;}",ref wanted12);
    // discarded temporary
    let usize[3] wanted13=[8,149,155];
    check_case(13,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){discard make();}",ref wanted13);
    // discard local read
    let usize[3] wanted14=[1,140,175];
    check_case(14,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();discard owner;}",ref wanted14);
    // match arm capture
    let usize[3] wanted15=[9,187,213];
    check_case(15,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let Option<R> item=.some(make());switch(item){case .some(r):{discard r;}case .none:{}};}",ref wanted15);
    // returned branch owner
    let usize[3] wanted16=[2,190,204];
    check_case(16,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(bool c){let R owner=make();if(c){return owner;}else{};return make();}",ref wanted16);
    // match break capture
    let usize[3] wanted17=[3,217,223];
    check_case(17,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let Option<R> item=.some(make());switch(item){case .some(r):{break;}case .none:{break;}};}}",ref wanted17);
}
