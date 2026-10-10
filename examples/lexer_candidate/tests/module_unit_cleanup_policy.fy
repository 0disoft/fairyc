module tests::module_unit_cleanup_policy;
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
@effects(console) fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}return switch(checker::check(ref sources)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);assert(false);checked(root,provider)}};}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::visible_count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){if(!diagnostics::visible_at(p,i)){i+=1;continue;}let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}


// Complete frozen reference coordinates; each row includes all related spans.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len!=0 && diagnostics::visible_count(p)==wanted[0]);var usize row=0;var usize offset=1;
    while(row<wanted[0]){assert(offset+5<=wanted.len);let usize related=wanted[offset+4];assert(offset+5+related*4<=wanted.len);var bool found=false;var usize i=0;
        while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(diagnostics::visible_at(p,i) && d.code as usize==wanted[offset] && d.module_index==wanted[offset+1] && d.start==wanted[offset+2] && d.end==wanted[offset+3] && d.related_count==related){var usize j=0;var bool complete=true;while(j<related){var bool matched=false;var usize k=0;while(k<d.related_count){let r=diagnostics::related_at(p,d,k);let at=offset+5+j*4;matched=matched || r.role as usize==wanted[at] && r.module_index==wanted[at+1] && r.start==wanted[at+2] && r.end==wanted[at+3];k+=1;}complete=complete && matched;j+=1;}found=found || complete;}i+=1;}assert(found);offset+=5+related*4;row+=1;
    }assert(offset==wanted.len);
}
@effects(console) fn void check_case(usize index,u32 expected,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected);golden(ref p,wanted);}
@test @effects(console) fn void cleanup_policy_enforces_abilities_and_effect_authority(){
    // local scope no drop
    let usize[10] wanted0=[1,1108,0,140,161,1,11,0,12,29];
    check_case(0,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();}",b"",ref wanted0);
    // moved return no drop
    let usize[1] wanted1=[0];
    check_case(1,0,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(){let R owner=make();return owner;}",b"",ref wanted1);
    // parameter scope no drop
    let usize[10] wanted2=[1,1108,0,147,149,1,11,0,12,29];
    check_case(2,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(R owner){}",b"",ref wanted2);
    // parameter move no drop
    let usize[1] wanted3=[0];
    check_case(3,0,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(R owner){return owner;}",b"",ref wanted3);
    // nested scope no drop
    let usize[10] wanted4=[1,1108,0,149,171,1,11,0,12,29];
    check_case(4,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){discard {let R owner=make();0};}",b"",ref wanted4);
    // early return no drop
    let usize[10] wanted5=[1,1108,0,160,167,1,11,0,12,29];
    check_case(5,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();return;}",b"",ref wanted5);
    // break local no drop
    let usize[10] wanted6=[1,1108,0,175,181,1,11,0,12,29];
    check_case(6,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let R owner=make();break;}}",b"",ref wanted6);
    // continue local no drop
    let usize[10] wanted7=[1,1108,0,175,184,1,11,0,12,29];
    check_case(7,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let R owner=make();continue;}}",b"",ref wanted7);
    // outer owner survives break no drop
    let usize[10] wanted8=[1,1108,0,146,183,1,11,0,12,29];
    check_case(8,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){let R owner=make();while(c){break;}}",b"",ref wanted8);
    // live replacement no drop
    let usize[19] wanted9=[2,1108,0,140,174,1,11,0,12,29,1108,0,160,165,1,11,0,12,29];
    check_case(9,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){var R owner=make();owner=make();}",b"",ref wanted9);
    // dead replacement no drop
    let usize[10] wanted10=[1,1108,0,188,201,1,11,0,12,29];
    check_case(10,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(){var R owner=make();let R saved=owner;owner=make();return saved;}",b"",ref wanted10);
    // propagation error edge no drop
    let usize[19] wanted11=[2,1108,0,181,192,1,11,0,12,29,1108,0,193,207,1,11,0,12,29];
    check_case(11,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn Result<i32,i32> f(){let R owner=make();let i32 n=fallible()?;return .ok(n);}",b"",ref wanted11);
    // consumed is_ok no drop
    let usize[10] wanted12=[1,1108,0,184,194,1,11,0,12,29];
    check_case(12,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let Result<R,i32> item=.ok(make());discard item.is_ok;}",b"",ref wanted12);
    // discarded temporary no drop
    let usize[10] wanted13=[1,1108,0,149,155,1,11,0,12,29];
    check_case(13,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){discard make();}",b"",ref wanted13);
    // discard local read no drop
    let usize[10] wanted14=[1,1108,0,140,175,1,11,0,12,29];
    check_case(14,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();discard owner;}",b"",ref wanted14);
    // match arm capture no drop
    let usize[10] wanted15=[1,1108,0,187,213,1,11,0,12,29];
    check_case(15,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let Option<R> item=.some(make());switch(item){case .some(r):{discard r;}case .none:{}};}",b"",ref wanted15);
    // returned branch owner no drop
    let usize[10] wanted16=[1,1108,0,190,204,1,11,0,12,29];
    check_case(16,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(bool c){let R owner=make();if(c){return owner;}else{};return make();}",b"",ref wanted16);
    // match break capture no drop
    let usize[10] wanted17=[1,1108,0,217,223,1,11,0,12,29];
    check_case(17,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let Option<R> item=.some(make());switch(item){case .some(r):{break;}case .none:{break;}};}}",b"",ref wanted17);
    // local scope missing effect
    let usize[10] wanted18=[1,7102,0,150,171,1,12,0,12,29];
    check_case(18,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();}",b"",ref wanted18);
    // moved return missing effect
    let usize[1] wanted19=[0];
    check_case(19,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(){let R owner=make();return owner;}",b"",ref wanted19);
    // parameter scope missing effect
    let usize[10] wanted20=[1,7102,0,157,159,1,12,0,12,29];
    check_case(20,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(R owner){}",b"",ref wanted20);
    // parameter move missing effect
    let usize[1] wanted21=[0];
    check_case(21,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(R owner){return owner;}",b"",ref wanted21);
    // nested scope missing effect
    let usize[10] wanted22=[1,7102,0,159,181,1,12,0,12,29];
    check_case(22,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){discard {let R owner=make();0};}",b"",ref wanted22);
    // early return missing effect
    let usize[10] wanted23=[1,7102,0,170,177,1,12,0,12,29];
    check_case(23,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();return;}",b"",ref wanted23);
    // break local missing effect
    let usize[10] wanted24=[1,7102,0,185,191,1,12,0,12,29];
    check_case(24,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let R owner=make();break;}}",b"",ref wanted24);
    // continue local missing effect
    let usize[10] wanted25=[1,7102,0,185,194,1,12,0,12,29];
    check_case(25,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let R owner=make();continue;}}",b"",ref wanted25);
    // outer owner survives break missing effect
    let usize[10] wanted26=[1,7102,0,156,193,1,12,0,12,29];
    check_case(26,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){let R owner=make();while(c){break;}}",b"",ref wanted26);
    // live replacement missing effect
    let usize[19] wanted27=[2,7102,0,150,184,1,12,0,12,29,7102,0,170,175,1,12,0,12,29];
    check_case(27,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){var R owner=make();owner=make();}",b"",ref wanted27);
    // dead replacement missing effect
    let usize[10] wanted28=[1,7102,0,198,211,1,12,0,12,29];
    check_case(28,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(){var R owner=make();let R saved=owner;owner=make();return saved;}",b"",ref wanted28);
    // propagation error edge missing effect
    let usize[19] wanted29=[2,7102,0,191,202,1,12,0,12,29,7102,0,203,217,1,12,0,12,29];
    check_case(29,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn Result<i32,i32> f(){let R owner=make();let i32 n=fallible()?;return .ok(n);}",b"",ref wanted29);
    // consumed is_ok missing effect
    let usize[10] wanted30=[1,7102,0,194,204,1,12,0,12,29];
    check_case(30,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let Result<R,i32> item=.ok(make());discard item.is_ok;}",b"",ref wanted30);
    // discarded temporary missing effect
    let usize[10] wanted31=[1,7102,0,159,165,1,12,0,12,29];
    check_case(31,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){discard make();}",b"",ref wanted31);
    // discard local read missing effect
    let usize[10] wanted32=[1,7102,0,150,185,1,12,0,12,29];
    check_case(32,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let R owner=make();discard owner;}",b"",ref wanted32);
    // match arm capture missing effect
    let usize[10] wanted33=[1,7102,0,197,223,1,12,0,12,29];
    check_case(33,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(){let Option<R> item=.some(make());switch(item){case .some(r):{discard r;}case .none:{}};}",b"",ref wanted33);
    // returned branch owner missing effect
    let usize[10] wanted34=[1,7102,0,200,214,1,12,0,12,29];
    check_case(34,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn R f(bool c){let R owner=make();if(c){return owner;}else{};return make();}",b"",ref wanted34);
    // match break capture missing effect
    let usize[10] wanted35=[1,7102,0,227,233,1,12,0,12,29];
    check_case(35,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}fn void f(bool c){while(c){let Option<R> item=.some(make());switch(item){case .some(r):{break;}case .none:{break;}};}}",b"",ref wanted35);
    // local scope authorized
    let usize[1] wanted36=[0];
    check_case(36,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(){let R owner=make();}",b"",ref wanted36);
    // early return authorized
    let usize[1] wanted37=[0];
    check_case(37,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(){let R owner=make();return;}",b"",ref wanted37);
    // break local authorized
    let usize[1] wanted38=[0];
    check_case(38,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(bool c){while(c){let R owner=make();break;}}",b"",ref wanted38);
    // live replacement authorized
    let usize[1] wanted39=[0];
    check_case(39,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(){var R owner=make();owner=make();}",b"",ref wanted39);
    // propagation error edge authorized
    let usize[1] wanted40=[0];
    check_case(40,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn Result<i32,i32> f(){let R owner=make();let i32 n=fallible()?;return .ok(n);}",b"",ref wanted40);
    // consumed is_ok authorized
    let usize[1] wanted41=[0];
    check_case(41,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(){let Result<R,i32> item=.ok(make());discard item.is_ok;}",b"",ref wanted41);
    // discarded temporary authorized
    let usize[1] wanted42=[0];
    check_case(42,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(){discard make();}",b"",ref wanted42);
    // match arm capture authorized
    let usize[1] wanted43=[0];
    check_case(43,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(){let Option<R> item=.some(make());switch(item){case .some(r):{discard r;}case .none:{}};}",b"",ref wanted43);
    // match break capture authorized
    let usize[1] wanted44=[0];
    check_case(44,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn Result<i32,i32> fallible(){return .ok(1);}@effects(console)fn void f(bool c){while(c){let Option<R> item=.some(make());switch(item){case .some(r):{break;}case .none:{break;}};}}",b"",ref wanted44);
    // nested resource blocker
    let usize[10] wanted45=[1,1108,0,126,159,1,11,0,12,29];
    check_case(45,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}resource Box{R inner;cleanup{}}fn void f(){let Box box=Box{.inner=make()};}",b"",ref wanted45);
    // repeated field blocker
    let usize[10] wanted46=[1,1108,0,126,165,1,11,0,12,29];
    check_case(46,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}resource Box{R a;R b;cleanup{}}fn void f(){let Box box=Box{.a=make(),.b=make()};}",b"",ref wanted46);
    // repeated local blocker
    let usize[10] wanted47=[1,1108,0,95,127,1,11,0,12,29];
    check_case(47,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){let R a=make();let R b=make();}",b"",ref wanted47);
    // nested Option blocker
    let usize[10] wanted48=[1,1108,0,95,145,1,11,0,12,29];
    check_case(48,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){let Option<Option<R>> item=.some(.some(make()));}",b"",ref wanted48);
    // repeated Result blocker
    let usize[10] wanted49=[1,1108,0,137,147,1,11,0,12,29];
    check_case(49,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){let Result<R,R> item=.ok(make());discard item.is_ok;}",b"",ref wanted49);
    // cleanup local missing effect
    let usize[10] wanted50=[1,7102,0,122,139,1,12,0,12,29];
    check_case(50,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}resource Outer{i32 n;cleanup{let R r=make();}}",b"",ref wanted50);
    // cleanup temporary missing effect
    let usize[10] wanted51=[1,7102,0,131,137,1,12,0,12,29];
    check_case(51,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}resource Outer{i32 n;cleanup{discard make();}}",b"",ref wanted51);
    // cleanup temporary no drop
    let usize[10] wanted52=[1,1108,0,121,127,1,11,0,12,29];
    check_case(52,3,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}resource Outer{i32 n;cleanup{discard make();}}",b"",ref wanted52);
    // provider missing effect
    let usize[10] wanted53=[1,7102,0,44,67,1,12,1,16,33];
    check_case(53,3,b"module main;import provider as p;fn void f(){let p::R r=p::make();}",b"module provider;@effects(console)pub resource R{i32 n;cleanup{discard 1;}}pub fn R make(){return R{.n=1};}",ref wanted53);
    // provider no drop
    let usize[10] wanted54=[1,1108,0,44,67,1,11,1,16,33];
    check_case(54,3,b"module main;import provider as p;fn void f(){let p::R r=p::make();}",b"module provider;@abilities(store)pub resource R{i32 n;cleanup{}}pub fn R make(){return R{.n=1};}",ref wanted54);
    // nested loop retains captured no drop
    let usize[1] wanted55=[0];
    check_case(55,0,b"module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn R f(bool c){while(c){let Option<R> item=.some(make());switch(item){case .some(r):{while(c){break;}return r;}case .none:{break;}};}return make();}",b"",ref wanted55);
    // generic cleanup requires effects
    let usize[10] wanted56=[1,7102,0,125,127,1,12,0,12,29];
    check_case(56,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}fn void dispose<T>(T n,R owner){}fn void f(){dispose<i32>(1,make());}",b"",ref wanted56);
    // generic cleanup with authority
    let usize[1] wanted57=[0];
    check_case(57,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}@effects(console)fn void dispose<T>(T n,R owner){}@effects(console)fn void f(){dispose<i32>(1,make());}",b"",ref wanted57);
}
