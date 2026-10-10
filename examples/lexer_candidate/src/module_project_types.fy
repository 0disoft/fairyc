module module_project_types;
import module_types as types;
import module_signatures as signatures;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as ir_proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_cleanup_mir_types as identity;
import module_unit_value_model as budget;
import std::vec as v;

pub struct Unit {usize start;usize count;}
// Maps use each input's canonical IDs. Nominal module/declaration coordinates
// remain global; only descriptor edges are rebound into the project graph.
pub resource Plan {types::Graph graph;v::Vec<Unit> units;v::Vec<usize> maps;usize work;cleanup{}}
fn clean::Problem fail(u32 code){return clean::Problem{.code,.module_index=0,.start=0,.end=0};}
fn u32 translated(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
pub fn Result<types::Graph,clean::Problem> from_ir(ref signatures::Program metadata,ref ir::Program unit){
    let checked=ir_proof::check(unit);if(checked.code!=0){return .err(checked);}
    var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;
    while(i<=clean::type_count(ir::cleanup_ir(unit))){if(!v::push<types::Type>(ref mut graph,clean::type_at(ir::cleanup_ir(unit),i)).is_ok){return .err(fail(1003));}i+=1;}
    return switch(signatures::own_types(metadata,graph)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(translated(code)))}};
}
fn Result<usize,clean::Problem> intern(ref mut v::Vec<types::Type> graph,ref mut v::Vec<usize> table,types::Type t,ref mut usize work){
    let usize count=types::count(graph);var usize bucket=identity::bucket(t);
    while(true){if(!budget::charge(work,1)){return .err(fail(1002));}let id=*v::at<usize>(table,bucket);
        if(id==0){if(count>=4096){return .err(fail(1002));}if(!v::push<types::Type>(graph,t).is_ok){return .err(fail(1003));}let slot=v::at_mut<usize>(table,bucket);*slot=count+1;return .ok(count+1);}
        if(identity::same(types::at(graph,id),t)){return .ok(id);}bucket=(bucket+1)&8191;
    }
    return .err(fail(1001));
}
pub fn Result<Plan,clean::Problem> build(ref signatures::Program metadata,ref v::Vec<types::Graph> inputs){
    let usize count=v::len<types::Graph>(inputs);if(count==0 || count>256){return .err(fail(if(count>256){1002}else{1001}));}
    var usize work=0;var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<Unit> units=v::empty<Unit>();var v::Vec<usize> maps=v::empty<usize>();var v::Vec<usize> table=v::empty<usize>();var usize i=0;
    while(i<8192){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}if(!v::push<usize>(ref mut table,0).is_ok){return .err(fail(1003));}i+=1;}
    var usize unit=0;while(unit<count){let input=v::at<types::Graph>(inputs,unit);let usize nt=types::graph_count(input);if(nt<signatures::type_count(metadata) || nt>4096){return .err(fail(if(nt>4096){1002}else{1001}));}
        var v::Vec<types::Type> copy=v::empty<types::Type>();i=1;while(i<=nt){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}let t=types::graph_at(input,i);if(i<=signatures::type_count(metadata) && !identity::same(t,signatures::type_at(metadata,i))){return .err(fail(1001));}if(!v::push<types::Type>(ref mut copy,t).is_ok){return .err(fail(1003));}i+=1;}
        // Revalidate input canonicality and authentic nominal/SDK coordinates
        // against the shared catalog before using any edge as a map index.
        let validated=switch(signatures::own_types(metadata,copy)){case .ok(p):{p}case .err(code):{return .err(fail(translated(code)));}};
        let usize start=v::len<usize>(ref maps);i=1;while(i<=nt){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}var t=types::graph_at(ref validated,i);
            if(t.first!=0){t.first=*v::at<usize>(ref maps,start+t.first-1);}if(t.kind==6){t.second=*v::at<usize>(ref maps,start+t.second-1);}
            // Array second is a length; pointer scalar carries ABI bits.
            let id=switch(intern(ref mut graph,ref mut table,t,ref mut work)){case .ok(id):{id}case .err(p):{return .err(p);}};
            if(!v::push<usize>(ref mut maps,id).is_ok){return .err(fail(1003));}i+=1;
        }if(!v::push<Unit>(ref mut units,Unit{.start,.count=nt}).is_ok){return .err(fail(1003));}unit+=1;
    }
    let owned=switch(signatures::own_types(metadata,graph)){case .ok(p):{p}case .err(code):{return .err(fail(translated(code)));}};
    return .ok(Plan{.graph=owned,.units,.maps,.work});
}
pub fn usize unit_count(ref Plan p){return v::len<Unit>(ref p.units);}
pub fn usize type_count(ref Plan p){return types::graph_count(ref p.graph);}
pub fn types::Type type_at(ref Plan p,usize id){return types::graph_at(ref p.graph,id);}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize mapped(ref Plan p,usize unit,usize id){if(id==0 || unit>=unit_count(p)){return 0;}let u=*v::at<Unit>(ref p.units,unit);return if(id>u.count){0}else{*v::at<usize>(ref p.maps,u.start+id-1)};}
pub fn Result<v::Vec<types::Type>,clean::Problem> copy_types(ref Plan p){var v::Vec<types::Type> result=v::empty<types::Type>();var usize i=1;while(i<=type_count(p)){if(!v::push<types::Type>(ref mut result,type_at(p,i)).is_ok){return .err(fail(1003));}i+=1;}return .ok(result);}
