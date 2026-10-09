module tests::module_unit_origin_paths;
import module_unit_origin_paths as paths;
import module_unit_value_model as m;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn m::Frame main_frame(ref syntax::Parsed p){var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3 && bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],b"main")){return m::Frame{.kind=3,.context=1,.module_index=0,.declaration=i+1,.result=0};}i+=1;}assert(false);return m::Frame{.kind=3,.context=1,.module_index=0,.declaration=0,.result=0};}
fn void marker(ref u8[] text,ref u8[] name,bool live){let p=parsed(text);var usize work=0;let observed=switch(paths::build(ref p,main_frame(ref p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=1;var bool found=false;while(i<=syntax::expression_count(ref p)){let e=syntax::expression_at(ref p,i);if(e.kind==2 && bytes::equal(syntax::expression_text(ref p,i),name)){let actual=switch(paths::live(ref observed,e.start,e.end,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(actual==live);found=true;}i+=1;}assert(found && work<1048576);}
@test fn void definite_returns_traps_and_branch_joins_exclude_suffixes(){
    marker(b"module main;fn void dead(){}fn void main(){return;dead();}",b"dead",false);
    marker(b"module main;fn void dead(){}fn void main(){assert(false);dead();}",b"dead",false);
    marker(b"module main;fn void dead(){}fn void main(){unsafe{return;};dead();}",b"dead",false);
    marker(b"module main;fn void dead(){}fn void main(){if(true){return;}else{return;};dead();}",b"dead",false);
    marker(b"module main;fn void alive(){}fn void main(){if(true){return;};alive();}",b"alive",true);
    marker(b"module main;enum E{A;B;}fn void dead(){}fn void main(){let E e=.A;switch(e){case .A:{return;}case .B:{return;}};dead();}",b"dead",false);
}
@test fn void loop_jumps_preserve_zero_iterations_and_for_post_reachability(){
    marker(b"module main;fn i32 post(){return 1;}fn void alive(){}fn void main(){for(var i32 n=0;n<1;n=post()){break;}alive();}",b"post",false);
    marker(b"module main;fn i32 post(){return 1;}fn void alive(){}fn void main(){for(var i32 n=0;n<1;n=post()){break;}alive();}",b"alive",true);
    marker(b"module main;fn i32 post(){return 1;}fn void dead(){}fn void main(){for(var i32 n=0;n<1;n=post()){continue;dead();}}",b"post",true);
    marker(b"module main;fn i32 post(){return 1;}fn void dead(){}fn void main(){for(var i32 n=0;n<1;n=post()){continue;dead();}}",b"dead",false);
    marker(b"module main;fn void alive(){}fn void dead(){}fn void main(){while(true){while(true){break;}alive();break;dead();}}",b"alive",true);
    marker(b"module main;fn void alive(){}fn void main(){for(let u8 b in b\"x\"){return;}alive();}",b"alive",true);
    marker(b"module main;fn void typed<T>(){}fn void alive(){}fn void main(){typed<i32>();let i32 n=1;discard n as i64;alive();}",b"alive",true);
}
@test fn void exhausted_work_is_an_error_not_a_dead_path(){
    let p=parsed(b"module main;fn void main(){return;}");let frame=main_frame(ref p);var usize work=1048576;switch(paths::build(ref p,frame,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.module_index==0);}};work=0;let observed=switch(paths::build(ref p,frame,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let d=syntax::declaration_at(ref p,frame.declaration-1);let b=syntax::statement_at(ref p,d.body);work=1048576;switch(paths::live(ref observed,b.start,b.end,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.start==b.start && e.end==b.end);}};
}
