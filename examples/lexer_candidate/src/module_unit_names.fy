module module_unit_names;
import module_unit_name_model as m;
import module_unit_name_walk as walk;
import module_unit_body_ast as ast;
import module_unit_bodies as bodies;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_signatures as signatures;
import module_types as types;
import module_ast_join as join;
import syntax as syntax;
import scanner as scanner;
import sdk_authority as sdk;
import std::bytes as bytes;
import std::vec as v;

pub resource Plan {v::Vec<m::Resolved> functions;v::Vec<m::Resolved> cleanups;usize symbols;usize references;usize work;cleanup{}}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
fn Result<v::Vec<usize>,join::Problem> annotations(ref bodies::Plan p,usize id,bool cleaning){var v::Vec<usize> result=v::empty<usize>();let a=if(cleaning){bodies::cleanup_ast(bodies::cleanup_at(p,id))}else{bodies::body_ast(bodies::at(p,id))};var usize i=1;while(i<=ast::type_count(a)){let usize t=if(cleaning){bodies::cleanup_annotation_at(bodies::cleanup_at(p,id),i)}else{bodies::annotation_at(bodies::at(p,id),i)};if(!v::push<usize>(ref mut result,t).is_ok){return .err(fail(11,0,0,0));}i+=1;}return .ok(result);}
pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref interfaces::Interfaces imported,ref bodies::Plan owned){
    let specialized=interfaces::specializations(imported);if(bodies::count(owned)!=generics::count(specialized) || bodies::cleanup_count(owned)!=generics::cleanup_count(specialized) || bodies::type_count(owned)!=interfaces::type_count(imported)){return .err(fail(12,0,0,0));}var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;var usize work=0;
    while(i<=bodies::type_count(owned)){if(!charge(ref mut work,1)){return .err(fail(10,0,0,0));}let a=bodies::type_at(owned,i);let b=interfaces::type_at(imported,i);if(a.kind!=b.kind || a.scalar!=b.scalar || a.module_index!=b.module_index || a.declaration!=b.declaration || a.first!=b.first || a.second!=b.second){return .err(fail(12,0,0,0));}if(!v::push<types::Type>(ref mut graph,a).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    let valid=generics::validate(specialized,sources,metadata,ref graph);if(valid.code!=0){return .err(valid);}var v::Vec<m::Resolved> functions=v::empty<m::Resolved>();var v::Vec<m::Resolved> cleanups=v::empty<m::Resolved>();var usize symbols=0;var usize references=0;var u32 kind=1;
    while(kind<=2){i=1;let usize count=if(kind==1){bodies::count(owned)}else{bodies::cleanup_count(owned)};while(i<=count){var usize module_index=0;var usize declaration=0;var usize receiver_type=0;var bool unsafe_function=false;
        if(kind==1){let b=bodies::at(owned,i);let s=bodies::signature(b);let expected=generics::at(specialized,i);if(s.module_index!=expected.module_index || s.declaration!=expected.declaration || s.argument_count!=expected.argument_count || s.result!=expected.result || s.parameter_count!=expected.parameter_count || s.effects!=expected.effects || s.must_use!=expected.must_use || s.unsafe_call!=expected.unsafe_call){return .err(fail(12,s.module_index,0,0));}var usize j=0;while(j<s.argument_count){if(bodies::argument_at(b,j)!=generics::argument_at(specialized,expected.argument_start+j)){return .err(fail(12,s.module_index,0,0));}j+=1;}module_index=s.module_index;declaration=s.declaration;unsafe_function=s.unsafe_call;}
        else{let s=bodies::cleanup_info(bodies::cleanup_at(owned,i));let expected=generics::cleanup_at(specialized,i);if(s.type!=expected.type || s.module_index!=expected.module_index || s.declaration!=expected.declaration || s.argument_count!=expected.argument_count || s.effects!=expected.effects || s.abilities!=expected.abilities || s.receiver_kind!=10 || s.start!=expected.start || s.end!=expected.end){return .err(fail(12,s.module_index,s.start,s.end));}var usize j=0;while(j<s.argument_count){if(bodies::cleanup_argument_at(bodies::cleanup_at(owned,i),j)!=generics::argument_at(specialized,expected.argument_start+j)){return .err(fail(12,s.module_index,s.start,s.end));}j+=1;}module_index=s.module_index;declaration=s.declaration;receiver_type=s.type;}
        let source=if(kind==1){bodies::source_at(owned,i)}else{bodies::cleanup_source_at(owned,i)};if(!bytes::equal(scanner::source_view(source),syntax::source_view(v::at<syntax::Parsed>(sources,module_index)))){return .err(fail(12,module_index,0,0));}let a=if(kind==1){bodies::body_ast(bodies::at(owned,i))}else{bodies::cleanup_ast(bodies::cleanup_at(owned,i))};let roots=switch(annotations(owned,i,kind==2)){case .ok(p):{p}case .err(p):{return .err(p);}};
        let frame=m::Frame{.kind,.id=i,.module_index,.declaration,.receiver_type,.unsafe_function,.sdk_vector=sdk::authentic(b"std::vec",scanner::source_view(source))};let result=switch(walk::resolve(a,source,metadata,frame,ref roots)){case .ok(p):{p}case .err(p):{return .err(p);}};
        if(!charge(ref mut work,m::work(ref result)+v::len<usize>(ref roots)) || symbols>16384-m::local_count(ref result) || references>16384-m::reference_count(ref result)){return .err(fail(10,module_index,ast::declaration(a).start,ast::declaration(a).end));}symbols+=m::local_count(ref result);references+=m::reference_count(ref result);
        if(kind==1){if(!v::push<m::Resolved>(ref mut functions,result).is_ok){return .err(fail(11,module_index,0,0));}}else{if(!v::push<m::Resolved>(ref mut cleanups,result).is_ok){return .err(fail(11,module_index,0,0));}}i+=1;
    }kind+=1;}return .ok(Plan{.functions,.cleanups,.symbols,.references,.work});
}
pub fn usize function_count(ref Plan p){return v::len<m::Resolved>(ref p.functions);}
pub fn usize cleanup_count(ref Plan p){return v::len<m::Resolved>(ref p.cleanups);}
pub fn ref m::Resolved function_at(ref Plan p,usize id){return v::at<m::Resolved>(ref p.functions,id-1);}
pub fn ref m::Resolved cleanup_at(ref Plan p,usize id){return v::at<m::Resolved>(ref p.cleanups,id-1);}
pub fn usize symbol_count(ref Plan p){return p.symbols;}
pub fn usize reference_count(ref Plan p){return p.references;}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize local_count(ref m::Resolved p){return m::local_count(p);}
pub fn m::Symbol symbol_at(ref m::Resolved p,usize id){return m::symbol_at(p,id);}
pub fn usize use_count(ref m::Resolved p){return m::use_count(p);}
pub fn usize use_at(ref m::Resolved p,usize expression){return m::use_at(p,expression);}
pub fn usize references(ref m::Resolved p){return m::reference_count(p);}
pub fn m::Reference reference_at(ref m::Resolved p,usize id){return m::reference_at(p,id);}
pub fn usize entry_symbol(ref m::Resolved p,usize entry){return m::entry_symbol(p,entry);}
pub fn usize pattern_symbol(ref m::Resolved p,usize expression){return m::pattern_symbol(p,expression);}
