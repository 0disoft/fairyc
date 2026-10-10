module tests::module_unit_constructor_facts;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_origin_returns as origins;
import module_unit_loan_live_model as live;
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


@effects(filesystem) fn syntax::Parsed allocator(){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return allocator();}};let data=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return allocator();}};return parsed(bytes::view(ref data));}
@effects(filesystem,console) fn origins::Plan owned(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,allocator()).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return owned(root,provider);}};
    let typed=primary(ref sources,ref metadata);
    return switch(origins::build(ref sources,ref metadata,ref typed)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);assert(false);owned(root,provider)}};
}
@effects(console) fn void origin_name(ref origins::Plan p,usize frame,usize origin,ref u8[] root,ref u8[] provider){
    let input=origins::owned_inputs(p);if(origin==2){discard io::write("Static");return;}let id=origin-2;let owner=lifetime_inputs::owner(input,id);
    if(owner.kind==11){discard io::write("Static");return;}let f=lifetime_inputs::frame(input,frame);
    if((owner.kind==1 || owner.kind==13) && owner.borrowed){var usize i=0;while(i<f.parameter_count){if(lifetime_inputs::parameter(input,frame,i)==id){discard io::write(if(owner.kind==13){"RegionParam("}else{"Param("});discard io::write_usize(i);discard io::write(")");return;}i+=1;}assert(false);}
    else{discard io::write("Local(");let text=if(f.value.module_index==0){root}else{provider};discard io::write_bytes(text[owner.start..owner.end]);discard io::write(")");}
}
// Frozen Go BorrowFact source coordinates, kinds, last uses and escape flags.
fn void golden(ref origins::Plan p,ref usize[] wanted,usize count){assert(count*8<=wanted.len);var usize total=0;var usize frame=1;while(frame<=origins::count(p)){if(lifetime_inputs::frame(origins::owned_inputs(p),frame).value.module_index!=1){total+=origins::borrow_fact_count(p,frame);}frame+=1;}assert(total==count);var usize i=0;while(i<count){var bool found=false;frame=1;while(frame<=origins::count(p)){let input=origins::owned_inputs(p);let f=lifetime_inputs::frame(input,frame);var usize j=0;while(j<origins::borrow_fact_count(p,frame)){let r=origins::borrow_fact_at(p,frame,j);let at=i*8;let usize mutable=if(r.mutable){1}else{0};let usize escapes=if(r.escapes){1}else{0};if(f.value.module_index==wanted[at] && live::start(lifetime_inputs::liveness(input,frame))==wanted[at+1] && mutable==wanted[at+2] && r.start==wanted[at+3] && r.end==wanted[at+4] && r.last_start==wanted[at+5] && r.last_end==wanted[at+6] && escapes==wanted[at+7]){found=true;}j+=1;}frame+=1;}assert(found);i+=1;}}
@effects(filesystem,console) fn void facts(usize index,ref u8[] root,ref u8[] provider,ref usize[] wanted,usize count,u32 rejection){let p=owned(root,provider);golden(ref p,wanted,count);let checked=switch(diagnostics::collect(ref p)){case .ok(v):{v}case .err(_):{assert(false);return;}};assert(diagnostics::visible_count(ref checked)==if(rejection==0){0}else{1});if(rejection!=0){var usize at=0;while(at<diagnostics::count(ref checked)){if(diagnostics::visible_at(ref checked,at)){assert(diagnostics::at(ref checked,at).code==rejection);}at+=1;}}discard io::write("case ");discard io::write_usize(index);discard io::write("\n");var usize frame=1;while(frame<=origins::count(ref p)){let input=origins::owned_inputs(ref p);let f=lifetime_inputs::frame(input,frame);if(f.value.module_index==1){frame+=1;continue;}var usize i=0;while(i<origins::borrow_fact_count(ref p,frame)){let r=origins::borrow_fact_at(ref p,frame,i);assert(r.frame==frame);discard io::write("fact ");discard io::write_usize(f.value.module_index);discard io::write(" ");discard io::write_usize(live::start(lifetime_inputs::liveness(input,frame)));discard io::write(" ");discard io::write_u32(if(r.mutable){1}else{0});discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write(" ");discard io::write_usize(r.last_start);discard io::write(" ");discard io::write_usize(r.last_end);discard io::write(" ");discard io::write_u32(if(r.escapes){1}else{0});discard io::write(" ");origin_name(ref p,frame,r.origin,root,provider);discard io::write("\n");i+=1;}frame+=1;}}
@test @effects(filesystem,console) fn void constructor_facts_preserve_the_owner(){
    let usize[1] wanted0=[0];
    facts(0,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn void f(ref Buffer b){let x=unsafe{str::from_raw_parts(b,b.data,b.length)};discard x;}",b"",ref wanted0,0,0);
    let usize[8] wanted1=[0,69,0,103,149,103,149,1];
    facts(1,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref str f(ref Buffer b){return unsafe{str::from_raw_parts(b,b.data,b.length)};}",b"",ref wanted1,1,0);
    let usize[8] wanted2=[0,69,0,164,165,164,165,1];
    facts(2,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref u8[] f(ref Buffer b){let x=unsafe{byte_slice::from_raw_parts(b,b.data,b.length)};return x;}",b"",ref wanted2,1,0);
    let usize[8] wanted3=[0,69,1,112,169,112,169,1];
    facts(3,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref mut u8[] f(ref mut Buffer b){return unsafe{byte_slice::from_raw_parts_mut(b,b.data,b.length)};}",b"",ref wanted3,1,0);
    let usize[8] wanted4=[0,69,0,104,151,104,151,1];
    facts(4,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref CStr f(ref Buffer b){return unsafe{CStr::from_raw_parts(b,b.data,b.length)};}",b"",ref wanted4,1,0);
    let usize[8] wanted5=[0,69,0,156,161,188,189,0];
    facts(5,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn void f(){let Buffer b=Buffer{.data=null,.length=0};let x=unsafe{str::from_raw_parts(ref b,b.data,b.length)};discard x;}",b"",ref wanted5,1,0);
    let usize[8] wanted6=[0,69,1,167,176,203,204,0];
    facts(6,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn void f(){var Buffer b=Buffer{.data=null,.length=0};let x=unsafe{byte_slice::from_raw_parts_mut(ref mut b,b.data,b.length)};discard x;}",b"",ref wanted6,1,0);
    let usize[8] wanted7=[0,69,0,133,138,211,212,0];
    facts(7,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn void f(){let Buffer b=Buffer{.data=null,.length=0};let owner=ref b;let x=unsafe{byte_slice::from_raw_parts(owner,b.data,b.length)};discard x;}",b"",ref wanted7,1,0);
    let usize[8] wanted8=[0,69,1,180,186,195,196,0];
    facts(8,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn void f(ref mut Buffer b){let x=unsafe{byte_slice::from_raw_parts_mut(b,b.data,b.length)};let ref mut u8[] y=x[..1];discard y;}",b"",ref wanted8,1,0);
    let usize[1] wanted9=[0];
    facts(9,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn ref str f(){let Buffer b=Buffer{.data=null,.length=0};return unsafe{str::from_raw_parts(ref b,b.data,b.length)};}",b"",ref wanted9,0,2102);
    let usize[1] wanted10=[0];
    facts(10,b"module main;resource Buffer{ptr mut void data;usize length;cleanup{}}fn void f(ref Buffer b){discard unsafe{str::from_raw_parts(b,b.data,b.length)};}",b"",ref wanted10,0,0);
    let usize[1] wanted11=[0];
    facts(11,b"module main;import std::alloc as a;fn void f(Region r){discard r.place(1);}",b"",ref wanted11,0,0);
    let usize[1] wanted12=[0];
    facts(12,b"module main;import std::alloc as a;fn void f(Region r){let x=r.place(1);discard x;}",b"",ref wanted12,0,0);
    let usize[8] wanted13=[0,35,0,87,97,87,97,1];
    facts(13,b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r){return r.place(1);}",b"",ref wanted13,1,0);
    let usize[8] wanted14=[0,35,0,104,105,104,105,1];
    facts(14,b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r){let x=r.place(1);return x;}",b"",ref wanted14,1,0);
    let usize[8] wanted15=[0,35,0,105,111,105,111,1];
    facts(15,b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r){let x=r.place(1)?;return .ok(x);}",b"",ref wanted15,1,0);
    let usize[1] wanted16=[0];
    facts(16,b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(ref mut a::Arena arena){region r in arena{let x=r.place(1);return x;}}",b"",ref wanted16,0,3101);
    let usize[8] wanted17=[0,35,0,94,127,94,127,1];
    facts(17,b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r,bool c){return if(c){r.place(1)}else{r.place(2)};}",b"",ref wanted17,1,0);
    let usize[16] wanted18=[0,56,0,108,121,108,121,1,2,39,0,105,115,105,115,1];
    facts(18,b"module main;import std::alloc as a;import provider as p;fn Result<ref i32,a::AllocError> f(Region r){return p::store(r,3);}",b"module provider;import std::alloc as a;pub fn Result<ref i32,a::AllocError> store(Region r,i32 n){return r.place(n);}",ref wanted18,2,0);
}
