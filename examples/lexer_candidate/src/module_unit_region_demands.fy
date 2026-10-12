module module_unit_region_demands;
import module_unit_generics as generics;
import module_unit_interface_work as meter;
import module_region_types as regions;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Demand{usize error_type;usize arena_type;}
fn Result<bool,join::Problem> active(ref syntax::Parsed source,usize module_index,usize first,usize stop,usize start,usize end,ref generics::Plan plan,ref mut meter::Budget work){
    var usize i=1;while(i<syntax::declaration_count(source)){if(!meter::charge(work,1)){return .err(meter::fail(10,module_index,start,end));}let d=syntax::declaration_at(source,i);
        if((d.kind==3 || d.kind==6) && d.start<=start && end<=d.end){if(d.generic_parameters==0 && module_index>=first && module_index<stop){return .ok(true);}var usize j=1;while(j<=generics::count(plan)){if(!meter::charge(work,1)){return .err(meter::fail(10,module_index,start,end));}let item=generics::at(plan,j);if(item.module_index==module_index && item.declaration==i+1){return .ok(true);}j+=1;}j=1;while(j<=generics::cleanup_count(plan)){if(!meter::charge(work,1)){return .err(meter::fail(10,module_index,start,end));}let item=generics::cleanup_at(plan,j);if(item.module_index==module_index && item.declaration==i+1){return .ok(true);}j+=1;}return .ok(false);}i+=1;
    }return .ok(false);
}
// Region dependencies belong to selected ordinary bodies or actual retained
// function/cleanup contexts. Uninstantiated template source stays unchecked.
pub fn Result<Demand,join::Problem> collect(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop,ref generics::Plan plan,ref mut v::Vec<types::Type> graph,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut meter::Budget work){
    let count=v::len<syntax::Parsed>(sources);if(count>512 || first>=stop || stop>count){return .err(meter::fail(9,first,0,0));}var bool[512] selected=[false;512];var usize i=first;while(i<stop){selected[i]=true;i+=1;}i=1;while(i<=generics::count(plan)){let item=generics::at(plan,i);if(item.module_index>=count){return .err(meter::fail(9,first,0,0));}selected[item.module_index]=true;i+=1;}i=1;while(i<=generics::cleanup_count(plan)){let item=generics::cleanup_at(plan,i);if(item.module_index>=count){return .err(meter::fail(9,first,0,0));}selected[item.module_index]=true;i+=1;}
    var Demand result=Demand{.error_type=0,.arena_type=0};var usize source_index=0;
    while(source_index<count){if(!meter::charge(work,1)){return .err(meter::fail(10,source_index,0,0));}if(!selected[source_index]){source_index+=1;continue;}let source=v::at<syntax::Parsed>(sources,source_index);i=1;
        while(i<=syntax::expression_count(source)){if(!meter::charge(work,1)){return .err(meter::fail(10,source_index,0,0));}let n=syntax::expression_at(source,i);if(n.kind==6){let callee=syntax::expression_at(source,n.left);
                if(callee.kind==8 && bytes::equal(syntax::expression_text(source,n.left),b"place")){let used=switch(active(source,source_index,first,stop,n.start,n.end,plan,work)){case .ok(value):{value}case .err(p):{return .err(p);}};if(used){result.error_type=switch(regions::error_type(sources,metadata,graph,source_index,n.start,n.end,ref mut work.work)){case .ok(t):{t}case .err(p):{return .err(p);}};let code=meter::mark(graph,pending,seen,work,result.error_type);if(code!=0){return .err(meter::fail(code,source_index,n.start,n.end));}}}
            }i+=1;
        }i=1;while(i<=syntax::statement_count(source)){if(!meter::charge(work,1)){return .err(meter::fail(10,source_index,0,0));}let s=syntax::statement_at(source,i);if(s.kind==17){let used=switch(active(source,source_index,first,stop,s.start,s.end,plan,work)){case .ok(value):{value}case .err(p):{return .err(p);}};if(used){let input=syntax::expression_at(source,s.expression);result.arena_type=switch(regions::arena_type(sources,metadata,graph,source_index,input.start,input.end,ref mut work.work)){case .ok(t):{t}case .err(p):{return .err(p);}};let code=meter::mark(graph,pending,seen,work,result.arena_type);if(code!=0){return .err(meter::fail(code,source_index,input.start,input.end));}}}i+=1;}source_index+=1;
    }return .ok(result);
}
