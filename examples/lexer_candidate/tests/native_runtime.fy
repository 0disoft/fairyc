module tests::native_runtime;
import native_runtime as runtime;
import native_runtime_contract as contract;
import std::bytes as bytes;
import std::fs as fs;
import std::process as process;
import std::text as text;

@effects(filesystem) fn usize number(ref mut fs::File file){
    var u8[4] raw=[0;4];assert(fs::read_exact(file,ref mut raw).is_ok);return raw[0] as usize+(raw[1] as usize)*256+(raw[2] as usize)*65536+(raw[3] as usize)*16777216;
}
@effects(filesystem) fn bytes::Buffer part(ref mut fs::File file){
    let usize n=number(file);assert(n<=2097152);var bytes::Buffer out=bytes::empty();assert(bytes::resize_zeroed(ref mut out,n).is_ok);assert(fs::read_exact(file,bytes::view_mut(ref mut out)).is_ok);return out;
}
@effects(filesystem) fn fs::File fixture(){
    var u8[4096] raw=[0;4096];let usize n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{assert(false);0}};
    let ref u8[] all=ref raw;let path=switch(text::validate_utf8(all[..n])){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};
    return switch(fs::open_read_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};
}
fn void append(ref mut bytes::Buffer output,ref u8[] value){assert(bytes::append(output,value).is_ok);}
fn void render(runtime::Plan plan,ref mut bytes::Buffer output){
    append(output,b"scalar:");assert(bytes::append_u32(output,plan.scalar_exit).is_ok);append(output,b"\n");var u32 id=1;
    while(id<=14){if(runtime::contains(ref plan,id)){append(output,contract::component_name(id));append(output,b"\n");}id+=1;}
    if(plan.vectors){append(output,b"vectors\n");}append(output,b"host:");assert(bytes::append_u32(output,plan.host).is_ok);append(output,b"\n");
}
@test @effects(filesystem) fn void matches_go_symbols_closures_order_and_diagnostics(){
    var fs::File file=fixture();var u8[12] magic=[0;12];assert(fs::read_exact(ref mut file,ref mut magic).is_ok && bytes::equal(ref magic,b"FYCRUNTIME1\n"));
    let usize count=number(ref mut file);assert(count>1000 && count<8192);var usize i=0;
    while(i<count){let usize mode=number(ref mut file);let bool trap=number(ref mut file)==1;let input=part(ref mut file);let expected=part(ref mut file);var bytes::Buffer actual=bytes::empty();
        if(mode==1){assert(bytes::append_u32(ref mut actual,runtime::symbol_kind(bytes::view(ref input))).is_ok);}
        else{switch(runtime::select(bytes::view(ref input),trap)){
            case .ok(plan):{render(plan,ref mut actual);}
            case .err(e):{append(ref mut actual,b"error:");assert(runtime::diagnostic(bytes::view(ref input),e,ref mut actual));}
        };}
        assert(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)));i+=1;
    }
    var u8[1] end=[0;1];let usize n=switch(fs::read(ref mut file,ref mut end)){case .ok(n):{n}case .err(_):{assert(false);1}};assert(n==0);assert(fs::close(file).is_ok);
}
@test fn void selection_is_inert_and_bounds_diagnostics(){
    switch(runtime::select(b"",false)){case .ok(plan):{assert(plan.components==0 && !plan.vectors && plan.host==0 && plan.scalar_exit==100);assert(!runtime::contains(ref plan,0) && !runtime::contains(ref plan,15));}case .err(_):{assert(false);}};
    switch(runtime::select(b"fy_private_vector_",true)){case .ok(plan):{assert(plan.components==64 && plan.vectors && plan.scalar_exit==128);}case .err(_):{assert(false);}};
    var bytes::Buffer out=bytes::empty();assert(!runtime::diagnostic(b"",runtime::Problem{.kind=102,.start=0,.end=1},ref mut out));assert(bytes::len(ref out)==0);
    assert(bytes::equal(contract::component_name(0),b"") && bytes::equal(contract::component_name(15),b""));
}
