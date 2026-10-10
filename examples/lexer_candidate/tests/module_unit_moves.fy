module tests::module_unit_moves;
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
@test @effects(console) fn void resource_transfers_follow_reachable_branches_and_loop_edges(){
    // live
    let usize[1] wanted0=[0];
    check_case(0,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();observe(ref owner);}",b"",ref wanted0);
    // binding transfer
    let usize[14] wanted1=[1,1101,0,306,311,2,7,0,260,265,8,0,288,293];
    check_case(1,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();let Owner next=owner;observe(ref owner);}",b"",ref wanted1);
    // double call
    let usize[14] wanted2=[1,1101,0,290,295,2,7,0,260,265,8,0,278,283];
    check_case(2,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();take(owner);take(owner);}",b"",ref wanted2);
    // restore
    let usize[1] wanted3=[0];
    check_case(3,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();take(owner);owner=make();observe(ref owner);}",b"",ref wanted3);
    // self move
    let usize[1] wanted4=[0];
    check_case(4,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();owner=owner;observe(ref owner);}",b"",ref wanted4);
    // grouped transfer
    let usize[14] wanted5=[1,1101,0,308,313,2,7,0,260,265,8,0,289,294];
    check_case(5,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();let Owner next=(owner);observe(ref owner);}",b"",ref wanted5);
    // dead field read
    let usize[14] wanted6=[1,1101,0,293,298,2,7,0,260,265,8,0,278,283];
    check_case(6,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();take(owner);discard owner.value;}",b"",ref wanted6);
    // branch mismatch
    let usize[14] wanted7=[1,1104,0,273,292,2,7,0,260,265,8,0,284,289];
    check_case(7,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();if(c){take(owner);};}",b"",ref wanted7);
    // both arms consume
    let usize[1] wanted8=[0];
    check_case(8,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();if(c){take(owner);}else{take(owner);};}",b"",ref wanted8);
    // exited arm
    let usize[1] wanted9=[0];
    check_case(9,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();if(c){take(owner);return;};observe(ref owner);}",b"",ref wanted9);
    // shadow root
    let usize[1] wanted10=[0];
    check_case(10,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();if(c){let Owner owner=make();take(owner);};observe(ref owner);}",b"",ref wanted10);
    // existing MaybeLive
    let usize[14] wanted11=[1,1104,0,273,292,2,7,0,260,265,8,0,284,289];
    check_case(11,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();if(c){take(owner);};if(c){discard owner.value;};}",b"",ref wanted11);
    // loop mismatch
    let usize[18] wanted12=[1,1105,0,273,295,3,7,0,260,265,9,0,273,280,8,0,287,292];
    check_case(12,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){take(owner);}}",b"",ref wanted12);
    // loop restore
    let usize[1] wanted13=[0];
    check_case(13,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){take(owner);owner=make();}}",b"",ref wanted13);
    // break mismatch
    let usize[18] wanted14=[1,1105,0,273,301,3,7,0,260,265,9,0,273,280,8,0,287,292];
    check_case(14,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();while(c){take(owner);break;}}",b"",ref wanted14);
    // break restore
    let usize[1] wanted15=[0];
    check_case(15,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){take(owner);owner=make();break;}}",b"",ref wanted15);
    // condition recheck
    let usize[14] wanted16=[1,1101,0,288,293,2,7,0,260,265,8,0,288,293];
    check_case(16,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();while(consumed(owner)){continue;}}",b"",ref wanted16);
    // terminal suffix
    let usize[1] wanted17=[0];
    check_case(17,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();take(owner);return;observe(ref owner);}",b"",ref wanted17);
    // Option transfer
    let usize[14] wanted18=[1,1101,0,321,326,2,7,0,260,265,8,0,302,307];
    check_case(18,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();let Option<Owner> item=.some(owner);observe(ref owner);}",b"",ref wanted18);
    // Option capture
    let usize[14] wanted19=[1,1101,0,347,351,2,7,0,268,272,8,0,294,298];
    check_case(19,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Option<Owner> item=.some(make());switch(item){case .some(r):{take(r);}case .none:{}};discard item;}",b"",ref wanted19);
    // Result propagation
    let usize[14] wanted20=[1,1101,0,321,325,2,7,0,279,283,8,0,307,311];
    check_case(20,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn Result<Owner,i32> f(){let Result<Owner,i32> item=fallible();let Owner r=item?;discard item;return .ok(r);}",b"",ref wanted20);
    // borrowed parameter
    let usize[1] wanted21=[0];
    check_case(21,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(ref Owner owner){observe(owner);}",b"",ref wanted21);
    // consuming parameter
    let usize[14] wanted22=[1,1101,0,279,284,2,7,0,248,253,8,0,260,265];
    check_case(22,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(Owner owner){take(owner);observe(ref owner);}",b"",ref wanted22);
    // provider transfer
    let usize[14] wanted23=[1,1101,0,102,107,2,7,0,60,65,8,0,86,91];
    check_case(23,3,b"module main;import provider as pr;fn void f(){let pr::Owner owner=pr::make();pr::take(owner);pr::take(owner);}",b"module provider;pub resource Owner{cleanup{}}pub fn Owner make(){return Owner{};}pub fn void take(Owner p){}",ref wanted23);
    // short circuit mismatch
    let usize[14] wanted24=[1,1104,0,281,301,2,7,0,260,265,8,0,295,300];
    check_case(24,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();discard c && consumed(owner);observe(ref owner);}",b"",ref wanted24);
    // short circuit exited
    let usize[1] wanted25=[0];
    check_case(25,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();discard c && {take(owner);return;false};observe(ref owner);}",b"",ref wanted25);
    // conditional transfer
    let usize[14] wanted26=[1,1101,0,324,329,2,7,0,260,265,8,0,294,299];
    check_case(26,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();let Owner next=if(c){owner}else{owner};observe(ref owner);}",b"",ref wanted26);
    // tail transfer
    let usize[1] wanted27=[0];
    check_case(27,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn Owner f(){let Owner owner=make();return owner;}",b"",ref wanted27);
    // tail transfer double
    let usize[14] wanted28=[1,1101,0,287,292,2,7,0,255,260,8,0,273,278];
    check_case(28,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn Owner f(){let Owner owner=make();take(owner);return owner;}",b"",ref wanted28);
    // nested loop break
    let usize[1] wanted29=[0];
    check_case(29,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){while(c){take(owner);owner=make();break;}continue;}observe(ref owner);}",b"",ref wanted29);
    // nested loop mismatch
    let usize[18] wanted30=[1,1105,0,282,310,3,7,0,260,265,9,0,282,289,8,0,296,301];
    check_case(30,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){while(c){take(owner);break;}continue;}observe(ref owner);}",b"",ref wanted30);
    // continue restore
    let usize[1] wanted31=[0];
    check_case(31,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){take(owner);owner=make();continue;}observe(ref owner);}",b"",ref wanted31);
    // continue mismatch
    let usize[18] wanted32=[1,1105,0,273,304,3,7,0,260,265,9,0,273,280,8,0,287,292];
    check_case(32,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();while(c){take(owner);continue;}observe(ref owner);}",b"",ref wanted32);
    // for mismatch
    let usize[18] wanted33=[1,1105,0,273,312,3,7,0,260,265,9,0,273,292,8,0,304,309];
    check_case(33,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();for(var i32 i=0;i<2;i+=1){take(owner);}}",b"",ref wanted33);
    // for restore
    let usize[1] wanted34=[0];
    check_case(34,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();for(var i32 i=0;i<2;i+=1){take(owner);owner=make();continue;}observe(ref owner);}",b"",ref wanted34);
    // branch return move
    let usize[1] wanted35=[0];
    check_case(35,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn Owner f(bool c){let Owner owner=make();if(c){return owner;}else{};return owner;}",b"",ref wanted35);
    // result capture
    let usize[14] wanted36=[1,1101,0,327,328,2,7,0,310,311,8,0,319,320];
    check_case(36,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Result<Owner,i32> item=fallible();switch(item){case .ok(r):{take(r);take(r);}case .err(e):{}};}",b"",ref wanted36);
    // option scope shadow
    let usize[1] wanted37=[0];
    check_case(37,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner r=make();let Option<Owner> item=.some(make());switch(item){case .some(r):{take(r);}case .none:{}};observe(ref r);}",b"",ref wanted37);
    // discard read
    let usize[1] wanted38=[0];
    check_case(38,0,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();discard owner;observe(ref owner);}",b"",ref wanted38);
    // burn transfer
    let usize[14] wanted39=[1,1101,0,127,132,2,7,0,86,91,8,0,113,118];
    check_case(39,3,b"module main;@abilities(store)resource Owner{i32 value;cleanup{}}fn void f(){let Owner owner=Owner{.value=1};burn owner;discard owner;}",b"",ref wanted39);
    // record transfer
    let usize[14] wanted40=[1,1101,0,346,351,2,7,0,292,297,8,0,327,332];
    check_case(40,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}resource Pair{Owner a;cleanup{}}fn void f(bool c){let Owner owner=make();let Pair pair=Pair{.a=owner};observe(ref owner);}",b"",ref wanted40);
    // partial field move
    let usize[14] wanted41=[1,1103,0,317,323,2,10,0,252,253,6,0,291,295];
    check_case(41,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}resource Pair{Owner a;cleanup{}}fn void f(bool c){let Pair pair=Pair{.a=make()};take(pair.a);}",b"",ref wanted41);
    // cleanup consumes
    let usize[14] wanted42=[1,1102,0,50,54,2,7,0,37,57,8,0,50,54];
    check_case(42,3,b"module main;resource Owner{i32 value;cleanup{take(this);}}fn void take(Owner p){}",b"",ref wanted42);
    // cleanup consumes branch
    let usize[27] wanted43=[2,1104,0,45,72,2,7,0,37,74,8,0,59,63,1102,0,59,63,2,7,0,37,74,8,0,59,63];
    check_case(43,3,b"module main;resource Owner{i32 value;cleanup{if(true){take(this);}else{};}}fn void take(Owner p){}",b"",ref wanted43);
    // dead borrow
    let usize[14] wanted44=[1,1101,0,305,310,2,7,0,260,265,8,0,278,283];
    check_case(44,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){let Owner owner=make();take(owner);let ref Owner r=ref owner;}",b"",ref wanted44);
    // dead field write
    let usize[14] wanted45=[1,1101,0,285,290,2,7,0,260,265,8,0,278,283];
    check_case(45,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();take(owner);owner.value=3;}",b"",ref wanted45);
    // dead repeat restore
    let usize[27] wanted46=[2,1105,0,285,308,2,7,0,260,265,9,0,285,292,1101,0,320,325,2,7,0,260,265,8,0,278,283];
    check_case(46,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){var Owner owner=make();take(owner);while(c){owner=make();}observe(ref owner);}",b"",ref wanted46);
    // borrowed partial field
    let usize[14] wanted47=[1,1103,0,294,300,2,10,0,252,253,6,0,283,287];
    check_case(47,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}resource Pair{Owner a;cleanup{}}fn void f(ref Pair pair){take(pair.a);}",b"",ref wanted47);
    // provider partial field
    let usize[14] wanted48=[1,1103,1,162,168,2,10,1,67,68,6,1,131,135];
    check_case(48,3,b"module main;import provider as p;fn void f(){p::probe();}",b"module provider;resource Owner{i32 n;cleanup{}}resource Pair{Owner a;cleanup{}}fn void take(Owner p){}pub fn void probe(){let Pair pair=Pair{.a=Owner{.n=1}};take(pair.a);}",ref wanted48);
    // generic transfer
    let usize[14] wanted49=[1,1101,0,348,353,2,7,0,296,301,8,0,329,334];
    check_case(49,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void take_generic<T>(T n,Owner owner){}fn void f(){let Owner owner=make();take_generic<i32>(1,owner);observe(ref owner);}",b"",ref wanted49);
    // empty for mismatch
    let usize[18] wanted50=[1,1105,0,267,288,3,7,0,254,259,9,0,267,288,8,0,280,285];
    check_case(50,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(){let Owner owner=make();for(;;){take(owner);}}",b"",ref wanted50);
    // iteration mismatch
    let usize[18] wanted51=[1,1105,0,309,343,3,7,0,254,259,9,0,309,328,8,0,335,340];
    check_case(51,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(){let Owner owner=make();let i32[2] x=[1,2];let ref i32[] xs=ref x;for(let i32 n in xs){take(owner);}}",b"",ref wanted51);
    // branch local moved
    let usize[14] wanted52=[1,1101,0,296,301,2,7,0,266,271,8,0,284,289];
    check_case(52,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(bool c){if(c){let Owner owner=make();take(owner);take(owner);}else{};}",b"",ref wanted52);
    // two dead owners
    let usize[27] wanted53=[2,1101,0,310,311,2,7,0,254,255,8,0,287,288,1101,0,325,326,2,7,0,273,274,8,0,295,296];
    check_case(53,3,b"module main;resource Owner{i32 value;cleanup{}}fn Owner make(){return Owner{.value=1};}fn void take(Owner p){}fn void observe(ref Owner p){}fn bool consumed(Owner p){return false;}fn Result<Owner,i32> fallible(){return .ok(make());}fn void f(){let Owner a=make();let Owner b=make();take(a);take(b);observe(ref a);observe(ref b);}",b"",ref wanted53);
    // cast boundary: raw cast is not a safe call loan
    let usize[1] wanted54=[0];
    check_case(54,0,b"module main;resource R{usize length;cleanup{}}unsafe extern C fn u32 probe(ptr mut void data,usize length);fn void f(){var R x=R{.length=0};discard unsafe{probe(ref mut x as ptr mut void,x.length)};}",b"",ref wanted54);
    // cast boundary: raw cast checks borrow creation
    let usize[14] wanted55=[1,2101,0,179,188,2,5,0,224,225,4,0,152,157];
    check_case(55,3,b"module main;resource R{usize length;cleanup{}}unsafe extern C fn u32 probe(ptr mut void data,usize length);fn void f(){var R x=R{.length=0};let ref R p=ref x;discard unsafe{probe(ref mut x as ptr mut void,x.length)};discard p;}",b"",ref wanted55);
    let usize[14] wanted56=[1,1101,0,105,110,2,7,0,91,96,8,0,121,126];
    check_case(56,3,b"module main;resource R{i32 n;cleanup{}}fn i32 consume(R value){return 1;}fn void f(){var R owner=R{.n=0};owner.n=consume(owner);}",b"",ref wanted56);
    let usize[14] wanted57=[1,1101,0,106,111,2,7,0,91,96,8,0,123,128];
    check_case(57,3,b"module main;resource R{i32 n;cleanup{}}fn i32 consume(R value){return 1;}fn void f(){var R owner=R{.n=0};(owner).n=consume(owner);}",b"",ref wanted57);
}
@effects(console) fn moves::Plan retained(){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;resource Owner{i32 n;cleanup{}}fn void f(){let Owner first=Owner{.n=1};let Owner second=first;discard second;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained();}};let typed=primary(ref sources,ref metadata);switch(moves::build(ref sources,ref typed,1048576)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==10);}};return switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(_):{assert(false);retained()}};}
@test @effects(console) fn void move_facts_own_binding_identity_and_scope_transitions(){let p=retained();assert(moves::count(ref p)==0 && moves::fact_count(ref p)==6 && moves::work(ref p)<=1048576);let a=moves::fact_at(ref p,0);let b=moves::fact_at(ref p,1);let c=moves::fact_at(ref p,2);let d=moves::fact_at(ref p,3);let e=moves::fact_at(ref p,4);let f=moves::fact_at(ref p,5);assert(a.kind==3 && a.before==0 && a.after==1);assert(b.kind==2 && b.before==1 && b.after==2 && b.binding==a.binding && b.move_start==b.start && b.move_end==b.end);assert(c.kind==3 && c.before==0 && c.after==1 && c.binding!=a.binding);assert(d.kind==1 && d.before==1 && d.after==1 && d.binding==c.binding);assert(e.kind==4 && e.before==2 && e.after==0 && e.binding==a.binding);assert(f.kind==4 && f.before==1 && f.after==0 && f.binding==c.binding);assert(a.frame==b.frame && b.frame==c.frame && c.frame==d.frame && d.frame==e.frame && e.frame==f.frame);}
