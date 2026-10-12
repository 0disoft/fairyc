module tests::module_unit_loop_control;
import module_unit_loop_control as controls;
import module_unit_body_ast as ast;
import module_unit_value_model as m;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn syntax::Parsed parsed(ref u8[] text){let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let result=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref result)==0);return result;}
fn m::Problem checked(ref u8[] source,usize declaration){let input=parsed(source);var usize work=0;let body=switch(ast::build(ref input,declaration,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return m::Problem{.code=12,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};}};return controls::check(ref body,m::Frame{.kind=1,.context=0,.module_index=0,.declaration,.result=0},ref mut work);}
@test fn void rejects_controls_outside_a_loop_including_expression_blocks(){let a=checked(b"module main;fn i32 main(){break;return 0;}",2);assert(a.code==1012 && a.start==26 && a.end==32);let b=checked(b"module main;fn i32 main(){continue;return 0;}",2);assert(b.code==1012 && b.start==26 && b.end==35);assert(checked(b"module main;fn i32 main(){return if(true){{break;0}}else{1};}",2).code==1012);assert(checked(b"module main;fn i32 main(){while({break;true}){}return 0;}",2).code==1012);}
@test fn void preserves_nested_body_controls_and_does_not_share_function_scope(){assert(checked(b"module main;fn i32 main(){while(true){if(true){break;}continue;}return 0;}",2).code==0);assert(checked(b"module main;fn i32 main(){while(true){while({break;true}){}break;}return 0;}",2).code==0);assert(checked(b"module main;fn i32 main(){for(var usize i=0;i<4;i+=1){continue;break;}return 0;}",2).code==0);assert(checked(b"module main;fn void each(ref u8[] data){for(let u8 byte in data){continue;break;}}",2).code==0);let source=b"module main;fn i32 valid(){while(true){break;}return 0;}fn i32 invalid(){continue;return 0;}";assert(checked(source,2).code==0 && checked(source,3).code==1012);}
