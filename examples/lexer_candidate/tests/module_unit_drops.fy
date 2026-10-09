module tests::module_unit_drops;
import module_unit_drops as drops;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn interfaces::Interfaces imported(ref v::Vec<syntax::Parsed> sources){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return imported(sources);}};let bound=switch(binding::bind(sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return imported(sources);}};return switch(interfaces::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);imported(sources)}};}
fn drops::Plan retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;import provider as q;fn i32 main(){let p::Outer first=p::make();let q::Outer second=q::make();return 0;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;@effects(filesystem)resource Leaf{i32 n;cleanup{discard this.n;}}@effects(console)pub resource Outer{Option<Leaf> first;Result<Leaf,i32> second;cleanup{discard this.first;}}pub fn Outer make(){return Outer{.first=.none,.second=.err(0)};}")).is_ok);let p=imported(ref sources);return switch(drops::build(ref p)){case .ok(p):{p}case .err(_):{assert(false);retained()}};
}
@test fn void nested_fields_tagged_effects_and_original_edges_survive_source_removal(){
    let p=retained();var usize outer=0;var usize leaf=0;var usize tagged=0;var usize i=1;while(i<=drops::count(ref p)){let n=drops::at(ref p,i);if(n.complete && n.needed){assert(n.droppable);if(n.own_effects==1){outer=i;assert(n.effects==3 && n.context!=0 && n.edge_count==2);}else if(n.own_effects==2){leaf=i;assert(n.effects==2 && n.context!=0);}else{tagged+=1;assert(n.effects==2 && n.context==0);}}i+=1;}assert(outer!=0 && leaf!=0 && outer!=leaf && tagged==2 && drops::work(ref p)<1048576);
    let node=drops::at(ref p,outer);i=0;while(i<node.edge_count){let e=drops::edge_at(ref p,node.edge_start+i);assert(e.owner==outer && e.kind==1 && e.module_index==1 && e.declaration!=0 && e.end>e.start && drops::at(ref p,e.target).effects==2);i+=1;}
}
@test fn void no_drop_propagates_through_values_but_borrows_have_no_obligations(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;@abilities(store)resource Locked{i32 n;cleanup{}}resource Holder{Option<Locked> item;cleanup{}}fn void inspect(ref Locked value){}fn i32 main(){let Holder value=Holder{.item=.none};return 0;}")).is_ok);let imported=imported(ref sources);let p=switch(drops::build(ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var usize denied=0;var usize borrows=0;var usize i=1;while(i<=drops::count(ref p)){let n=drops::at(ref p,i);if(n.complete && !n.droppable){denied+=1;assert(n.needed);}let t=interfaces::type_at(ref imported,i);if(t.kind==7){borrows+=1;assert(n.complete && !n.needed && n.droppable && n.effects==0 && n.edge_count==0);}i+=1;}assert(denied==3 && borrows==1);
}
@effects(filesystem)
@test fn void vec_element_drop_authority_is_inherited_and_aliases_share_context(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let source=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import std::vec as v;import std::vec as q;@effects(filesystem)resource Token{i32 n;cleanup{discard this.n;}}fn usize main(){let v::Vec<Token> value=v::empty<Token>();let q::Vec<Token> second=q::empty<Token>();return v::len<Token>(ref value);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);let imported=imported(ref sources);let p=switch(drops::build(ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var usize vectors=0;var usize i=1;while(i<=drops::count(ref p)){let t=interfaces::type_at(ref imported,i);let n=drops::at(ref p,i);if(t.kind==13 && n.complete){vectors+=1;assert(n.needed && n.droppable && n.own_effects==0 && n.effects==2 && n.context!=0 && n.edge_count==1);let e=drops::edge_at(ref p,n.edge_start);assert(e.owner==i && e.target==t.first && e.kind==5 && e.module_index==1 && drops::at(ref p,e.target).effects==2);}i+=1;}assert(vectors==1);
}
@test fn void unprojected_provider_types_are_rejected_instead_of_claiming_zero_effects(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::value();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;@effects(filesystem)pub resource Hidden{i32 n;cleanup{discard this.n;}}pub fn Hidden hidden(){return Hidden{.n=0};}pub fn i32 value(){return 0;}")).is_ok);let imported=imported(ref sources);let p=switch(drops::build(ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize missing=0;var usize i=1;while(i<=drops::count(ref p)){if(interfaces::type_at(ref imported,i).kind==3){missing+=1;assert(!drops::at(ref p,i).complete);switch(drops::required(ref p,i)){case .ok(_):{assert(false);}case .err(code):{assert(code==9);}};}i+=1;}assert(missing==1);
}
@effects(filesystem)
@test fn void recursive_heap_backed_vec_drop_graph_reaches_a_fixed_point(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let source=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import std::vec as v;@effects(console)resource Node{v::Vec<Node> children;cleanup{discard this.children;}}fn usize main(){let v::Vec<Node> value=v::empty<Node>();return v::len<Node>(ref value);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);let imported=imported(ref sources);let p=switch(drops::build(ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize cyclic=0;var usize i=1;while(i<=drops::count(ref p)){let t=interfaces::type_at(ref imported,i);let n=drops::at(ref p,i);if(n.complete && (t.kind==3 || t.kind==13)){cyclic+=1;assert(n.needed && n.droppable && n.effects==1 && n.edge_count==1);}i+=1;}assert(cyclic==2 && drops::work(ref p)<1048576);
}
