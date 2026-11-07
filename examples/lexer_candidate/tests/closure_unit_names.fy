module tests::closure_unit_names;
import module_unit_ordinary_names as ordinary;
import module_unit_name_model as names;
import module_unit_name_walk as walk;
import module_unit_body_ast as ast;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_signatures as signatures;
import copied_closure_tables as private;
import syntax as syntax;
import scanner as scanner;
import body_state as body;
import bodies as entries;
import types as types;
import records as records;
import attributes as attributes;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn v::Vec<syntax::Parsed> sources(ref u8[] raw){var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut out,parsed(raw)).is_ok);assert(v::push<syntax::Parsed>(ref mut out,parsed(b"module provider;pub fn i32 action(i32 x){return x;}")).is_ok);return out;}
fn signatures::Program metadata(ref v::Vec<syntax::Parsed> sources){return switch(signatures::build(sources)){case .ok(p):{p}case .err(_):{assert(false);metadata(sources)}};}
fn ordinary::Plan resolved(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return resolved(sources,metadata);}};let imported=switch(interfaces::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return resolved(sources,metadata);}};return switch(ordinary::build(sources,metadata,ref bound,ref imported)){case .ok(p):{p}case .err(_):{assert(false);resolved(sources,metadata)}};}

// All successful arenas and private tables come from the original parser and
// preparation, then the actual unit binding/interface/ordinary-name owners.
fn void admitted(ref u8[] raw,usize captures){
    let sources=sources(raw);let original=v::at<syntax::Parsed>(ref sources,0);assert(syntax::closure_prepared(original));let metadata=metadata(ref sources);let plan=resolved(ref sources,ref metadata);assert(ordinary::count(ref plan)==4);
    var usize constructions=0;var usize provider_calls=0;var usize value_calls=0;var usize snapshot_initializers=0;var usize visible_after=0;var usize f=1;
    while(f<=ordinary::count(ref plan)){let function=ordinary::at(ref plan,f);let a=ordinary::body_ast(function);let r=ordinary::resolved(function);var usize id=1;
        while(id<=ast::expression_count(a)){let n=ast::expression_at(a,id);
            if(n.kind==36){constructions+=1;let helper=ast::expression_at(a,n.left);let target=helper.token-private::helper_flag();assert(syntax::closure_value_captures(original,target)==captures+1);let d=syntax::declaration_at(original,target-1);assert(d.start==n.start && d.end==n.end && !d.public);
                let use=names::use_at(r,n.left);let reference=names::reference_at(r,use);assert(use!=0 && reference.kind==2 && reference.module_index==0 && reference.declaration==target && reference.declaration_kind==3 && reference.source_index==ordinary::expression_origin(function,n.left));
                var usize[4] args=[0;4];var usize count=0;var usize link=n.right;while(link!=0){assert(count<4);let row=ast::expression_at(a,link);assert(row.kind==7);args[count]=row.left;count+=1;link=row.right;}assert(count==captures);var usize index=0;var usize previous=0;
                while(count>0){count-=1;let argument=args[count];let captured=ast::expression_at(a,argument);let capture_use=names::use_at(r,argument);assert(capture_use==use+index+1);let reference=names::reference_at(r,capture_use);assert(reference.kind==1 && reference.source_index==ordinary::expression_origin(function,argument));let symbol=names::symbol_at(r,reference.symbol);assert(symbol.kind==2 && symbol.pattern==captured.token-private::snapshot_flag() && symbol.pattern>previous && symbol.entry!=0);previous=symbol.pattern;
                    let entry=ast::entry_at(a,symbol.entry-1);assert(entry.kind==102 && ast::entry_origin(a,symbol.entry)==symbol.pattern && names::entry_symbol(r,symbol.entry)==reference.symbol);let init=names::reference_at(r,names::use_at(r,entry.expression));assert(init.kind==1 && init.symbol!=reference.symbol);let value=names::symbol_at(r,init.symbol);assert(value.kind==1 && value.pattern==0 && bytes::equal(raw[value.start..value.end],raw[captured.start..captured.end]));snapshot_initializers+=1;index+=1;
                }
            }
            if(n.kind==6){let use=names::reference_at(r,names::use_at(r,n.left));let name=raw[use.start..use.end];if(bytes::equal(name,b"p::action")){assert(use.kind==2 && use.module_index==1 && use.declaration==2);provider_calls+=1;}else if(bytes::equal(name,b"action")){assert(use.kind==1 && names::symbol_at(r,use.symbol).kind==1);value_calls+=1;}}
            id+=1;
        }
        // Synthetic snapshot symbols never shadow the ordinary parameter a.
        id=1;while(id<=names::reference_count(r)){let use=names::reference_at(r,id);if(captures>0 && bytes::equal(raw[use.start..use.end],b"a") && use.kind==1){let symbol=names::symbol_at(r,use.symbol);if(symbol.kind==1 && use.start>syntax::declaration_at(original,syntax::declaration_count(original)-1).end){visible_after+=1;}}id+=1;}
        f+=1;
    }
    assert(constructions==1 && provider_calls==1 && value_calls==1 && snapshot_initializers==captures);if(captures>0){assert(visible_after==1);}
}
@test fn void closure_unit_names_original_captures_and_provider_shadowing(){
    admitted(b"module main;import provider as p;fn i32 action(i32 x){return x;}fn fn i32(i32) make(){let fn i32(i32) action=fn [] i32(i32 x){return x;};discard p::action(1);return action;}fn i32 apply(fn i32(i32) action){return action(1);}",0);
    admitted(b"module main;import provider as p;fn i32 action(i32 x){return x;}fn fn i32(i32) make(i32 a){let fn i32(i32) action=fn [i32 a] i32(i32 x){return a+x;};discard a;discard p::action(1);return action;}fn i32 apply(fn i32(i32) action){return action(1);}",1);
    admitted(b"module main;import provider as p;fn i32 action(i32 x){return x;}fn fn i32(i32) make(i32 a,i32 b){let fn i32(i32) action=fn [i32 a,i32 b] i32(i32 x){return a+b+x;};discard a;discard p::action(1);return action;}fn i32 apply(fn i32(i32) action){return action(1);}",2);
    admitted(b"module main;import provider as p;fn i32 action(i32 x){return x;}fn fn i32(i32) make(i32 a,i32 b,i32 c){let fn i32(i32) action=fn [i32 a,i32 b,i32 c] i32(i32 x){return a+b+c+x;};discard a;discard p::action(1);return action;}fn i32 apply(fn i32(i32) action){return action(1);}",3);
    admitted(b"module main;import provider as p;fn i32 action(i32 x){return x;}fn fn i32(i32) make(i32 a,i32 b,i32 c,i32 d){let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x){return a+b+c+d+x;};discard a;discard p::action(1);return action;}fn i32 apply(fn i32(i32) action){return action(1);}",4);
}

