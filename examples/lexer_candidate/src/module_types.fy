module module_types;
import scalar_views as views;
import module_catalog as catalog;
import syntax as syntax;
import type_state as scalars;
import literal_types as literals;
import pointer_types as pointers;
import c_abi_types as abi;
import std::bytes as bytes;
import std::vec as v;

// IDs belong to one owned graph. Composite identity refers to canonical child
// IDs; nominal leaves retain module+declaration, never file-local numeric tags.
// 1 scalar, 2 struct, 3 resource, 4 enum, 5 Option, 6 Result,
// 7 shared borrow, 8 mutable borrow, 9 text, 10 bytes, 11 mutable bytes, 12 pointer.
// 13 authenticated SDK Vec, whose first edge is its exact element identity.
pub struct Type {u32 kind;u32 scalar;usize module_index;usize declaration;usize first;usize second;}
pub struct Problem {u32 code;usize module_index;usize start;usize end;}
pub struct Resolution {usize type;Problem problem;}
pub resource Graph {v::Vec<Type> types;cleanup{}}
fn bool same(Type a,Type b){return a.kind==b.kind && a.scalar==b.scalar && a.module_index==b.module_index && a.declaration==b.declaration && a.first==b.first && a.second==b.second;}
pub fn u32 validate(ref catalog::Catalog c,ref v::Vec<Type> types){
    if(v::len<Type>(types)>4096){return 10;}var usize i=0;while(i<v::len<Type>(types)){let t=*v::at<Type>(types,i);
        var usize previous=0;while(previous<i){if(same(*v::at<Type>(types,previous),t)){return 9;}previous+=1;}
        if(t.kind==13){if(t.scalar!=0 || t.first==0 || t.first>i || t.second!=0 || !catalog::sdk_vector(c,t.module_index,t.declaration) || !element(types,t.first,0)){return 9;}}
        else if(t.kind==2 || t.kind==3 || t.kind==4){let usize item=catalog::identity(c,t.module_index,t.declaration);if(item==0 || t.scalar!=0 || t.first!=0 || t.second!=0){return 9;}let d=catalog::declaration_at(c,item);if(d.kind!=if(t.kind==2){4}else if(t.kind==3){6}else{11} || t.kind==3 && catalog::sdk_vector(c,t.module_index,t.declaration)){return 9;}}
        else{if(t.module_index!=0 || t.declaration!=0){return 9;}
            if(t.kind==1){if(t.scalar==0 || t.scalar>7 && !abi::valid(t.scalar) || t.first!=0 || t.second!=0){return 9;}}
            else if(t.kind==15){if(!views::extended(t.scalar) || t.first!=0 || t.second!=0){return 9;}}
            else if(t.kind==12){if(t.second!=0){return 9;}if(t.first==0){if(!pointers::valid(t.scalar) || pointers::inner(t.scalar)>=256 && pointers::inner(t.scalar)<=767){return 9;}}else if(t.scalar>1 || t.first>i || v::at<Type>(types,t.first-1).kind!=2){return 9;}}
            else if(t.kind>=9 && t.kind<=11 || t.kind==14){if(t.scalar!=0 || t.first!=0 || t.second!=0){return 9;}}
            else if(t.kind>=5 && t.kind<=8){if(t.scalar!=0 || t.first==0 || t.first>i || t.kind==6 && (t.second==0 || t.second>i) || t.kind!=6 && t.second!=0){return 9;}}
            else{return 9;}
        }i+=1;
    }return 0;
}
pub fn Result<Graph,u32> assemble(ref catalog::Catalog c,v::Vec<Type> types){let u32 code=validate(c,ref types);if(code!=0){return .err(code);}return .ok(Graph{.types});}
pub fn usize graph_count(ref Graph graph){return v::len<Type>(ref graph.types);}
pub fn u32 validate_graph(ref catalog::Catalog c,ref Graph graph){return validate(c,ref graph.types);}
pub fn Type graph_at(ref Graph graph,usize id){assert(id>0 && id<=graph_count(graph));return *v::at<Type>(ref graph.types,id-1);}
pub fn usize count(ref v::Vec<Type> graph){return v::len<Type>(graph);}
pub fn Type at(ref v::Vec<Type> graph,usize id){assert(id>0 && id<=count(graph));return *v::at<Type>(graph,id-1);}
fn Resolution fail(u32 code,usize module_index,usize start,usize end){return Resolution{.type=0,.problem=Problem{.code,.module_index,.start,.end}};}
fn Type node(u32 kind,u32 scalar,usize module_index,usize declaration,usize first,usize second){return Type{.kind,.scalar,.module_index,.declaration,.first,.second};}
fn bool element(ref v::Vec<Type> graph,usize id,usize depth){
    if(id==0 || id>count(graph) || depth>=64){return false;}let t=at(graph,id);
    if(t.kind==5 || t.kind==13){return t.first<id && element(graph,t.first,depth+1);}
    if(t.kind==6){return t.first<id && t.second<id && element(graph,t.first,depth+1) && element(graph,t.second,depth+1);}
    return t.kind==1 && t.scalar>=1 && t.scalar<=6 || t.kind>=2 && t.kind<=4 || t.kind==12;
}
fn usize intern(ref mut v::Vec<Type> graph,Type value){
    var usize i=1;while(i<=count(graph)){if(same(at(graph,i),value)){return i;}i+=1;}
    if(count(graph)>=4096 || !v::push<Type>(graph,value).is_ok){return 0;}return count(graph);
}
fn Resolution accepted(ref mut v::Vec<Type> graph,Type value,usize module_index,usize start,usize end){let usize type=intern(graph,value);return if(type==0){fail(if(count(graph)>=4096){10}else{11},module_index,start,end)}else{Resolution{.type,.problem=Problem{.code=0,.module_index,.start=0,.end=0}}};}
fn Resolution walk(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref mut v::Vec<Type> graph,usize depth){
    if(root==0 || root>syntax::type_count(p)){return fail(9,module_index,0,0);}let n=syntax::type_at(p,root);if(depth>=64){return fail(10,module_index,n.start,n.end);}
    if(n.kind==1){let name=syntax::type_text(p,root);let u32 scalar=scalars::primitive(p,root);let u32 c_scalar=abi::resolve(p,root);
        if(scalar!=7 || bytes::equal(name,b"void")){return accepted(graph,node(1,scalar,0,0,0,0),module_index,n.start,n.end);}
        if(abi::valid(c_scalar)){return accepted(graph,node(1,c_scalar,0,0,0,0),module_index,n.start,n.end);}
        let found=catalog::lookup(c,module_index,name);if(found.code!=0){return fail(found.code,module_index,n.start,n.end);}let d=catalog::declaration_at(c,found.item);
        if(d.kind!=4 && d.kind!=6 && d.kind!=11 || catalog::sdk_vector(c,d.module_index,d.declaration)){return fail(9,module_index,n.start,n.end);}
        return accepted(graph,node(if(d.kind==4){2}else if(d.kind==6){3}else{4},0,d.module_index,d.declaration,0,0),module_index,n.start,n.end);
    }
    if(n.kind==2 || n.kind==3){let u32 literal=literals::resolve(p,root);if(views::extended(literal)){return accepted(graph,node(15,literal,0,0,0,0),module_index,n.start,n.end);}if(literal!=0){return accepted(graph,node(if(literal==9){9}else if(literal==10){10}else if(literal==12){14}else{11},0,0,0,0,0),module_index,n.start,n.end);}
        let r=walk(c,p,module_index,n.left,graph,depth+1);if(r.problem.code!=0){return r;}let inner=at(graph,r.type);
        if(inner.kind!=2 && inner.kind!=3 && inner.kind!=13 && !(inner.kind==1 && inner.scalar>=1 && inner.scalar<=6) && !(inner.kind==12 && inner.scalar==1048591)){return fail(9,module_index,n.start,n.end);}
        return accepted(graph,node(if(n.kind==2){7}else{8},0,0,0,r.type,0),module_index,n.start,n.end);
    }
    if(n.kind==11 || n.kind==12){let child=syntax::type_at(p,n.left);if(child.kind==1 && scalars::primitive(p,n.left)==7 && !bytes::equal(syntax::type_text(p,n.left),b"void")){let r=walk(c,p,module_index,n.left,graph,depth+1);if(r.problem.code!=0){return r;}if(at(graph,r.type).kind!=2){return fail(9,module_index,n.start,n.end);}return accepted(graph,node(12,if(n.kind==12){1}else{0},0,0,r.type,0),module_index,n.start,n.end);}let u32 pointer=pointers::resolve(p,root);return if(pointer==0){fail(9,module_index,n.start,n.end)}else{accepted(graph,node(12,pointer,0,0,0,0),module_index,n.start,n.end)};}
    if(n.kind==6){let name=syntax::type_text(p,n.left);let bool option=bytes::equal(name,b"Option");let usize args=syntax::type_argument_count(p,root);
        if(!option && !bytes::equal(name,b"Result")){
            let found=catalog::lookup(c,module_index,name);if(found.code!=0){return fail(if(found.code==5){9}else{found.code},module_index,n.start,n.end);}let d=catalog::declaration_at(c,found.item);
            if(args!=1 || !catalog::sdk_vector(c,d.module_index,d.declaration)){return fail(9,module_index,n.start,n.end);}
            let first=walk(c,p,module_index,syntax::type_argument(p,root,0),graph,depth+1);if(first.problem.code!=0){return first;}
            if(!element(graph,first.type,0)){return fail(9,module_index,n.start,n.end);}
            return accepted(graph,node(13,0,d.module_index,d.declaration,first.type,0),module_index,n.start,n.end);
        }
        if(!option && !bytes::equal(name,b"Result") || args!=if(option){1}else{2}){return fail(9,module_index,n.start,n.end);}
        let first=walk(c,p,module_index,syntax::type_argument(p,root,0),graph,depth+1);if(first.problem.code!=0){return first;}var usize second=0;
        if(!option){let last=walk(c,p,module_index,syntax::type_argument(p,root,1),graph,depth+1);if(last.problem.code!=0){return last;}second=last.type;}
        return accepted(graph,node(if(option){5}else{6},0,0,0,first.type,second),module_index,n.start,n.end);
    }
    return fail(9,module_index,n.start,n.end);
}
pub fn Resolution resolve(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref mut v::Vec<Type> graph){
    if(!catalog::source_matches(c,module_index,p)){return fail(12,module_index,0,syntax::source_view(p).len);}
    // Roll back all speculative interning after a failed resolution. A rejected
    // branch cannot publish a valid prefix or consume the graph's type budget.
    let usize before=count(graph);let result=walk(c,p,module_index,root,graph,0);if(result.problem.code!=0){while(count(graph)>before){discard v::pop<Type>(graph);}}return result;
}
