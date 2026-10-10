module module_units;
import candidate_limits as limits;
import syntax as syntax;
import module_ast_join as join;
import std::vec as v;

pub struct Usage {usize bytes;usize declarations;usize expressions;usize statements;usize entries;usize types;usize fields;usize attributes;usize arguments;}
pub struct Unit {usize first;usize stop;Usage usage;}
pub resource Plan {v::Vec<Unit> units;usize files;usize bytes;cleanup{}}
fn Usage empty(){return Usage{.bytes=0,.declarations=0,.expressions=0,.statements=0,.entries=0,.types=0,.fields=0,.attributes=0,.arguments=0};}
pub fn Usage measured(ref syntax::Parsed p){return Usage{.bytes=syntax::source_view(p).len,.declarations=syntax::declaration_count(p),.expressions=syntax::expression_count(p),.statements=syntax::statement_count(p),.entries=syntax::entry_count(p),.types=syntax::type_count(p),.fields=syntax::field_count(p),.attributes=syntax::attribute_count(p),.arguments=syntax::attribute_argument_count(p)};}
fn bool room(usize current,usize added,usize cap){return current<=cap && added<=cap-current;}
fn bool fits(Usage a,Usage b,usize separator){return room(a.bytes,separator,limits::program_bytes()) && room(a.bytes+separator,b.bytes,limits::program_bytes()) && room(a.declarations,b.declarations,512) && room(a.expressions,b.expressions,limits::program_expressions()) && room(a.statements,b.statements,8192) && room(a.entries,b.entries,1024) && room(a.types,b.types,4096) && room(a.fields,b.fields,1024) && room(a.attributes,b.attributes,4096) && room(a.arguments,b.arguments,4096);}
fn Usage added(Usage a,Usage b,usize separator){return Usage{.bytes=a.bytes+separator+b.bytes,.declarations=a.declarations+b.declarations,.expressions=a.expressions+b.expressions,.statements=a.statements+b.statements,.entries=a.entries+b.entries,.types=a.types+b.types,.fields=a.fields+b.fields,.attributes=a.attributes+b.attributes,.arguments=a.arguments+b.arguments};}
fn join::Problem fail(u32 code,usize module_index,usize end){return join::Problem{.code,.module_index,.start=0,.end};}
pub fn usize count(ref Plan p){return v::len<Unit>(ref p.units);}
pub fn usize file_count(ref Plan p){return p.files;}
pub fn usize source_bytes(ref Plan p){return p.bytes;}
pub fn Unit at(ref Plan p,usize index){assert(index<count(p));return *v::at<Unit>(ref p.units,index);}

pub fn Result<Plan,join::Problem> build(ref v::Vec<syntax::Parsed> sources){
    let usize files=v::len<syntax::Parsed>(sources);if(files==0 || files>limits::source_files()){return .err(fail(10,0,0));}
    var v::Vec<Unit> units=v::empty<Unit>();var Usage usage=empty();var usize total=0;var usize first=0;var usize index=0;
    while(index<files){let p=v::at<syntax::Parsed>(sources,index);let Usage next=measured(p);
        if(syntax::problem_count(p)!=0 || next.declarations==0 || syntax::declaration_at(p,0).kind!=1){return .err(fail(9,index,next.bytes));}
        if(next.bytes>limits::source_bytes() || !fits(empty(),next,0)){return .err(fail(10,index,next.bytes));}
        let usize separator=if(index==0){0}else{1};if(!room(total,separator,limits::source_set_bytes()) || !room(total+separator,next.bytes,limits::source_set_bytes())){return .err(fail(10,index,next.bytes));}total+=separator+next.bytes;
        if(!fits(usage,next,if(index==first){0}else{1})){
            if(!v::push<Unit>(ref mut units,Unit{.first,.stop=index,.usage}).is_ok){return .err(fail(11,index,0));}first=index;usage=empty();
        }
        usage=added(usage,next,if(index==first){0}else{1});index+=1;
    }
    if(!v::push<Unit>(ref mut units,Unit{.first,.stop=files,.usage}).is_ok){return .err(fail(11,files-1,0));}
    return .ok(Plan{.units,.files,.bytes=total});
}

pub fn Result<syntax::Parsed,join::Problem> joined(ref Plan p,ref v::Vec<syntax::Parsed> sources,usize index){
    if(index>=count(p) || v::len<syntax::Parsed>(sources)!=p.files){return .err(fail(9,0,0));}
    let Unit unit=at(p,index);return join::join_range(sources,unit.first,unit.stop);
}
