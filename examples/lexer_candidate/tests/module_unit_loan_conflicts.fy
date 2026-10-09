module tests::module_unit_loan_conflicts;
import module_unit_origin_diagnostics as diagnostics;
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
@effects(console) fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return checked(root,provider);}};let typed=primary(ref sources,ref metadata);let p=switch(origins::build(ref sources,ref metadata,ref typed)){case .ok(p):{p}case .err(p):{discard io::write("problem ");discard io::write_u32(p.code);discard io::write(" ");discard io::write_usize(p.start);discard io::write("\n");assert(false);return checked(root,provider);}};return switch(diagnostics::collect(ref p)){case .ok(p):{p}case .err(_):{assert(false);checked(root,provider)}};}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::visible_count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){if(!diagnostics::visible_at(p,i)){i+=1;continue;}let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}


// Exact Go diagnostic source anchors, independent of report insertion order.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len==diagnostics::visible_count(p)*7);var usize at=0;while(at<wanted.len){var bool found=false;var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(d.code==2101 && d.module_index==wanted[at] && d.start==wanted[at+1] && d.end==wanted[at+2] && d.related_count==2){let a=diagnostics::related_at(p,d,0);let b=diagnostics::related_at(p,d,1);let loan=if(a.role==4){a}else{b};let use=if(a.role==5){a}else{b};if(loan.role==4 && use.role==5 && loan.module_index==d.module_index && use.module_index==d.module_index && loan.start==wanted[at+3] && loan.end==wanted[at+4] && use.start==wanted[at+5] && use.end==wanted[at+6]){found=true;}}i+=1;}assert(found);at+=7;}}
@effects(console) fn void check_case(usize index,u32 expected,usize count,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected && diagnostics::visible_count(ref p)==count);if(count!=0){golden(ref p,wanted);}var usize i=0;while(i<diagnostics::count(ref p)){let d=diagnostics::at(ref p,i);assert(d.code==2101 && d.related_count==2);i+=1;}}
@test @effects(console) fn void owner_writes_and_borrow_creation_follow_live_loans(){
    let usize[7] wanted0=[0,306,318,292,300,328,329];
    check_case(0,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;edit(ref mut data);observe(p);}",b"",ref wanted0);
    let usize[7] wanted1=[0,317,325,296,308,332,333];
    check_case(1,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref mut u8[] p=ref mut data;observe(ref data);edit(p);}",b"",ref wanted1);
    let usize[7] wanted2=[0,308,320,292,300,308,320];
    check_case(2,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;both(p,ref mut data);}",b"",ref wanted2);
    let usize[7] wanted3=[0,309,321,292,300,322,323];
    check_case(3,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;reverse(ref mut data,p);}",b"",ref wanted3);
    let usize[7] wanted4=[0,291,303,282,290,291,303];
    check_case(4,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];both(ref data,ref mut data);}",b"",ref wanted4);
    let usize[7] wanted5=[0,300,312,287,299,300,312];
    check_case(5,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];exclusive(ref mut data,ref mut data);}",b"",ref wanted5);
    let usize[1] wanted6=[0];
    check_case(6,0,0,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;observe(p);edit(ref mut data);}",b"",ref wanted6);
    let usize[1] wanted7=[0];
    check_case(7,0,0,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];edit(ref mut data);edit(ref mut data);}",b"",ref wanted7);
    let usize[1] wanted8=[0];
    check_case(8,0,0,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;shared(p,ref data);}",b"",ref wanted8);
    let usize[1] wanted9=[0];
    check_case(9,0,0,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];shared(ref data,ref data);}",b"",ref wanted9);
    let usize[7] wanted10=[0,301,308,292,300,319,320];
    check_case(10,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;data[0]=2;observe(p);}",b"",ref wanted10);
    let usize[1] wanted11=[0];
    check_case(11,0,0,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];let ref u8[] p=ref data;observe(p);data[0]=2;}",b"",ref wanted11);
    let usize[21] wanted12=[0,291,315,282,290,291,315,0,292,299,282,290,292,299,0,302,314,282,290,302,314];
    check_case(12,3,3,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[1] data=[42];both(ref data,{data[0]=2;ref mut data});}",b"",ref wanted12);
    let usize[7] wanted13=[0,113,114,107,112,125,126];
    check_case(13,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;let ref i32 p=ref x;x=2;observe(p);}",b"",ref wanted13);
    let usize[1] wanted14=[0];
    check_case(14,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;let ref i32 p=ref x;observe(p);x=2;}",b"",ref wanted14);
    let usize[7] wanted15=[0,140,141,107,112,152,153];
    check_case(15,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;let ref i32 p=ref x;let ref i32 q=p;observe(p);x=2;observe(q);}",b"",ref wanted15);
    let usize[1] wanted16=[0];
    check_case(16,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;let ref i32 p=ref x;let ref i32 q=p;observe(q);x=2;}",b"",ref wanted16);
    let usize[1] wanted17=[0];
    check_case(17,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;var i32 y=2;var ref i32 p=ref x;p=ref y;x=3;observe(p);}",b"",ref wanted17);
    let usize[7] wanted18=[0,149,150,119,124,161,162];
    check_case(18,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;var i32 y=2;var ref i32 p=ref x;let ref i32 q=p;p=ref y;x=3;observe(q);observe(p);}",b"",ref wanted18);
    let usize[7] wanted19=[0,147,148,125,130,159,160];
    check_case(19,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(bool c){var i32 x=1;var i32 y=2;var ref i32 p=ref x;if(c){p=ref y;};x=3;observe(p);}",b"",ref wanted19);
    let usize[1] wanted20=[0];
    check_case(20,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(bool c){var i32 x=1;var i32 y=2;var ref i32 p=ref x;if(c){p=ref y;}else{p=ref y;};x=3;observe(p);}",b"",ref wanted20);
    let usize[1] wanted21=[0];
    check_case(21,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;let ref i32 p=ref x;return;x=3;observe(p);}",b"",ref wanted21);
    let usize[7] wanted22=[0,139,140,113,118,136,137];
    check_case(22,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(bool c){var i32 x=1;let ref i32 p=ref x;while(c){observe(p);x=2;}}",b"",ref wanted22);
    let usize[1] wanted23=[0];
    check_case(23,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(bool c){var i32 x=1;let ref i32 p=ref x;while(c){observe(p);x=2;break;}}",b"",ref wanted23);
    let usize[7] wanted24=[0,139,140,113,118,136,137];
    check_case(24,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(bool c){var i32 x=1;let ref i32 p=ref x;while(c){observe(p);x=2;continue;}}",b"",ref wanted24);
    let usize[7] wanted25=[0,128,129,121,126,139,140];
    check_case(25,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;let Option<ref i32> p=.some(ref x);x=2;switch(p){case .some(q):{observe(q);}case .none:{}};}",b"",ref wanted25);
    let usize[1] wanted26=[0];
    check_case(26,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn void f(){var i32 x=1;var Option<ref i32> p=.some(ref x);p=.none;x=2;discard p;}",b"",ref wanted26);
    let usize[7] wanted27=[0,152,153,145,150,164,165];
    check_case(27,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn ref i32 id(ref i32 p){return p;}fn void f(){var i32 x=1;let ref i32 p=id(ref x);x=2;observe(p);}",b"",ref wanted27);
    let usize[1] wanted28=[0];
    check_case(28,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn ref i32 second(ref i32 a,ref i32 b){return b;}fn void f(){var i32 x=1;var i32 y=2;let ref i32 p=second(ref x,ref y);x=3;observe(p);}",b"",ref wanted28);
    let usize[7] wanted29=[0,188,189,181,186,200,201];
    check_case(29,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}fn ref i32 second(ref i32 a,ref i32 b){return b;}fn void f(){var i32 x=1;var i32 y=2;let ref i32 p=second(ref x,ref y);y=3;observe(p);}",b"",ref wanted29);
    let usize[7] wanted30=[0,112,113,105,110,124,125];
    check_case(30,3,1,b"module main;import provider as p;fn void observe(ref i32 p){}fn void f(){var i32 x=1;let ref i32 r=p::id(ref x);x=2;observe(r);}",b"module provider;pub fn ref i32 id(ref i32 p){return p;}",ref wanted30);
    let usize[7] wanted31=[0,117,118,110,115,129,130];
    check_case(31,3,1,b"module main;import provider as p;fn void observe(ref i32 p){}fn void f(){var i32 x=1;let ref i32 r=p::id<i32>(ref x);x=2;observe(r);}",b"module provider;pub fn ref T id<T>(ref T p){return p;}",ref wanted31);
    let usize[7] wanted32=[0,300,307,294,298,300,307];
    check_case(32,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[2] data=[1,2];for(let u8 b in data){data[0]=3;break;}}",b"",ref wanted32);
    let usize[7] wanted33=[0,321,328,293,301,321,328];
    check_case(33,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[2] data=[1,2];let ref u8[] p=ref data;for(let u8 b in p){data[0]=3;break;}}",b"",ref wanted33);
    let usize[1] wanted34=[0];
    check_case(34,0,0,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(){var u8[2] data=[1,2];let ref u8[] p=ref data;for(let u8 b in p){discard b;break;}data[0]=3;}",b"",ref wanted34);
    let usize[7] wanted35=[0,383,391,319,328,380,381];
    check_case(35,3,1,b"module main;fn void observe(ref u8[] p){}fn void edit(ref mut u8[] p){}fn void both(ref u8[] p,ref mut u8[] q){}fn void reverse(ref mut u8[] p,ref u8[] q){}fn void shared(ref u8[] p,ref u8[] q){}fn void exclusive(ref mut u8[] p,ref mut u8[] q){}fn void f(bool c){var u8[2] data=[1,2];var u8[1] other=[3];let ref u8[] p=ref other;let ref u8[] q=ref data;for(let u8 b in q){observe(p);other[0]=4;if(c){continue;}}}",b"",ref wanted35);
    let usize[7] wanted36=[0,146,149,138,145,160,161];
    check_case(36,3,1,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}struct S{i32 a;i32 b;}fn void f(){var S s=S{.a=1,.b=2};let ref i32 p=ref s.a;s.b=3;observe(p);}",b"",ref wanted36);
    let usize[1] wanted37=[0];
    check_case(37,0,0,b"module main;fn void observe(ref i32 p){}fn void edit(ref mut i32 p){}struct S{i32 a;i32 b;}fn void f(){var S s=S{.a=1,.b=2};let ref i32 p=ref s.a;observe(p);s.b=3;}",b"",ref wanted37);
}
