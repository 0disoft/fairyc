module tests::closure_unit_values;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_calls as calls;
import module_signatures as signatures;
import copied_closure_tables as private;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

// This receipt stops at source-backed unit values. It does not certify MIR or
// execute the user's closure body. Go supplies only original source strings.
pub struct Receipt {u32 phase;u32 code;usize module_index;usize start;usize end;usize typed23;usize kind4;usize ordered_arguments;usize helpers;usize captures;usize passes;}
fn Receipt failed(u32 phase,u32 code,usize module_index,usize start,usize end){return Receipt{.phase,.code,.module_index,.start,.end,.typed23=0,.kind4=0,.ordered_arguments=0,.helpers=0,.captures=0,.passes=0};}
// The source compiler does not admit i32-to-u32 casts. Preserve bounded scanner
// and preparation codes using the same scalar operations as closure_value_c.
fn u32 positive(i32 value){if(value<0 || value>65536){return 1;}var i32 left=value;var u32 code=0;while(left!=0){left-=1;code+=1;}return code;}
fn bool leaf(ref values::Plan p,usize type){if(type==0 || type>values::type_count(p)){return false;}let t=values::type_at(p,type);return t.kind==23 && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0;}
fn bool i32_value(ref values::Plan p,usize type){if(type==0 || type>values::type_count(p)){return false;}let t=values::type_at(p,type);return t.kind==1 && t.scalar==1;}
// Values owns its expression vector privately. Inspect only published records.
fn usize expression_index(ref model::Values p,usize source_index){var usize low=0;var usize high=model::expression_count(p);while(low<high){let mid=low+(high-low)/2;if(model::expression_at(p,mid+1).source_index<source_index){low=mid+1;}else{high=mid;}}return if(low<model::expression_count(p) && model::expression_at(p,low+1).source_index==source_index){low+1}else{0};}
fn Receipt inspect(ref v::Vec<syntax::Parsed> sources,ref closure::Ready ready,ref values::Plan p){
    var Receipt receipt=failed(0,0,0,0,0);receipt.passes=closure::passes(ready);let imported=closure::imported(ready);var usize i=0;
    while(i<values::count(p)){let frame=values::at(p,i);let f=model::frame(frame);let source=v::at<syntax::Parsed>(sources,f.module_index);var usize n=1;
        while(n<=model::expression_count(frame)){let e=model::expression_at(frame,n);let original=syntax::expression_at(source,e.source_index);
            if(e.kind==36){if(e.state!=1 || !leaf(p,e.type) || e.target==0 || e.left!=original.left || e.right!=original.right || e.start!=original.start || e.end!=original.end){return failed(8,20,f.module_index,e.start,e.end);}receipt.typed23+=1;
                let target=syntax::expression_at(source,e.left);let encoded=private::value_captures(syntax::private_names(source),e.target);if(target.kind!=2 || target.token!=private::helper_flag()+e.target || encoded==0 || encoded>5){return failed(8,21,f.module_index,e.start,e.end);}let captures=encoded-1;
                var bool found=false;var usize c=0;while(c<interfaces::callable_count(imported)){let helper=interfaces::callable_at(imported,c);if(helper.module_index==f.module_index && helper.declaration==e.target){if(helper.template || helper.foreign || helper.unsafe_call || helper.effects!=0 || !i32_value(p,helper.result) || helper.parameter_count!=captures+1){return failed(8,22,f.module_index,e.start,e.end);}found=true;}c+=1;}
                if(!found){return failed(8,23,f.module_index,e.start,e.end);}receipt.helpers+=1;receipt.captures+=captures;
                if(syntax::call_argument_count(source,e.source_index)!=captures){return failed(8,24,f.module_index,e.start,e.end);}var usize previous=0;c=0;
                while(c<captures){let child=syntax::call_argument(source,e.source_index,c);let operand=syntax::expression_at(source,child);let local=expression_index(frame,child);if(local==0 || operand.kind!=2 || operand.token<=private::snapshot_flag()+previous){return failed(8,25,f.module_index,e.start,e.end);}let value=model::expression_at(frame,local);if(value.state!=1 || !i32_value(p,value.type) || value.binding==0){return failed(8,26,f.module_index,operand.start,operand.end);}previous=operand.token-private::snapshot_flag();c+=1;}
            }n+=1;
        }
        n=0;while(n<model::call_count(frame)){let call=model::call_at(frame,n);let original=syntax::expression_at(source,call.source_index);if(!call.checked || call.argument_count!=syntax::call_argument_count(source,call.source_index)){return failed(8,30,f.module_index,call.start,call.end);}
            if(call.kind==4){let e=model::expression_at(frame,call.expression);let callee=expression_index(frame,e.left);if(callee==0 || e.left!=original.left || e.right!=original.right || !i32_value(p,e.type) || e.target!=0 || call.target_module!=0 || call.target_declaration!=0 || call.instance!=0 || call.argument_count!=1){return failed(8,31,f.module_index,call.start,call.end);}let named=model::expression_at(frame,callee);if(named.kind!=2 || named.binding==0 || named.place.mutable || !leaf(p,named.type)){return failed(8,32,f.module_index,named.start,named.end);}receipt.kind4+=1;}
            var usize a=0;while(a<call.argument_count){let arg=model::argument_at(frame,call.argument_start+a);let written=syntax::call_argument(source,call.source_index,a);let node=syntax::expression_at(source,written);if(!arg.checked || arg.invocation!=n+1 || arg.position!=a || arg.source_index!=written || arg.start!=node.start || arg.end!=node.end || arg.expected==0 || arg.actual!=arg.expected || arg.expression!=expression_index(frame,written)){return failed(8,33,f.module_index,node.start,node.end);}receipt.ordered_arguments+=1;a+=1;}n+=1;
        }i+=1;
    }
    let declared=closure::call_plan(ready);var usize value_calls=0;i=0;while(i<calls::count(declared)){let call=calls::at(declared,i);if(call.callable!=0){let source=v::at<syntax::Parsed>(sources,call.module_index);let e=syntax::expression_at(source,call.source_index);if(call.callable!=e.left || call.signature!=0 || call.result!=0 || call.target_module!=0 || call.target_declaration!=0 || call.parameter_start!=0 || call.parameter_count!=1){return failed(8,34,call.module_index,call.start,call.end);}value_calls+=1;}i+=1;}
    if(value_calls!=receipt.kind4 || values::unresolved(p)!=0 || values::checked_calls(p)!=values::invocation_count(p) || values::checked_arguments(p)!=values::arguments(p)){return failed(8,35,0,0,0);}return receipt;
}
pub fn Receipt checked(ref u8[] text){
    let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{return failed(1,11,0,0,text.len);}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(code):{return failed(2,positive(code),0,0,text.len);}};
    let source=switch(syntax::parse(stream,512)){case .ok(s):{s}case .err(code):{return failed(3,positive(code),0,0,text.len);}};
    if(syntax::problem_count(ref source)!=0){let e=syntax::problem_at(ref source,0);return failed(3,e.code,0,e.start,e.end);}
    var sources=v::empty<syntax::Parsed>();if(!v::push<syntax::Parsed>(ref mut sources,source).is_ok){return failed(1,11,0,0,text.len);}
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(e):{return failed(4,e.code,e.module_index,e.start,e.end);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(e):{return failed(5,e.code,e.module_index,e.start,e.end);}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(r):{r}case .err(e):{return failed(6,e.code,e.module_index,e.start,e.end);}};
    let p=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{return failed(7,e.code,e.module_index,e.start,e.end);}};return inspect(ref sources,ref ready,ref p);
}
@effects(console) fn void failure(ref u8[] name,ref u8[] raw,Receipt r){
    discard io::write_stderr("closure-unit-values case=");discard io::write_stderr_bytes(name);discard io::write_stderr(" phase=");discard io::write_stderr_u32(r.phase);discard io::write_stderr(" code=");discard io::write_stderr_u32(r.code);discard io::write_stderr(" module=");discard io::write_stderr_usize(r.module_index);discard io::write_stderr(" span=");discard io::write_stderr_usize(r.start);discard io::write_stderr("..");discard io::write_stderr_usize(r.end);discard io::write_stderr("\nsource: ");discard io::write_stderr_bytes(raw);discard io::write_stderr("\n");
    discard io::write_stderr("typed23=");discard io::write_stderr_usize(r.typed23);discard io::write_stderr(" kind4=");discard io::write_stderr_usize(r.kind4);discard io::write_stderr(" args=");discard io::write_stderr_usize(r.ordered_arguments);discard io::write_stderr(" helpers=");discard io::write_stderr_usize(r.helpers);discard io::write_stderr(" captures=");discard io::write_stderr_usize(r.captures);discard io::write_stderr("\n");
}
@effects(console) pub fn void verify(ref u8[] name,ref u8[] raw,bool accept,usize constructs,usize invocations,usize captures){
    // Capacity/allocation/stale-record failures are not language refusals.
    let r=checked(raw);let accepted=r.phase==0 && r.code==0;let valid=if(accept){accepted && r.typed23==constructs && r.kind4==invocations && r.helpers==constructs && r.captures==captures}else{!accepted && r.phase>=2 && r.phase<=7 && r.code!=0 && r.code!=10 && r.code!=11 && r.code!=12 && r.start<=r.end && r.end<=raw.len};
    if(!valid){failure(name,raw,r);assert(false);return;}
    discard io::write("closure-unit-values case=");discard io::write_bytes(name);discard io::write(" phase=");discard io::write_u32(r.phase);discard io::write(" code=");discard io::write_u32(r.code);discard io::write(" span=");discard io::write_usize(r.start);discard io::write("..");discard io::write_usize(r.end);discard io::write(" typed23=");discard io::write_usize(r.typed23);discard io::write(" kind4=");discard io::write_usize(r.kind4);discard io::write(" args=");discard io::write_usize(r.ordered_arguments);discard io::write(" helpers=");discard io::write_usize(r.helpers);discard io::write(" captures=");discard io::write_usize(r.captures);discard io::write(" passes=");discard io::write_usize(r.passes);discard io::write("\n");
}
