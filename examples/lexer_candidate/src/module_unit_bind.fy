module module_unit_bind;
import module_bind as binding;
import module_ast_join as join;
import module_signatures as signatures;
import module_catalog as catalog;
import syntax as syntax;
import raw_view_intrinsics as raw;
import std::bytes as bytes;
import std::vec as v;

pub struct Coordinate {u32 kind;usize module_index;usize declaration;usize start;usize end;}
pub resource Bound {syntax::Parsed source;v::Vec<Coordinate> definitions;v::Vec<Coordinate> targets;v::Vec<usize> type_targets;v::Vec<usize> value_targets;usize first;usize stop;usize work;cleanup{}}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576 - *work){return false;};*work=*work+amount;return true;}
fn Coordinate coordinate(catalog::Declaration d){return Coordinate{.kind=d.kind,.module_index=d.module_index,.declaration=d.declaration,.start=d.source_start,.end=d.source_end};}
fn Result<usize,u32> target(ref signatures::Program metadata,ref mut v::Vec<Coordinate> targets,ref mut v::Vec<usize> cache,usize id){
    if(id==0 || id>=v::len<usize>(cache)){return .err(9);}let usize known=*v::at<usize>(cache,id);if(known!=0){return .ok(known);}
    if(!v::push<Coordinate>(targets,coordinate(signatures::declaration_at(metadata,id))).is_ok){return .err(11);}let usize added=v::len<Coordinate>(targets);let ref mut usize slot=v::at_mut<usize>(cache,id);*slot=added;return .ok(added);
}
fn bool parameter(ref syntax::Parsed p,usize owner,usize root){
    let d=syntax::declaration_at(p,owner);let n=syntax::type_at(p,root);
    if((d.kind!=3 && d.kind!=6) || d.generic_parameters==0 || n.start<d.start || n.end>d.end){return false;}var usize i=0;
    while(i<syntax::generic_parameter_count(p,owner)){let param=syntax::generic_parameter_at(p,owner,i);if(bytes::equal(syntax::source_view(p)[param.start..param.end],syntax::type_text(p,root))){return true;}i+=1;}
    return false;
}
fn join::Problem location(ref syntax::Parsed p,usize first,u32 code,usize start,usize end){
    var usize module_index=first;var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==1){if(start>=d.start && start<=d.end){return fail(code,module_index,start-d.start,if(end>d.end){d.end-d.start}else{end-d.start});}module_index+=1;}i+=1;}return fail(code,first,0,0);
}
pub fn ref syntax::Parsed parsed(ref Bound p){return ref p.source;}
pub fn usize first(ref Bound p){return p.first;}
pub fn usize stop(ref Bound p){return p.stop;}
pub fn usize work(ref Bound p){return p.work;}
pub fn usize target_count(ref Bound p){return v::len<Coordinate>(ref p.targets);}
pub fn Coordinate target_at(ref Bound p,usize id){assert(id>0 && id<=target_count(p));return *v::at<Coordinate>(ref p.targets,id-1);}
pub fn Coordinate definition_at(ref Bound p,usize id){assert(id>0 && id<=v::len<Coordinate>(ref p.definitions));return *v::at<Coordinate>(ref p.definitions,id-1);}
pub fn usize type_target(ref Bound p,usize id){assert(id>0 && id<=v::len<usize>(ref p.type_targets));return *v::at<usize>(ref p.type_targets,id-1);}
pub fn usize value_target(ref Bound p,usize id){assert(id>0 && id<=v::len<usize>(ref p.value_targets));return *v::at<usize>(ref p.value_targets,id-1);}

