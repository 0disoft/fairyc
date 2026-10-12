module module_unit_interface_annotations;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import module_ast_join as join;
import std::vec as v;
import storage_intrinsics as storage;

import module_unit_interfaces as owner;
import module_unit_interface_work as meter;

fn void request(ref mut v::Vec<owner::Annotation> table,usize root){if(root!=0){let ref mut owner::Annotation slot=v::at_mut<owner::Annotation>(table,root-1);slot.required=true;}}
pub fn Result<v::Vec<owner::Annotation>,join::Problem> annotations(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref mut v::Vec<types::Type> graph,ref mut meter::Budget work){
    let p=binding::parsed(bound);var v::Vec<bool> vectors=v::empty<bool>();var usize source_index=binding::first(bound);while(source_index<binding::stop(bound)){if(!meter::charge(work,1)){return .err(meter::fail(10,source_index,0,0));}if(!v::push<bool>(ref mut vectors,signatures::sdk_vector_source(metadata,source_index)).is_ok){return .err(meter::fail(11,source_index,0,0));}source_index+=1;}var v::Vec<owner::Annotation> table=v::empty<owner::Annotation>();var usize i=0;
    while(i<syntax::type_count(p)){if(!meter::charge(work,1)){return .err(meter::fail(10,binding::first(bound),0,0));}if(!v::push<owner::Annotation>(ref mut table,owner::Annotation{.type=0,.required=false,.deferred=false}).is_ok){return .err(meter::fail(11,binding::first(bound),0,0));}i+=1;}
    i=0;while(i<syntax::declaration_count(p)){request(ref mut table,syntax::declaration_at(p,i).result_type);i+=1;}
    i=0;while(i<syntax::entry_count(p)){request(ref mut table,syntax::entry_at(p,i).type_root);i+=1;}
    i=0;while(i<syntax::field_count(p)){let field=syntax::field_at(p,i);if(field.kind!=1 && field.kind!=2){request(ref mut table,field.type_root);}i+=1;}
    i=1;while(i<=syntax::expression_count(p)){if(!meter::charge(work,1)){return .err(meter::fail(10,binding::first(bound),0,0));}let e=syntax::expression_at(p,i);if(e.kind==23){request(ref mut table,e.right);}else if(e.kind==20){var usize link=e.right;while(link!=0){let n=syntax::type_at(p,link);request(ref mut table,storage::annotation_root(p,i,n.left,*v::at<bool>(ref vectors,syntax::module_index(p,e.start))));link=n.right;}}i+=1;}
    i=1;while(i<=syntax::statement_count(p)){let s=syntax::statement_at(p,i);if(s.kind==18){request(ref mut table,s.second);}i+=1;}
    var usize base=0;var usize module_index=binding::first(bound);
    while(module_index<binding::stop(bound)){let original=v::at<syntax::Parsed>(sources,module_index);var usize owner=0;var v::Vec<usize> queries=v::empty<usize>();i=1;
        while(i<=syntax::type_count(original)){let n=syntax::type_at(original,i);while(owner+1<syntax::declaration_count(original) && n.start>=syntax::declaration_at(original,owner).end){owner+=1;}
            var usize query=0;
            if(v::at<owner::Annotation>(ref table,base+i-1).required){let d=syntax::declaration_at(original,owner);if(d.generic_parameters!=0){let ref mut owner::Annotation slot=v::at_mut<owner::Annotation>(ref mut table,base+i-1);slot.deferred=true;}
                else{query=i;}}
            if(!v::push<usize>(ref mut queries,query).is_ok){return .err(meter::fail(11,module_index,n.start,n.end));}
            i+=1;
        }
        let resolved=switch(signatures::resolve_types(metadata,original,module_index,ref queries,graph)){case .ok(p):{p}case .err(e):{let problem=e.problem;var u32 code=problem.code;if(code==9 && e.request!=0){let n=syntax::type_at(original,e.request);owner=0;while(owner+1<syntax::declaration_count(original) && n.start>=syntax::declaration_at(original,owner).end){owner+=1;}if(syntax::declaration_at(original,owner).kind==4 && n.kind==8){code=2;}}return .err(meter::fail(code,module_index,problem.start,problem.end));}};
        i=0;while(i<v::len<usize>(ref resolved)){let slot=v::at_mut<owner::Annotation>(ref mut table,base+i);slot.type=*v::at<usize>(ref resolved,i);i+=1;}
        base+=syntax::type_count(original);module_index+=1;
    }return .ok(table);
}
