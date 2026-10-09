module tests::module_unit_owner_moves;
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


// Source anchors are frozen from the Go reference and independently anchored
// call-held cases. Shared aliases deterministically report the furthest use.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len==diagnostics::visible_count(p)*9);var usize at=0;while(at<wanted.len){var bool found=false;var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(d.code==2105 && d.module_index==wanted[at] && d.start==wanted[at+1] && d.end==wanted[at+2] && d.related_count==3){var bool owner=false;var bool loan=false;var bool use=false;var usize j=0;while(j<3){let r=diagnostics::related_at(p,d,j);assert(r.module_index==d.module_index);if(r.role==6 && r.start==wanted[at+3] && r.end==wanted[at+4]){owner=true;}if(r.role==4 && r.start==wanted[at+5] && r.end==wanted[at+6]){loan=true;}if(r.role==5 && r.start==wanted[at+7] && r.end==wanted[at+8]){use=true;}j+=1;}if(owner && loan && use){found=true;}}i+=1;}assert(found);at+=9;}}
@effects(console) fn void check_case(usize index,u32 expected,usize count,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected && diagnostics::visible_count(ref p)==count);if(count!=0){golden(ref p,wanted);}var usize i=0;while(i<diagnostics::count(ref p)){let d=diagnostics::at(ref p,i);assert(d.code==2105 && d.related_count==3);i+=1;}}
@test @effects(console) fn void owner_moves_respect_retained_and_call_held_loans(){
    let usize[9] wanted0=[0,263,268,199,204,238,247,277,278];
    check_case(0,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let Owner next=owner;observe(p);}",b"",ref wanted0);
    let usize[1] wanted1=[0];
    check_case(1,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);let Owner next=owner;}",b"",ref wanted1);
    let usize[9] wanted2=[0,263,270,199,204,238,247,279,280];
    check_case(2,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let Owner next=(owner);observe(p);}",b"",ref wanted2);
    let usize[1] wanted3=[0];
    check_case(3,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);let Owner next=(owner);}",b"",ref wanted3);
    let usize[9] wanted4=[0,284,289,199,204,238,247,298,299];
    check_case(4,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;var Owner next=Owner{.value=0};next=owner;observe(p);}",b"",ref wanted4);
    let usize[1] wanted5=[0];
    check_case(5,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);var Owner next=Owner{.value=0};next=owner;}",b"",ref wanted5);
    let usize[9] wanted6=[0,256,261,199,204,238,247,271,272];
    check_case(6,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;consume(owner);observe(p);}",b"",ref wanted6);
    let usize[1] wanted7=[0];
    check_case(7,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);consume(owner);}",b"",ref wanted7);
    let usize[9] wanted8=[0,277,282,199,204,238,247,292,293];
    check_case(8,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let Option<Owner> next=.some(owner);observe(p);}",b"",ref wanted8);
    let usize[1] wanted9=[0];
    check_case(9,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);let Option<Owner> next=.some(owner);}",b"",ref wanted9);
    let usize[9] wanted10=[0,279,284,199,204,238,247,307,308];
    check_case(10,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let Result<Owner,i32> next=.ok(owner);discard next;observe(p);}",b"",ref wanted10);
    let usize[1] wanted11=[0];
    check_case(11,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);let Result<Owner,i32> next=.ok(owner);discard next;}",b"",ref wanted11);
    let usize[9] wanted12=[0,280,285,199,204,238,247,308,309];
    check_case(12,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let Result<i32,Owner> next=.err(owner);discard next;observe(p);}",b"",ref wanted12);
    let usize[1] wanted13=[0];
    check_case(13,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);let Result<i32,Owner> next=.err(owner);discard next;}",b"",ref wanted13);
    let usize[9] wanted14=[0,285,290,199,204,238,247,300,301];
    check_case(14,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let ref Owner q=p;observe(p);consume(owner);observe(q);}",b"",ref wanted14);
    let usize[1] wanted15=[0];
    check_case(15,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let ref Owner q=p;observe(q);consume(owner);}",b"",ref wanted15);
    let usize[18] wanted16=[0,282,287,199,204,238,247,297,298,0,282,287,199,204,264,273,308,309];
    check_case(16,3,2,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let ref Owner q=ref owner;consume(owner);observe(p);observe(q);}",b"",ref wanted16);
    let usize[9] wanted17=[0,274,279,199,204,238,247,300,301];
    check_case(17,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let ref Owner q=p;consume(owner);observe(p);observe(q);}",b"",ref wanted17);
    let usize[9] wanted18=[0,237,242,199,204,227,236,237,242];
    check_case(18,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};both(ref owner,owner);}",b"",ref wanted18);
    let usize[9] wanted19=[0,255,260,199,204,238,247,255,260];
    check_case(19,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;both(p,owner);}",b"",ref wanted19);
    let usize[9] wanted20=[0,246,251,199,204,227,236,246,251];
    check_case(20,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};both(ref owner,{consume(owner);Owner{.value=0}});}",b"",ref wanted20);
    let usize[1] wanted21=[0];
    check_case(21,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn Owner f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);return owner;}",b"",ref wanted21);
    let usize[1] wanted22=[0];
    check_case(22,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);return;consume(owner);observe(p);}",b"",ref wanted22);
    let usize[1] wanted23=[0];
    check_case(23,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};var Owner other=Owner{.value=1};var ref Owner p=ref owner;p=ref other;consume(owner);observe(p);}",b"",ref wanted23);
    let usize[9] wanted24=[0,314,319,205,210,276,285,329,330];
    check_case(24,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(bool c){var Owner owner=Owner{.value=42};var Owner other=Owner{.value=1};var ref Owner p=ref owner;if(c){p=ref other;};consume(owner);observe(p);}",b"",ref wanted24);
    let usize[1] wanted25=[0];
    check_case(25,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(bool c){var Owner owner=Owner{.value=42};var Owner other=Owner{.value=1};var ref Owner p=ref owner;if(c){p=ref other;}else{p=ref other;};consume(owner);observe(p);}",b"",ref wanted25);
    let usize[9] wanted26=[0,271,276,199,204,252,261,285,286];
    check_case(26,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let Option<ref Owner> p=.some(ref owner);consume(owner);switch(p){case .some(q):{observe(q);}case .none:{}};}",b"",ref wanted26);
    let usize[1] wanted27=[0];
    check_case(27,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};var Option<ref Owner> p=.some(ref owner);p=.none;consume(owner);discard p;}",b"",ref wanted27);
    let usize[9] wanted28=[0,273,278,199,204,254,263,287,288];
    check_case(28,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let Result<ref Owner,i32> p=.ok(ref owner);consume(owner);switch(p){case .ok(q):{observe(q);}case .err(_):{}};}",b"",ref wanted28);
    let usize[9] wanted29=[0,260,265,199,204,241,250,275,276];
    check_case(29,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=id(ref owner);consume(owner);observe(p);}",b"",ref wanted29);
    let usize[1] wanted30=[0];
    check_case(30,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};var Owner other=Owner{.value=1};let ref Owner p=ref owner;consume(other);observe(p);}",b"",ref wanted30);
    let usize[9] wanted31=[0,267,272,193,198,249,258,282,283];
    check_case(31,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(Owner input){var Owner owner=Owner{.value=42};let ref Owner p=ref input;consume(input);observe(p);}",b"",ref wanted31);
    let usize[9] wanted32=[0,320,327,240,247,285,296,336,337];
    check_case(32,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let Option<Owner> wrapped=.some(owner);let ref Option<Owner> p=ref wrapped;let Option<Owner> next=wrapped;discard p;}",b"",ref wanted32);
    let usize[9] wanted33=[0,260,265,199,204,236,251,276,277];
    check_case(33,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref i32 p=ref owner.value;consume(owner);discard *p;}",b"",ref wanted33);
    let usize[1] wanted34=[0];
    check_case(34,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref i32 p=ref owner.value;discard *p;consume(owner);}",b"",ref wanted34);
    let usize[9] wanted35=[0,305,310,234,239,280,295,321,322];
    check_case(35,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn ref T gid<T>(ref T p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref i32 p=gid<i32>(ref owner.value);consume(owner);discard *p;}",b"",ref wanted35);
    let usize[9] wanted36=[0,295,300,221,226,270,285,311,312];
    check_case(36,3,1,b"module main;import provider as pr;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(){var Owner owner=Owner{.value=42};let ref i32 p=pr::id<i32>(ref owner.value);consume(owner);discard *p;}",b"module provider;pub fn ref T id<T>(ref T p){return p;}",ref wanted36);
    let usize[9] wanted37=[0,303,308,247,252,285,294,318,319];
    check_case(37,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(bool c){var Owner owner=Owner{.value=42};while(c){var Owner inner=Owner{.value=1};let ref Owner p=ref inner;consume(inner);observe(p);break;}}",b"",ref wanted37);
    let usize[1] wanted38=[0];
    check_case(38,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void both(ref Owner p,Owner q){}fn ref Owner id(ref Owner p){return p;}fn void f(bool c){var Owner owner=Owner{.value=42};while(c){var Owner inner=Owner{.value=1};let ref Owner p=ref inner;observe(p);consume(inner);break;}}",b"",ref wanted38);
}
