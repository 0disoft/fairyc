module generic_source_plan;
import generic_source_plan_input as source;
import generic_source_catalog as catalog;
import generic_source_dependency_model as model;
import generic_concrete_types as concrete;
import generic_specialization_identity as encoder;
import generic_package_bindings as bindings;
import module_public_interface_model as records;
import module_public_interface_write as write;
import project_manifest_values as lexical;
import project_path as budget;
import std::bytes as bytes;
import std::vec as v;

pub struct Limits{usize total;usize per_template;usize depth;}
pub fn Limits defaults(){return Limits{.total=4096,.per_template=512,.depth=64};}
pub struct Problem{u32 code;usize root;usize template_index;}
pub struct Node{usize template_index;usize count;usize[8] arguments;records::Text key;records::Text id;usize first;usize dependencies;usize next;u32 state;}
struct Request{usize template_index;usize count;usize[8] arguments;records::Text key;}
struct Frame{usize node;usize next;usize first;usize count;usize names_first;usize[32] children;}
resource Computed{bytes::Buffer names;v::Vec<Node> nodes;v::Vec<usize> edges;v::Vec<usize> roots;v::Vec<usize> order;cleanup{}}
// Keep the exact original input available when bounded planning fails. Neither
// a partial computation nor an unrelated input can be sealed into a Plan.
pub resource Attempt{source::Input input;v::Vec<Computed> values;Problem problem;cleanup{}}
pub resource Plan{Attempt attempt;cleanup{}}
pub fn ref source::Input attempt_input(ref Attempt p){return ref p.input;}
pub fn Problem problem(ref Attempt p){return p.problem;}
fn ref Computed computed(ref Plan p){return v::at<Computed>(ref p.attempt.values,0);}
pub fn Plan seal(Attempt p){assert(p.problem.code==0 && v::len<Computed>(ref p.values)==1);return Plan{.attempt=p};}
pub fn ref source::Input input(ref Plan p){return ref p.attempt.input;}
pub fn usize count(ref Plan p){return v::len<Node>(ref computed(p).nodes);}
pub fn Node at(ref Plan p,usize at){assert(at<count(p));return *v::at<Node>(ref computed(p).nodes,at);}
pub fn ref u8[] text(ref Plan p,records::Text t){assert(t.start<=t.end && t.end<=bytes::len(ref computed(p).names));return bytes::view(ref computed(p).names)[t.start..t.end];}
pub fn usize root_count(ref Plan p){return v::len<usize>(ref computed(p).roots);}
pub fn usize root_at(ref Plan p,usize at){assert(at<root_count(p));return *v::at<usize>(ref computed(p).roots,at);}
pub fn usize ordered_at(ref Plan p,usize at){assert(at<v::len<usize>(ref computed(p).order));return *v::at<usize>(ref computed(p).order,at);}
pub fn usize dependency_at(ref Plan p,usize at){assert(at<v::len<usize>(ref computed(p).edges));return *v::at<usize>(ref computed(p).edges,at);}
pub fn ref u8[] error_code(u32 code){if(code==2003){return b"G2003";}if(code==2005){return b"G2005";}if(code==2006){return b"G2006";}return b"";}
fn Result<records::Text,u32> retain(ref mut bytes::Buffer names,ref u8[] value,ref mut usize work){if(value.len>1048576 || bytes::len(names)>1048576-value.len || !budget::charge(work,value.len+1)){return .err(10);}let start=bytes::len(names);if(!bytes::append(names,value).is_ok){return .err(11);}return .ok(records::Text{.start,.end=bytes::len(names)});}
fn Result<Request,u32> request(ref source::Input input,usize template_index,usize count,usize[8] arguments,ref mut bytes::Buffer names,ref mut usize work){
    let templates=source::templates(input);let types=source::types(input);if(template_index>=catalog::count(templates) || count==0 || count>8 || count!=catalog::template_at(templates,template_index).count){return .err(9);}let row=catalog::template_at(templates,template_index);var bytes::Buffer key=bytes::empty();if(!budget::charge(work,catalog::text(templates,row.id).len+2)){return .err(10);}if(!bytes::append(ref mut key,catalog::text(templates,row.id)).is_ok || !bytes::append_byte(ref mut key,60).is_ok){return .err(11);}var usize i=0;
    while(i<count){if(arguments[i]==0 || arguments[i]>concrete::count(types)){return .err(9);}let t=concrete::at(types,arguments[i]-1);let name=concrete::text(types,t.id);if(!budget::charge(work,name.len+2)){return .err(10);}if(i!=0 && !bytes::append_byte(ref mut key,44).is_ok || !bytes::append(ref mut key,name).is_ok){return .err(11);}i+=1;}if(!bytes::append_byte(ref mut key,62).is_ok){return .err(11);}let saved=switch(retain(names,bytes::view(ref key),work)){case .ok(value):{value}case .err(code):{return .err(code);}};return .ok(Request{.template_index,.count,.arguments,.key=saved});
}
fn Result<i32,u32> compare(ref bytes::Buffer names,records::Text a,records::Text z,ref mut usize work){let data=bytes::view(names);let left=data[a.start..a.end];let right=data[z.start..z.end];if(!budget::charge(work,left.len+right.len+1)){return .err(10);}return .ok(if(bytes::equal(left,right)){0}else if(lexical::before(left,right)){-1}else{1});}
fn Result<v::Vec<Request>,u32> children(ref source::Input input,Node parent,ref mut bytes::Buffer names,ref mut usize work){
    let graph=source::dependencies(input);let range=model::range_at(graph,parent.template_index);var v::Vec<Request> out=v::empty<Request>();var usize i=0;while(i<range.count){if(!budget::charge(work,1)){return .err(10);}let e=model::edge_at(graph,range.first+i);var usize[8] args=[0;8];var usize j=0;while(j<e.count){if(!budget::charge(work,1)){return .err(10);}let a=source::argument_at(input,e.first+j);if(a.parameter!=0){if(a.parameter>parent.count){return .err(9);}args[j]=parent.arguments[a.parameter-1];}else{args[j]=a.concrete;}j+=1;}
        let child=switch(request(input,e.target-1,e.count,args,names,work)){case .ok(value):{value}case .err(code):{return .err(code);}};var usize at=0;var bool duplicate=false;while(at<v::len<Request>(ref out)){let old=*v::at<Request>(ref out,at);let order=switch(compare(names,child.key,old.key,work)){case .ok(value):{value}case .err(code):{return .err(code);}};if(order==0){duplicate=true;break;}if(order<0){break;}at+=1;}
        if(!duplicate){if(!v::push<Request>(ref mut out,child).is_ok){return .err(11);}var usize k=v::len<Request>(ref out)-1;while(k>at){let old=*v::at<Request>(ref out,k-1);{let slot=v::at_mut<Request>(ref mut out,k);*slot=old;}k-=1;}{let slot=v::at_mut<Request>(ref mut out,at);*slot=child;}}i+=1;
    }return .ok(out);
}
// Enter only the next sorted request, matching Go's cycle/budget/ability order.
fn Result<usize,u32> enter(ref source::Input input,Request incoming,Limits limits,ref mut bytes::Buffer names,ref mut bytes::Buffer pending_names,ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> heads,ref mut v::Vec<usize> counts,ref mut v::Vec<Request> pending,ref mut v::Vec<Frame> stack,ref mut usize work){
    if(v::len<Frame>(stack)>=limits.depth){return .err(2006);}var usize id=*v::at<usize>(heads,incoming.template_index);while(id!=0){if(!budget::charge(work,incoming.count+1)){return .err(10);}let old=*v::at<Node>(nodes,id-1);var bool same=old.count==incoming.count;var usize i=0;while(i<incoming.count){if(old.arguments[i]!=incoming.arguments[i]){same=false;}i+=1;}if(same){return if(old.state==2){.ok(id)}else{.err(2005)};}id=old.next;}
    if(v::len<Node>(nodes)>=limits.total || *v::at<usize>(counts,incoming.template_index)>=limits.per_template){return .err(2006);}let templates=source::templates(input);let row=catalog::template_at(templates,incoming.template_index);var usize i=0;while(i<incoming.count){if(!budget::charge(work,1)){return .err(10);}let p=catalog::parameter_at(templates,row.first+i);if(p.scalar && concrete::at(source::types(input),incoming.arguments[i]-1).kind!=1){return .err(2003);}i+=1;}
    let key=switch(retain(names,bytes::view(pending_names)[incoming.key.start..incoming.key.end],work)){case .ok(value):{value}case .err(code):{return .err(code);}};let parent=Node{.template_index=incoming.template_index,.count=incoming.count,.arguments=incoming.arguments,.key,.id=records::Text{.start=0,.end=0},.first=0,.dependencies=0,.next=*v::at<usize>(heads,incoming.template_index),.state=1};if(!v::push<Node>(nodes,parent).is_ok){return .err(11);}id=v::len<Node>(nodes);{let slot=v::at_mut<usize>(heads,incoming.template_index);*slot=id;}{let slot=v::at_mut<usize>(counts,incoming.template_index);*slot+=1;}
    let names_first=bytes::len(pending_names);let selected=switch(children(input,parent,pending_names,work)){case .ok(value):{value}case .err(code):{return .err(code);}};let first=v::len<Request>(pending);i=0;while(i<v::len<Request>(ref selected)){if(!budget::charge(work,1)){return .err(10);}if(v::len<Request>(pending)>=2048){return .err(10);}if(!v::push<Request>(pending,*v::at<Request>(ref selected,i)).is_ok){return .err(11);}i+=1;}
    return if(v::push<Frame>(stack,Frame{.node=id,.next=0,.first,.count=v::len<Request>(ref selected),.names_first,.children=[0;32]}).is_ok){.ok(id)}else{.err(11)};
}
fn Result<records::Text,u32> field(ref mut bytes::Buffer names,ref u8[] value,ref mut usize work){let start=bytes::len(names);let code=write::raw(names,value,work);return if(code!=0){.err(code)}else{.ok(records::Text{.start,.end=bytes::len(names)})};}
fn Result<bytes::Buffer,u32> identity(ref source::Input input,Node node,Frame frame,ref bytes::Buffer names,ref v::Vec<Node> nodes,ref mut usize work){
    let templates=source::templates(input);let row=catalog::template_at(templates,node.template_index);let packages=catalog::packages(templates);var bytes::Buffer facts_names=bytes::empty();var usize measured=0;
    let declaration=switch(field(ref mut facts_names,catalog::text(templates,row.id),ref mut measured)){case .ok(value):{value}case .err(code):{return .err(code);}};let owner=switch(field(ref mut facts_names,bindings::text(packages,bindings::module_at(packages,row.module_index).namespace),ref mut measured)){case .ok(value):{value}case .err(code):{return .err(code);}};let definition=switch(field(ref mut facts_names,catalog::text(templates,row.definition),ref mut measured)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var v::Vec<encoder::Concrete> args=v::empty<encoder::Concrete>();var v::Vec<records::Text> abilities=v::empty<records::Text>();var usize i=0;while(i<node.count){let code=concrete::append_argument(source::types(input),node.arguments[i],ref mut facts_names,ref mut args,ref mut abilities,ref mut measured);if(code!=0){return .err(code);}i+=1;}
    var v::Vec<records::Text> dependencies=v::empty<records::Text>();i=0;while(i<frame.count){let child=*v::at<Node>(nodes,frame.children[i]-1);if(child.state!=2){return .err(9);}let id=switch(field(ref mut facts_names,bytes::view(names)[child.id.start..child.id.end],ref mut measured)){case .ok(value):{value}case .err(code):{return .err(code);}};if(!v::push<records::Text>(ref mut dependencies,id).is_ok){return .err(11);}i+=1;}
    let facts=encoder::assemble(facts_names,encoder::Root{.declaration,.owner,.definition},args,abilities,v::empty<encoder::Constraint>(),v::empty<encoder::Operand>(),dependencies);let result=switch(encoder::compute(ref facts,ref mut measured)){case .ok(value):{value}case .err(code):{discard budget::charge(work,measured);return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}return .ok(result);
}
fn u32 complete(ref source::Input input,ref mut bytes::Buffer names,ref mut bytes::Buffer pending_names,ref mut v::Vec<Node> nodes,ref mut v::Vec<Request> pending,ref mut v::Vec<Frame> stack,ref mut v::Vec<usize> edges,ref mut v::Vec<usize> order,ref mut usize work){
    var Frame frame=*v::at<Frame>(stack,v::len<Frame>(stack)-1);var usize i=1;while(i<frame.count){let item=frame.children[i];let key=v::at<Node>(nodes,item-1).id;var usize at=i;while(at>0){let before=v::at<Node>(nodes,frame.children[at-1]-1).id;let sorted=switch(compare(names,key,before,work)){case .ok(value):{value}case .err(code):{return code;}};if(sorted>=0){break;}let previous=frame.children[at-1];frame.children[at]=previous;at-=1;}frame.children[at]=item;i+=1;}
    let node=*v::at<Node>(nodes,frame.node-1);let result=switch(identity(input,node,frame,names,nodes,work)){case .ok(value):{value}case .err(code):{return code;}};let id=switch(retain(names,bytes::view(ref result),work)){case .ok(value):{value}case .err(code):{return code;}};let first=v::len<usize>(edges);i=0;while(i<frame.count){if(!budget::charge(work,1)){return 10;}if(!v::push<usize>(edges,frame.children[i]).is_ok){return 11;}i+=1;}
    {let slot=v::at_mut<Node>(nodes,frame.node-1);slot.id=id;slot.first=first;slot.dependencies=frame.count;slot.state=2;}if(!v::push<usize>(order,frame.node).is_ok){return 11;}while(v::len<Request>(pending)>frame.first){discard v::pop<Request>(pending);}if(!bytes::truncate(pending_names,frame.names_first).is_ok){return 9;}discard v::pop<Frame>(stack);return 0;
}
fn Result<Computed,Problem> compute(ref source::Input input,Limits limits,ref mut usize work){
    if(limits.total==0 || limits.total>4096 || limits.per_template==0 || limits.per_template>512 || limits.depth==0 || limits.depth>64){return .err(Problem{.code=2006,.root=0,.template_index=0});}if(*work>16777216){return .err(Problem{.code=10,.root=0,.template_index=0});}let templates=source::templates(input);var bytes::Buffer names=bytes::empty();var bytes::Buffer pending_names=bytes::empty();var v::Vec<Node> nodes=v::empty<Node>();var v::Vec<usize> heads=v::empty<usize>();var v::Vec<usize> counts=v::empty<usize>();var v::Vec<Request> pending=v::empty<Request>();var v::Vec<Frame> stack=v::empty<Frame>();var v::Vec<usize> edges=v::empty<usize>();var v::Vec<usize> roots=v::empty<usize>();var v::Vec<usize> order=v::empty<usize>();var usize i=0;
    while(i<catalog::count(templates)){if(!budget::charge(work,1)){return .err(Problem{.code=10,.root=0,.template_index=i});}if(!v::push<usize>(ref mut heads,0).is_ok || !v::push<usize>(ref mut counts,0).is_ok){return .err(Problem{.code=11,.root=0,.template_index=i});}i+=1;}
    let data=source::roots(input);i=0;while(i<model::edge_count(data)){let e=model::edge_at(data,i);var usize[8] arguments=[0;8];var usize j=0;while(j<e.count){arguments[j]=source::root_argument(input,e.first+j);j+=1;}let root=switch(request(input,e.target-1,e.count,arguments,ref mut pending_names,work)){case .ok(value):{value}case .err(code):{return .err(Problem{.code,.root=i+1,.template_index=e.target-1});}};let selected=switch(enter(input,root,limits,ref mut names,ref mut pending_names,ref mut nodes,ref mut heads,ref mut counts,ref mut pending,ref mut stack,work)){case .ok(value):{value}case .err(code):{return .err(Problem{.code,.root=i+1,.template_index=e.target-1});}};if(!v::push<usize>(ref mut roots,selected).is_ok){return .err(Problem{.code=11,.root=i+1,.template_index=e.target-1});}
        while(v::len<Frame>(ref stack)!=0){if(!budget::charge(work,1)){return .err(Problem{.code=10,.root=i+1,.template_index=e.target-1});}let current=*v::at<Frame>(ref stack,v::len<Frame>(ref stack)-1);var u32 code=0;var usize target=e.target-1;
            if(current.next<current.count){let incoming=*v::at<Request>(ref pending,current.first+current.next);target=incoming.template_index;let next=switch(enter(input,incoming,limits,ref mut names,ref mut pending_names,ref mut nodes,ref mut heads,ref mut counts,ref mut pending,ref mut stack,work)){case .ok(value):{value}case .err(error):{return .err(Problem{.code=error,.root=i+1,.template_index=target});}};let parent_at=if(v::at<Frame>(ref stack,v::len<Frame>(ref stack)-1).node==current.node){v::len<Frame>(ref stack)-1}else{v::len<Frame>(ref stack)-2};let parent=v::at_mut<Frame>(ref mut stack,parent_at);parent.children[current.next]=next;parent.next+=1;}
            else{target=v::at<Node>(ref nodes,current.node-1).template_index;code=complete(input,ref mut names,ref mut pending_names,ref mut nodes,ref mut pending,ref mut stack,ref mut edges,ref mut order,work);}if(code!=0){return .err(Problem{.code,.root=i+1,.template_index=target});}
        }discard bytes::clear(ref mut pending_names);i+=1;
    }return .ok(Computed{.names,.nodes,.edges,.roots,.order});
}

pub fn Attempt attempt(source::Input input,Limits limits,ref mut usize work){
    var v::Vec<Computed> values=v::empty<Computed>();switch(compute(ref input,limits,work)){case .ok(p):{if(!v::push<Computed>(ref mut values,p).is_ok){return Attempt{.input,.values,.problem=Problem{.code=11,.root=0,.template_index=0}};}}case .err(e):{return Attempt{.input,.values,.problem=e};}};return Attempt{.input,.values,.problem=Problem{.code=0,.root=0,.template_index=0}};
}
pub fn Result<Plan,Problem> build(source::Input input,Limits limits,ref mut usize work){let result=attempt(input,limits,work);let failed=problem(ref result);if(failed.code!=0){return .err(failed);}return .ok(seal(result));}