pub fn Result<Bound,join::Problem> bind(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){
    let usize files=v::len<syntax::Parsed>(sources);if(files!=signatures::module_count(metadata)){return .err(fail(12,0,0,0));}
    if(first>=stop || stop>files){return .err(fail(9,0,0,0));}var usize work=0;var usize module_index=0;
    while(module_index<files){if(!charge(ref mut work,1)){return .err(fail(10,module_index,0,0));}if(!signatures::source_matches(metadata,module_index,v::at<syntax::Parsed>(sources,module_index))){return .err(fail(12,module_index,0,0));}module_index+=1;}
    let p=switch(join::join_range(sources,first,stop)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<Coordinate> definitions=v::empty<Coordinate>();var v::Vec<Coordinate> targets=v::empty<Coordinate>();var v::Vec<usize> cache=v::empty<usize>();var v::Vec<usize> type_targets=v::empty<usize>();var v::Vec<usize> value_targets=v::empty<usize>();
    var usize i=0;while(i<=signatures::declaration_count(metadata)){if(!charge(ref mut work,1)){return .err(fail(10,first,0,0));}if(!v::push<usize>(ref mut cache,0).is_ok){return .err(fail(11,first,0,0));}i+=1;}
    module_index=first;while(module_index<stop){let file=v::at<syntax::Parsed>(sources,module_index);i=0;while(i<syntax::declaration_count(file)){if(!charge(ref mut work,1)){return .err(fail(10,module_index,0,0));}let d=syntax::declaration_at(file,i);if(!v::push<Coordinate>(ref mut definitions,Coordinate{.kind=d.kind,.module_index,.declaration=i+1,.start=d.start,.end=d.end}).is_ok){return .err(fail(11,module_index,d.start,d.end));}i+=1;}module_index+=1;}
    var usize owner=0;i=1;while(i<=syntax::type_count(ref p)){let n=syntax::type_at(ref p,i);if(!charge(ref mut work,1)){return .err(location(ref p,first,10,n.start,n.end));}var usize bound=0;
        while(owner+1<syntax::declaration_count(ref p) && (syntax::declaration_at(ref p,owner).kind==1 || syntax::declaration_at(ref p,owner).kind==2 || n.start>=syntax::declaration_at(ref p,owner).end)){owner+=1;}
        if(n.kind==1 && !binding::builtin(syntax::type_text(ref p,i)) && !parameter(ref p,owner,i)){
            let usize module_index=first+syntax::module_index(ref p,n.start);let found=signatures::declaration_lookup_measured(metadata,module_index,syntax::type_text(ref p,i),ref mut work);if(work>1048576){return .err(location(ref p,first,10,n.start,n.end));}if(found.code!=0){return .err(location(ref p,first,found.code,n.start,n.end));}let d=signatures::declaration_at(metadata,found.item);if(d.kind!=4 && d.kind!=6 && d.kind!=11){return .err(location(ref p,first,9,n.start,n.end));}
            bound=switch(target(metadata,ref mut targets,ref mut cache,found.item)){case .ok(id):{id}case .err(code):{return .err(location(ref p,first,code,n.start,n.end));}};
        }if(!v::push<usize>(ref mut type_targets,bound).is_ok){return .err(location(ref p,first,11,n.start,n.end));}i+=1;
    }
    i=1;while(i<=syntax::expression_count(ref p)){let n=syntax::expression_at(ref p,i);if(!charge(ref mut work,1)){return .err(location(ref p,first,10,n.start,n.end));}var usize bound=0;
        if(n.kind==2 && raw::named(syntax::expression_text(ref p,i))==0 && binding::qualified(syntax::expression_text(ref p,i))){
            let usize module_index=first+syntax::module_index(ref p,n.start);let found=signatures::declaration_lookup_measured(metadata,module_index,syntax::expression_text(ref p,i),ref mut work);if(work>1048576){return .err(location(ref p,first,10,n.start,n.end));}if(found.code!=0){return .err(location(ref p,first,found.code,n.start,n.end));}
            bound=switch(target(metadata,ref mut targets,ref mut cache,found.item)){case .ok(id):{id}case .err(code):{return .err(location(ref p,first,code,n.start,n.end));}};
        }if(!v::push<usize>(ref mut value_targets,bound).is_ok){return .err(location(ref p,first,11,n.start,n.end));}i+=1;
    }
    return .ok(Bound{.source=p,.definitions,.targets,.type_targets,.value_targets,.first,.stop,.work});
}
