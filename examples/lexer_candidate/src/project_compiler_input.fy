module project_compiler_input;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import project_path as work;
import candidate_limits as limits;
import module_catalog as catalog;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Origin{usize owner;usize module_index;usize identity_start;usize identity_end;usize path_start;usize path_end;}
pub resource Input{v::Vec<syntax::Parsed> sources;v::Vec<usize> owners;v::Vec<Origin> origins;v::Vec<catalog::Binding> bindings;bytes::Buffer names;cleanup{}}
// 1 bounds/work, 2 allocation, 3 exact arena copy, 4 invalid resolved coordinate.
pub struct Problem{u32 code;u32 reason;usize owner;usize module_index;usize declaration;}
fn Problem fail(u32 code,u32 reason,usize owner,usize module_index,usize declaration){return Problem{.code,.reason,.owner,.module_index,.declaration};}
pub fn usize count(ref Input input){return v::len<syntax::Parsed>(ref input.sources);}
pub fn ref v::Vec<syntax::Parsed> sources(ref Input input){return ref input.sources;}
pub fn ref syntax::Parsed source(ref Input input,usize at){assert(at<count(input));return v::at<syntax::Parsed>(ref input.sources,at);}
pub fn ref v::Vec<usize> owners(ref Input input){return ref input.owners;}
pub fn ref v::Vec<catalog::Binding> bindings(ref Input input){return ref input.bindings;}
pub fn Origin origin(ref Input input,usize at){assert(at<count(input));return *v::at<Origin>(ref input.origins,at);}
pub fn ref u8[] instance_id(ref Input input,usize at){let p=origin(input,at);return bytes::view(ref input.names)[p.identity_start..p.identity_end];}
pub fn ref u8[] file(ref Input input,usize at){let p=origin(input,at);return bytes::view(ref input.names)[p.path_start..p.path_end];}
pub fn Result<catalog::Catalog,catalog::Problem> declarations(ref Input input){return catalog::build_scoped(ref input.sources,ref input.owners,ref input.bindings);}
// An edit overlay keeps the exact module/import headers. Import declaration IDs
// can change when private closure helpers are lifted, so remap those IDs rather
// than reusing stale coordinates. No source or metadata owner is mutated.
fn usize header(ref syntax::Parsed p,usize at){var usize cursor=at;while(cursor<syntax::declaration_count(p)){let d=syntax::declaration_at(p,cursor);if(d.kind==1 || d.kind==2){break;}cursor+=1;}return cursor;}
fn Result<v::Vec<usize>,u32> overlay_headers(ref syntax::Parsed before,ref syntax::Parsed after,ref mut usize meter){
    var v::Vec<usize> mapping=v::empty<usize>();var usize i=0;
    while(i<syntax::declaration_count(before)){if(!work::charge(meter,1)){return .err(1);}if(!v::push<usize>(ref mut mapping,0).is_ok){return .err(2);}i+=1;}
    var usize a=header(before,0);var usize b=header(after,0);
    while(a<syntax::declaration_count(before) && b<syntax::declaration_count(after)){
        let old=syntax::declaration_at(before,a);let changed=syntax::declaration_at(after,b);
        if(!work::charge(meter,old.end-old.start+changed.end-changed.start+1)){return .err(1);}
        if(old.kind!=changed.kind || !bytes::equal(syntax::source_view(before)[old.start..old.end],syntax::source_view(after)[changed.start..changed.end])){return .err(4);}
        {let slot=v::at_mut<usize>(ref mut mapping,a);*slot=b+1;}
        a=header(before,a+1);b=header(after,b+1);
    }
    if(a!=syntax::declaration_count(before) || b!=syntax::declaration_count(after)){return .err(4);}
    return .ok(mapping);
}
pub fn Result<Input,Problem> with_source(ref Input original,usize selected,ref syntax::Parsed replacement,ref mut usize meter){
    if(selected>=count(original) || syntax::problem_count(replacement)!=0){return .err(fail(4,0,0,selected,0));}
    let mapping=switch(overlay_headers(source(original,selected),replacement,meter)){case .ok(p):{p}case .err(code):{return .err(fail(code,0,0,selected,0));}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var v::Vec<Origin> origins=v::empty<Origin>();var v::Vec<catalog::Binding> bindings=v::empty<catalog::Binding>();var usize at=0;var usize total=0;
    while(at<count(original)){
        let parsed=if(at==selected){replacement}else{source(original,at)};let size=syntax::source_view(parsed).len;
        if(size>limits::source_set_bytes()-total){return .err(fail(1,0,0,at,0));}total+=size;
        var usize copied_work=0;let copy=switch(syntax::copy(parsed,ref mut copied_work)){case .ok(p):{p}case .err(code):{return .err(fail(3,code,0,at,0));}};
        if(!work::charge(meter,copied_work+size+1)){return .err(fail(1,0,0,at,0));}
        if(!v::push<syntax::Parsed>(ref mut sources,copy).is_ok || !v::push<usize>(ref mut owners,*v::at<usize>(ref original.owners,at)).is_ok || !v::push<Origin>(ref mut origins,origin(original,at)).is_ok){return .err(fail(2,0,0,at,0));}at+=1;
    }
    at=0;while(at<v::len<catalog::Binding>(ref original.bindings)){
        if(!work::charge(meter,1)){return .err(fail(1,0,0,selected,0));}
        var binding=*v::at<catalog::Binding>(ref original.bindings,at);
        if(binding.source==selected){
            if(binding.declaration==0 || binding.declaration>v::len<usize>(ref mapping)){return .err(fail(4,0,0,selected,binding.declaration));}
            let mapped=*v::at<usize>(ref mapping,binding.declaration-1);
            if(mapped==0){return .err(fail(4,0,0,selected,binding.declaration));}binding.declaration=mapped;
        }
        if(!v::push<catalog::Binding>(ref mut bindings,binding).is_ok){return .err(fail(2,0,0,selected,0));}at+=1;
    }
    if(!work::charge(meter,bytes::len(ref original.names)+1)){return .err(fail(1,0,0,selected,0));}
    let names=switch(bytes::copy(bytes::view(ref original.names))){case .ok(p):{p}case .err(_):{return .err(fail(2,0,0,selected,0));}};
    return .ok(Input{.sources,.owners,.origins,.bindings,.names});
}
fn void put(ref mut v::Vec<usize> values,usize at,usize value){let slot=v::at_mut<usize>(values,at);*slot=value;}
pub fn Result<Input,Problem> project(ref visibility::Resolved resolved,ref mut usize meter){return project_mode(resolved,false,meter);}
pub fn Result<Input,Problem> project_tests(ref visibility::Resolved resolved,ref mut usize meter){return project_mode(resolved,true,meter);}
fn Result<Input,Problem> project_mode(ref visibility::Resolved resolved,bool tests,ref mut usize meter){
    let workspace=visibility::workspace_graph(resolved);let nodes=graph::node_count(workspace);
    if(nodes==0 || nodes>limits::source_files() || !work::charge(meter,nodes+1)){return .err(fail(1,0,0,0,0));}
    var v::Vec<usize> offsets=v::empty<usize>();var v::Vec<usize> positions=v::empty<usize>();var usize total=0;var usize owner=0;
    while(owner<nodes){let count=graph::module_count(workspace,owner);if(count>limits::source_files()-total || !work::charge(meter,count+1)){return .err(fail(1,0,owner,0,0));}
        if(!v::push<usize>(ref mut offsets,total).is_ok){return .err(fail(2,0,owner,0,0));}var usize at=0;while(at<count){if(!v::push<usize>(ref mut positions,0).is_ok){return .err(fail(2,0,owner,at,0));}at+=1;}total+=count;owner+=1;
    }
    if(total==0){return .err(fail(1,0,0,0,0));}
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var v::Vec<Origin> origins=v::empty<Origin>();var v::Vec<catalog::Binding> bindings=v::empty<catalog::Binding>();var bytes::Buffer names=bytes::empty();var usize source_bytes=0;owner=0;
    while(owner<nodes){var usize ordinal=0;while(ordinal<graph::module_count(workspace,owner)){let module_index=graph::module_order(workspace,owner,ordinal);let offset=*v::at<usize>(ref offsets,owner);
        if(module_index>=graph::module_count(workspace,owner) || *v::at<usize>(ref positions,offset+module_index)!=0){return .err(fail(4,0,owner,module_index,0));}
        let parsed=graph::module_parser(workspace,owner,module_index);let size=syntax::source_view(parsed).len;
        if(size>limits::source_bytes() || size>limits::source_set_bytes()-source_bytes){return .err(fail(1,0,owner,module_index,0));}source_bytes+=size;
        var usize copied_work=0;let copy=switch(if(tests && graph::module_is_test(workspace,owner,module_index)){syntax::copy_tests(parsed,ref mut copied_work)}else{syntax::copy(parsed,ref mut copied_work)}){case .ok(value):{value}case .err(code):{return .err(fail(3,code,owner,module_index,0));}};
        let id=graph::node_id(workspace,owner);let path=graph::module_file(workspace,owner,module_index);
        if(!work::charge(meter,copied_work+size+id.len+path.len+1)){return .err(fail(1,0,owner,module_index,0));}
        let identity_start=bytes::len(ref names);if(!bytes::append(ref mut names,id).is_ok){return .err(fail(2,0,owner,module_index,0));}let identity_end=bytes::len(ref names);let path_start=identity_end;if(!bytes::append(ref mut names,path).is_ok){return .err(fail(2,0,owner,module_index,0));}
        put(ref mut positions,offset+module_index,v::len<syntax::Parsed>(ref sources)+1);
        if(!v::push<syntax::Parsed>(ref mut sources,copy).is_ok || !v::push<usize>(ref mut owners,owner).is_ok || !v::push<Origin>(ref mut origins,Origin{.owner,.module_index,.identity_start,.identity_end,.path_start,.path_end=bytes::len(ref names)}).is_ok){return .err(fail(2,0,owner,module_index,0));}ordinal+=1;
    }owner+=1;}
    var usize at=0;while(at<visibility::lookup_count(resolved)){let l=visibility::lookup(resolved,at);if(!work::charge(meter,1)){return .err(fail(1,0,l.owner,l.source_module,l.declaration));}
        if(l.owner>=nodes || l.provider>=nodes || l.source_module>=graph::module_count(workspace,l.owner) || l.provider_module>=graph::module_count(workspace,l.provider)){return .err(fail(4,0,l.owner,l.source_module,l.declaration));}
        let source=*v::at<usize>(ref positions,*v::at<usize>(ref offsets,l.owner)+l.source_module);let target=*v::at<usize>(ref positions,*v::at<usize>(ref offsets,l.provider)+l.provider_module);
        if(source==0 || target==0){return .err(fail(4,0,l.owner,l.source_module,l.declaration));}
        let parsed=v::at<syntax::Parsed>(ref sources,source-1);if(l.declaration>=syntax::declaration_count(parsed)){return .err(fail(4,0,l.owner,l.source_module,l.declaration));}let d=syntax::declaration_at(parsed,l.declaration);
        if(d.kind!=2 || l.start<d.start || l.end>d.end || !v::push<catalog::Binding>(ref mut bindings,catalog::Binding{.source=source-1,.declaration=l.declaration+1,.target=target-1}).is_ok){return .err(fail(4,0,l.owner,l.source_module,l.declaration));}at+=1;
    }
    return .ok(Input{.sources,.owners,.origins,.bindings,.names});
}