fn names::Frame frame(){return names::Frame{.kind=3,.id=1,.module_index=0,.declaration=2,.receiver_type=0,.unsafe_function=false,.sdk_vector=false};}
fn v::Vec<usize> annotations(ref ast::Ast a){var v::Vec<usize> roots=v::empty<usize>();var usize i=0;while(i<ast::type_count(a)){assert(v::push<usize>(ref mut roots,0).is_ok);i+=1;}return roots;}
// Reconstructed/mutated arenas are refusal controls only. The metadata remains
// the current original source owner; no forged table grants target authority.
fn syntax::Parsed tampered(ref syntax::Parsed p,u32 mode){
    var usize creation=0;var usize i=1;while(i<=syntax::expression_count(p)){if(syntax::expression_at(p,i).kind==36){creation=i;}i+=1;}assert(creation!=0);let n=syntax::expression_at(p,creation);let row=syntax::expression_at(p,n.right);
    var v::Vec<syntax::Declaration> declarations=v::empty<syntax::Declaration>();i=0;while(i<syntax::declaration_count(p)){assert(v::push<syntax::Declaration>(ref mut declarations,syntax::declaration_at(p,i)).is_ok);i+=1;}
    var v::Vec<body::Node> expressions=v::empty<body::Node>();i=1;while(i<=syntax::expression_count(p)){let old=syntax::expression_at(p,i);var node=body::Node{.kind=old.kind,.start=old.start,.end=old.end,.token=old.token,.left=old.left,.right=old.right};
        if(mode<=3 && i==n.left){node.token=if(mode==0){private::helper_flag()+9999}else if(mode==1){private::helper_flag()+4}else if(mode==2){private::helper_flag()+3}else{1};}
        if(i==row.left){if(mode==4){node.token=private::snapshot_flag()+9999;}else if(mode==5){node.token=1;}}
        if(mode==6 && i==creation){node.right=0;}if(mode==7 && i==n.right){node.kind=8;}
        assert(v::push<body::Node>(ref mut expressions,node).is_ok);i+=1;
    }
    var v::Vec<body::Statement> statements=v::empty<body::Statement>();i=1;while(i<=syntax::statement_count(p)){let n=syntax::statement_at(p,i);assert(v::push<body::Statement>(ref mut statements,body::Statement{.kind=n.kind,.start=n.start,.end=n.end,.expression=n.expression,.target=n.target,.first=n.first,.second=n.second,.entry=n.entry,.operator=n.operator}).is_ok);i+=1;}
    var v::Vec<entries::Entry> locals=v::empty<entries::Entry>();i=0;while(i<syntax::entry_count(p)){let e=syntax::entry_at(p,i);assert(v::push<entries::Entry>(ref mut locals,entries::Entry{.kind=e.kind,.start=e.start,.end=e.end,.name_start=e.name_start,.name_end=e.name_end,.type_start=e.type_start,.type_end=e.type_end,.type_root=e.type_root,.expression=e.expression,.id=i+1}).is_ok);i+=1;}
    var v::Vec<types::Node> nodes=v::empty<types::Node>();i=1;while(i<=syntax::type_count(p)){assert(v::push<types::Node>(ref mut nodes,syntax::type_at(p,i)).is_ok);i+=1;}
    var usize work=0;let stream=switch(syntax::copy_source(p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return tampered(p,mode);}};let table=switch(private::copy(syntax::private_names(p))){case .ok(p):{p}case .err(_):{assert(false);return tampered(p,mode);}};
    return syntax::assemble_private(stream,declarations,v::empty<syntax::Problem>(),expressions,locals,statements,nodes,v::empty<records::Field>(),v::empty<attributes::Attribute>(),v::empty<attributes::Argument>(),table);
}
fn void forged(u32 mode){
    let sources=sources(b"module main;fn fn i32(i32) make(i32 a){let fn i32(i32) action=fn [i32 a] i32(i32 x){return a+x;};return action;}fn i32 ordinary(i32 a,i32 x){return a+x;}pub fn i32 public_target(i32 a,i32 x){return a+x;}");let metadata=metadata(ref sources);let p=tampered(v::at<syntax::Parsed>(ref sources,0),mode);var usize work=0;let a=switch(ast::build(ref p,2,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize creation=0;var usize i=1;while(i<=ast::expression_count(ref a)){if(ast::expression_at(ref a,i).kind==36){creation=i;}i+=1;}assert(creation!=0);let n=ast::expression_at(ref a,creation);let at=ast::expression_at(ref a,if(mode<=3){n.left}else if(mode==6){creation}else if(mode==7){n.right}else{ast::expression_at(ref a,n.right).left});
    let roots=annotations(ref a);switch(walk::resolve(ref a,syntax::source_stream(ref p),ref metadata,frame(),ref roots)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==if(mode>=6){3}else{1} && p.module_index==0 && p.start==at.start && p.end==at.end);}};
}
@test fn void closure_unit_names_refuse_forged_helper_and_capture_operands(){var u32 mode=0;while(mode<8){forged(mode);mode+=1;}}
fn void missing_capture(){let sources=sources(b"module main;fn fn i32(i32) make(){let fn i32(i32) action=fn [i32 absent] i32(i32 x){return absent+x;};return action;}");let metadata=metadata(ref sources);var usize work=0;let a=switch(ast::build(v::at<syntax::Parsed>(ref sources,0),2,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let roots=annotations(ref a);switch(walk::resolve(ref a,syntax::source_stream(v::at<syntax::Parsed>(ref sources,0)),ref metadata,frame(),ref roots)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1 && bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,0))[p.start..p.end],b"absent"));}};}
fn void signature_refused(ref u8[] raw){let sources=sources(raw);assert(switch(signatures::build(ref sources)){case .ok(_):{false}case .err(p):{p.code!=0}});}
fn void preparation_refused(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return;}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)!=0);}case .err(code):{assert(code!=0);}};}
@test fn void closure_unit_names_keep_owner_refusals(){
    missing_capture();
    signature_refused(b"module main;fn void transport<T>(fn i32(i32) action){}");
    signature_refused(b"module main;fn void transport(Option<fn i32(i32)> action){}");
    preparation_refused(b"module main;fn fn i32(i32) make(){var fn i32(i32) action=fn [] i32(i32 x){return x;};return action;}");
    preparation_refused(b"module main;fn fn i32(i32) make(){let i32 a=1;let fn i32(i32) action=fn [ref mut i32 a] i32(i32 x){return x;};return action;}");
    preparation_refused(b"module main;fn fn i32(i32) make(){let i64 a=1;let fn i32(i32) action=fn [i64 a] i32(i32 x){return x;};return action;}");
    preparation_refused(b"module main;fn fn i32(i32) make(){let i32 a=1;let fn i32(i32) action=fn [i32 a,i32 a] i32(i32 x){return a+x;};return action;}");
    preparation_refused(b"module main;fn fn i32(i32) make(){let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;let i32 e=5;let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d,i32 e] i32(i32 x){return x;};return action;}");
}
