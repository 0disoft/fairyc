module main;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_layout_index as layouts;
import module_unit_c_sdk as sdk;
import module_unit_c_enums as enums;
import module_unit_c_switches as switches;
import module_unit_cfg_model as cfg;
import c_text as w;
import std::bytes as bytes;
import std::io as io;

// Independent scanner: do not call the index or the production enum scanner.
fn usize scan(ref ir::Program p,usize type,ref mut usize visits,ref mut u32 code){
    if(type==0 || type>clean::type_count(ir::cleanup_ir(p))){*code=1001;return 0;}
    var usize found=0;var usize i=0;while(i<ir::layout_count(p)){*visits=*visits+1;if(ir::layout_at(p,i).value.type==type){if(found!=0){*code=1001;return 0;}found=i+1;}i+=1;}return found;
}
fn cfg::Block branch(){return cfg::Block{.kind=18,.source=5,.target=7,.first=11,.second=12,.drop=0,.start=0,.end=1};}
fn void compare(ref ir::Program p,ref layouts::Index index){
    let b=ir::at(p,1);var bytes::Buffer old=bytes::empty();var bytes::Buffer actual=bytes::empty();var w::State a=w::State{.code=0};var w::State z=w::State{.code=0};var usize old_work=0;var usize work=0;
    assert(enums::expression(p,b,3,ref mut old,ref mut a,ref mut old_work));assert(enums::expression_indexed(p,index,b,3,ref mut actual,ref mut z,ref mut work));
    assert(a.code==0 && z.code==0 && old_work==2050 && work==3);assert(bytes::equal(bytes::view(ref old),bytes::view(ref actual)));
    assert(bytes::equal(bytes::view(ref actual),b"fy_v_3=(struct fy_unit_t_2){.fy_tag=0};\nfy_v_3.fy_payload.fy_variant_2.fy_f_1=fy_v_1;\n"));
    old=bytes::empty();actual=bytes::empty();old_work=0;work=0;
    assert(switches::branch(p,b,branch(),ref mut old,ref mut a,ref mut old_work));assert(switches::branch_indexed(p,index,b,branch(),ref mut actual,ref mut z,ref mut work));
    assert(a.code==0 && z.code==0 && old_work==2050 && work==3);assert(bytes::equal(bytes::view(ref old),bytes::view(ref actual)));
    assert(bytes::equal(bytes::view(ref actual),b"if(fy_v_4.fy_tag==1){goto fy_b_11;\n}else{goto fy_b_12;\n}\n"));
}
@effects(console) fn i32 main(){
    let p=ir::c_layout_lookup_fixture(2048,0);var usize work=0;let plan=switch(sdk::build(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return 2;}};let index=sdk::layout_lookup(ref plan);
    // SDK construction already paid the index build once: 2050 types + 2048
    // layouts, then the existing two SDK type scans, with no SDK source units.
    assert(work==8198);var usize visits=0;var usize queries=0;var u32 code=0;var usize i=2;
    while(i<=2049){let expected=scan(ref p,i,ref mut visits,ref mut code);let actual=layouts::find(index,i,ref mut queries,ref mut code);assert(expected==actual && actual==2050-i);i+=1;}
    assert(code==0 && visits==4194304 && queries==2048);compare(ref p,index);
    visits=0;queries=0;assert(scan(ref p,2050,ref mut visits,ref mut code)==0);assert(layouts::find(index,2050,ref mut queries,ref mut code)==0 && code==0 && queries==1);
    i=0;while(i<2){queries=0;code=0;assert(layouts::find(index,if(i==0){0}else{2051},ref mut queries,ref mut code)==0 && code==1001 && queries==1);i+=1;}
    work=1048576;let exhausted=switch(layouts::build(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e==1002}};assert(exhausted && work==1048576);
    queries=1048576;code=0;assert(layouts::find(index,2,ref mut queries,ref mut code)==0 && code==1002 && queries==1048576);
    queries=0;code=1001;assert(layouts::find(index,2,ref mut queries,ref mut code)==0 && code==1001 && queries==0);
    var u32 mode=1;while(mode<=3){let malformed=ir::c_layout_lookup_fixture(2048,mode);work=0;let rejected=switch(sdk::build(ref malformed,ref mut work)){case .ok(_):{false}case .err(e):{e.code==1001}};assert(rejected);if(mode==1){visits=0;code=0;assert(scan(ref malformed,2049,ref mut visits,ref mut code)==0 && code==1001);}mode+=1;}
    mode=4;while(mode<=6){let malformed=ir::c_layout_lookup_fixture(2048,mode);work=0;let prepared=switch(sdk::build(ref malformed,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return 3;}};let b=ir::at(ref malformed,1);var bytes::Buffer out=bytes::empty();var w::State s=w::State{.code=0};work=0;
        assert(!enums::expression_indexed(ref malformed,sdk::layout_lookup(ref prepared),b,3,ref mut out,ref mut s,ref mut work) && s.code==1001 && bytes::len(ref out)==0);
        s.code=0;work=0;assert(!switches::branch_indexed(ref malformed,sdk::layout_lookup(ref prepared),b,branch(),ref mut out,ref mut s,ref mut work) && s.code==1001 && bytes::len(ref out)==0);mode+=1;
    }
    let b=ir::at(ref p,1);var bytes::Buffer out=bytes::empty();var w::State s=w::State{.code=0};work=1048576;
    assert(!enums::expression_indexed(ref p,index,b,3,ref mut out,ref mut s,ref mut work) && s.code==1002 && work==1048576 && bytes::len(ref out)==0);
    s.code=0;work=1048576;assert(!switches::branch_indexed(ref p,index,b,branch(),ref mut out,ref mut s,ref mut work) && s.code==1002 && work==1048576 && bytes::len(ref out)==0);
    discard io::write("layouts=2048 scan=4194304 index=2048 sdk=8198\n");discard io::write("enum=equal switch=equal work=2050->3 malformed=1001 budget=1002\n");return 0;
}
