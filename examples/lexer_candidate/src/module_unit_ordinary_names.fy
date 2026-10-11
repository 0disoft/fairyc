module module_unit_ordinary_names;
import module_unit_name_model as m;
import module_unit_name_walk as walk;
import module_unit_body_ast as ast;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import std::vec as v;

// Ordinary ASTs stay in the source owner. Only binding tables and source IDs
// survive the temporary body projection used by the shared lexical walker.
pub resource Function {usize module_index;usize declaration;m::Resolved names;v::Vec<usize> expressions;v::Vec<usize> entries;cleanup{}}
pub resource Plan {v::Vec<Function> functions;usize symbols;usize references;usize work;cleanup{}}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref binding::Bound bound,ref interfaces::Interfaces imported){
    let valid=binding::validate(bound,sources,metadata);if(valid.code!=0){return .err(valid);}
    if(interfaces::first(imported)!=binding::first(bound) || interfaces::stop(imported)!=binding::stop(bound) || interfaces::annotation_count(imported)!=syntax::type_count(binding::parsed(bound))){return .err(fail(12,binding::first(bound),0,0));}
    var usize work=0;var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;
    while(i<=interfaces::type_count(imported)){if(!ast::charge(ref mut work,1)){return .err(fail(10,0,0,0));}if(!v::push<types::Type>(ref mut graph,interfaces::type_at(imported,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    let current=generics::validate(interfaces::specializations(imported),sources,metadata,ref graph);if(current.code!=0){return .err(current);}
    var v::Vec<Function> functions=v::empty<Function>();var usize symbols=0;var usize references=0;var usize type_offset=0;var usize module_index=binding::first(bound);
    while(module_index<binding::stop(bound)){let original=v::at<syntax::Parsed>(sources,module_index);i=0;
        while(i<syntax::declaration_count(original)){let d=syntax::declaration_at(original,i);if(!ast::charge(ref mut work,1)){return .err(fail(10,module_index,d.start,d.end));}
            if(d.kind==3 && d.generic_parameters==0 && d.body!=0){
                let a=switch(ast::build(original,i+1,ref mut work)){case .ok(a):{a}case .err(code):{return .err(fail(code,module_index,d.start,d.end));}};
                var v::Vec<usize> roots=v::empty<usize>();var v::Vec<usize> expressions=v::empty<usize>();var v::Vec<usize> entries=v::empty<usize>();var usize n=1;
                while(n<=ast::type_count(ref a)){if(!ast::charge(ref mut work,1)){return .err(fail(10,module_index,d.start,d.end));}let usize root=type_offset+ast::type_origin(ref a,n);let annotation=interfaces::annotation_at(imported,root);if(annotation.deferred || annotation.type>interfaces::type_count(imported)){return .err(fail(12,module_index,d.start,d.end));}if(!v::push<usize>(ref mut roots,annotation.type).is_ok){return .err(fail(11,module_index,d.start,d.end));}n+=1;}
                n=1;while(n<=ast::expression_count(ref a)){if(!ast::charge(ref mut work,1)){return .err(fail(10,module_index,d.start,d.end));}if(!v::push<usize>(ref mut expressions,ast::expression_origin(ref a,n)).is_ok){return .err(fail(11,module_index,d.start,d.end));}n+=1;}
                n=1;while(n<=ast::entry_count(ref a)){if(!ast::charge(ref mut work,1)){return .err(fail(10,module_index,d.start,d.end));}if(!v::push<usize>(ref mut entries,ast::entry_origin(ref a,n)).is_ok){return .err(fail(11,module_index,d.start,d.end));}n+=1;}
                let frame=m::Frame{.kind=3,.id=v::len<Function>(ref functions)+1,.module_index,.declaration=i+1,.receiver_type=0,.unsafe_function=d.unsafe_end!=0,.sdk_vector=signatures::sdk_vector_source(metadata,module_index)};
                let resolved=switch(walk::resolve(ref a,syntax::source_stream(original),metadata,frame,ref roots)){case .ok(p):{p}case .err(p):{return .err(p);}};
                if(!ast::charge(ref mut work,m::work(ref resolved)) || symbols>16384-m::local_count(ref resolved) || references>16384-m::reference_count(ref resolved) || v::len<Function>(ref functions)>=512){return .err(fail(10,module_index,d.start,d.end));}symbols+=m::local_count(ref resolved);references+=m::reference_count(ref resolved);
                if(!v::push<Function>(ref mut functions,Function{.module_index,.declaration=i+1,.names=resolved,.expressions,.entries}).is_ok){return .err(fail(11,module_index,d.start,d.end));}
            }i+=1;
        }type_offset+=syntax::type_count(original);module_index+=1;
    }if(type_offset!=interfaces::annotation_count(imported)){return .err(fail(12,binding::first(bound),0,0));}return .ok(Plan{.functions,.symbols,.references,.work});
}
pub fn usize count(ref Plan p){return v::len<Function>(ref p.functions);}
pub fn ref Function at(ref Plan p,usize id){return v::at<Function>(ref p.functions,id-1);}
pub fn usize module_index(ref Function p){return p.module_index;}
pub fn usize declaration(ref Function p){return p.declaration;}
pub fn ref m::Resolved resolved(ref Function p){return ref p.names;}
pub fn usize expression_origin(ref Function p,usize id){return *v::at<usize>(ref p.expressions,id-1);}
pub fn usize entry_origin(ref Function p,usize id){return *v::at<usize>(ref p.entries,id-1);}
pub fn usize symbols(ref Plan p){return p.symbols;}
pub fn usize references(ref Plan p){return p.references;}
pub fn usize work(ref Plan p){return p.work;}
