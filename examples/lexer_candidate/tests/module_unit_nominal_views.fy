module tests::module_unit_nominal_views;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import module_types as types;
import module_catalog as catalog;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void nominal_views_retain_element_identity_and_slice_origins(){
    let p=switch(checked(b"module main;struct Span{usize n;}struct Token{i64 x;Span span;}fn ref Token[] select(ref Token[] a){let child=a[1..];let alias=child;return alias;}fn i64 shared(ref Token[] a){let ref Token item=ref a[0];return item.x;}fn i64 mutable(ref mut Token[] a){let ref mut Token item=ref mut a[0];return item.x;}fn usize length(ref Token[] a){return a.len;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==4 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let parameter=model::local_at(c,1);let view=values::type_at(ref p,parameter.type);assert(view.kind==if(i==2){18}else{17});let element=values::type_at(ref p,view.first);assert(element.kind==2 && element.module_index==0);var usize j=2;while(j<=model::local_count(c)){let local=model::local_at(c,j);let t=values::type_at(ref p,local.type);if(t.kind!=1){assert(local.origin.root==1 && local.origin.scope==1);}j+=1;}j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==9){assert(e.type==view.first && e.place.root==1 && e.place.mutable==(i==2));count+=1;}j+=1;}i+=1;}assert(count==2);
}
@test fn void nominal_array_fields_keep_plain_named_whole_owners(){
    let p=switch(checked(b"module main;struct Token{i64 x;}struct Page{Token[4] data;usize n;}fn i64 shared(ref Page p){let ref Token[] whole=ref (p.data);let range=ref ((p.data[1..]));let alias=range;let ref Token item=ref p.data[0];return whole[0].x+alias[0].x+item.x;}fn i64 mutable(ref mut Page p){let ref mut Token[] whole=ref mut p.data;let first=whole[0].x;let range=ref mut p.data[..2];let second=range[0].x;let ref mut Token item=ref mut p.data[0];return first+second+item.x;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(values::checked_returns(ref p)==2 && values::unresolved(ref p)==0);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=2;while(j<=model::local_count(c)){let local=model::local_at(c,j);let t=values::type_at(ref p,local.type);if(t.kind!=1){assert(local.origin.root==1 && local.origin.scope==1);}j+=1;}i+=1;}
}
@test fn void nominal_view_call_shortening_preserves_exact_local_types(){
    let p=switch(checked(b"module main;struct Token{i64 x;}fn i64 leaf(ref Token[] a){return a[0].x;}fn i64 main(ref mut Token[] a){let alias=a[1..];return leaf(alias);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::checked_arguments(ref p)==1 && values::shared_arguments(ref p)==1 && values::unresolved(ref p)==0);
}
@test fn void nominal_view_errors_preserve_reference_codes_and_spans(){
    failure(b"module main;struct Token{i64 x;}struct Other{i64 x;}struct Page{Token[2] data;}fn void main(ref Page p){let ref Other[] v=ref p.data;}",1006,b"ref p.data");
    failure(b"module main;struct Token{i64 x;}fn void main(ref mut Token[] a){let ref Token[] b=a;}",1006,b"a");
    failure(b"module main;struct Token{i64 x;}fn void leaf(ref Token[] a){}struct Other{i64 x;}fn void main(ref mut Other[] a){leaf(a);}",1006,b"a");
    failure(b"module main;struct Token{i64 x;}struct Page{Token[2] data;}fn void main(ref Page p){let v=ref mut p.data[..];}",1008,b"p.data");
    failure(b"module main;struct Token{i64 x;}fn void main(ref Token[] a){let v=ref mut a[0];}",1008,b"a[0]");
    failure(b"module main;struct Token{i64 x;}fn void main(ref Token[] a,bool low){let v=a[low..];}",1006,b"low");
    failure(b"module main;struct Token{i64 x;}fn void main(ref Token[] a){let v=ref a[..];}",2,b"ref a[..]");
    failure(b"module main;struct Token{i64 x;}struct Page{Token[2] data;}fn void main(ref Page p){let v=ref p.data;}",2,b"ref p.data");
    failure(b"module main;struct Token{i64 x;}struct Page{ptr i32 pointer;Token[2] data;}fn void main(ref Page p){let ref Token[] v=ref p.data;}",2,b"ref p.data");
    failure(b"module main;struct Token{i64 x;}struct Page{ptr i32 pointer;Token[2] data;}fn void main(ref Page p){let v=ref p.data[..];}",1008,b"p.data");
    failure(b"module main;struct Token{i64 x;}struct Page{Token[2] data;}fn void main(Page[2] pages){let ref Token[] v=ref pages[0].data;}",2,b"ref pages[0].data");
    failure(b"module main;struct Token{i64 x;}struct Page{ptr i32 pointer;Token[2] data;}fn void main(ref Page p){let v=ref p.data[0];}",2,b"ref p.data[0]");
    failure(b"module main;struct Token{i64 x;}struct Page{ptr i32 pointer;Token[2] data;}fn void main(ref mut Page p){let v=ref mut p.data[0];}",1008,b"p.data[0]");
    failure(b"module main;struct Token{i64 x;}struct Page{ptr i32 pointer;Token[2] data;}fn void main(ref Page p){let v=ref p.data[0].x;}",2,b"ref p.data[0].x");
    failure(b"module main;struct Token{i64 x;}struct Page{ptr i32 pointer;Token[2] data;}fn void main(ref mut Page p){let v=ref mut p.data[0].x;}",1008,b"p.data[0].x");
}
@test fn void nominal_views_retain_provider_identity_in_selected_units(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(ref p::Token[] a){return p::read(a);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;struct Span{usize n;}pub struct Token{i64 x;Span span;}pub fn i64 read(ref Token[] a){let alias=a[..];return alias[0].x;}" )).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_arguments(ref p)==1 && values::checked_returns(ref p)==1);let c=values::at(ref p,0);let t=values::type_at(ref p,model::local_at(c,1).type);assert(t.kind==17);let item=values::type_at(ref p,t.first);assert(item.kind==2 && item.module_index==1 && item.declaration==3);
}
@test fn void nominal_view_graph_rejects_forged_descriptor_children(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;struct Token{i64 x;}" )).is_ok);let c=switch(catalog::build(ref sources)){case .ok(c):{c}case .err(_):{assert(false);return;}};var v::Vec<types::Type> graph=v::empty<types::Type>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=2,.scalar=0,.module_index=0,.declaration=2,.first=0,.second=0}).is_ok);assert(v::push<types::Type>(ref mut graph,types::Type{.kind=17,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=0}).is_ok);assert(types::validate(ref c,ref graph)==0);var usize i=0;while(i<5){let slot=v::at_mut<types::Type>(ref mut graph,1);*slot=types::Type{.kind=if(i==4){18}else{17},.scalar=if(i==0){1}else{0},.module_index=if(i==1){1}else{0},.declaration=0,.first=if(i==2){2}else{1},.second=if(i==3 || i==4){1}else{0}};assert(types::validate(ref c,ref graph)==9);i+=1;}
}

@test fn void nominal_named_arrays_use_context_or_explicit_ranges(){
    let p=switch(checked(b"module main;struct Token{i64 x;}fn i64 main(Token[2] a){let ref Token[] whole=ref a;let range=ref a[1..];return whole[0].x+range[0].x;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);let whole=model::local_at(c,2);let range=model::local_at(c,3);assert(whole.type==range.type && values::type_at(ref p,whole.type).kind==17 && whole.origin.root==1 && range.origin.root==1);
}
